package k8scontroller

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	customlogger "github.com/VedRatan/k8swatchdog/logger"
	"github.com/VedRatan/remediation-server/ai"
	"github.com/VedRatan/remediation-server/handlers"
	"github.com/VedRatan/remediation-server/k8s"
	"github.com/VedRatan/remediation-server/types"
	k8sgptv1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	jsonApiMachinery "k8s.io/apimachinery/pkg/runtime/serializer/json"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	factory   dynamicinformer.DynamicSharedInformerFactory
	resultGVR = schema.GroupVersionResource{
		Group:    "core.k8sgpt.ai",
		Version:  "v1alpha1",
		Resource: "results",
	}
	extraprompt = "Generate a remediated Kubernetes Pod YAML manifest for above faulty Pod. Generate a valid pod YAML with no extra fields, don't change the metadata of the pod. Ensure the YAML is valid, properly formatted, and does not include any unnecessary fields, comments, or text explanations."
)

type controller struct {
	clientset         client.Client
	resLister         cache.GenericLister
	queue             workqueue.TypedRateLimitingInterface[any]
	wg                wait.Group
	aiClient          ai.AIClient
	Informer          cache.SharedIndexInformer
	eventRegistration cache.ResourceEventHandlerRegistration
	logger            *zap.Logger
}

func K8sGptResultInformer() cache.SharedIndexInformer {
	informer := factory.ForResource(resultGVR).Informer()
	return informer
}

func K8sGptLister() cache.GenericLister {
	lister := factory.ForResource(resultGVR).Lister()
	return lister
}

func NewController(client client.Client) *controller {
	factory = dynamicinformer.NewDynamicSharedInformerFactory(k8s.NewDynamicClient(), time.Minute)
	resInformer := K8sGptResultInformer()
	resLister := K8sGptLister()

	aiClient, err := ai.GetAiClient(types.AiAgent)
	if err != nil {
		fmt.Printf("failed to get AI client: %v", err)
		os.Exit(1)
	}
	c := &controller{
		clientset: client,
		resLister: resLister,
		Informer:  resInformer,
		wg:        wait.Group{},
		aiClient:  aiClient,
		queue:     workqueue.NewTypedRateLimitingQueue[any](workqueue.DefaultTypedControllerRateLimiter[any]()),
	}

	eventRegistration, err := resInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDel,
		},
	)
	if err != nil {
		fmt.Printf("error in registering event handler: %v", err)
		os.Exit(1)
	}

	logger, err := customlogger.NewLogger("remediation-server")
	if err != nil {
		fmt.Println("Error initializing logger:", err)
		os.Exit(1)
	}

	c.eventRegistration = eventRegistration
	c.logger = logger

	return c
}

func (c *controller) Start(ctx context.Context) {
	if c.resLister == nil {
		return
	}
	c.wg.StartWithContext(ctx, func(ctx context.Context) {
		defer c.logger.Info("worker stopped")
		c.logger.Info("worker starting ....")
		wait.UntilWithContext(ctx, c.worker, 1*time.Second)
	})
}

func (c *controller) Stop() {
	defer c.logger.Info("queue stopped")
	defer c.wg.Wait()
	defer c.logger.Sync()
	// Unregister the event handlers
	c.UnregisterEventHandlers()
	c.logger.Info("queue stopping ....")
	c.queue.ShutDown()
}

func (c *controller) UnregisterEventHandlers() {
	if err := c.Informer.RemoveEventHandler(c.eventRegistration); err != nil {
		c.logger.Error("error removing event handlers:", zap.Error(err))
		return
	}
	c.logger.Info("unregister event handlers")
}

func (c *controller) worker(ctx context.Context) {
	for c.processItem(ctx) {
	}
}

func (c *controller) processItem(ctx context.Context) bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Forget(item)
	err := c.reconcile(ctx, item)
	if err != nil {
		c.logger.Info("reconciliation failed", zap.Error(err))
		c.queue.Done(item)
		c.queue.AddRateLimited(item)
		return true
	}

	c.queue.Done(item)
	return true
}

func (c *controller) reconcile(ctx context.Context, item any) error {
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		c.logger.Error("error getting key from cache", zap.Error(err))
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	err = c.createRemediationRequest(ns, name)
	if err != nil {
		c.logger.Error("error creating remediation request to k8s-agent", zap.Error(err), zap.String("name", name), zap.String("namespace", ns))
		return err
	}
	return nil
}

func (c *controller) createRemediationRequest(ns string, name string) error {
	ctx := context.Background()
	var result k8sgptv1alpha1.Result

	// Fetch and process the object
	obj, err := c.resLister.ByNamespace(ns).Get(name)
	if err != nil {
		c.logger.Error("error getting result obj", zap.Error(err), zap.String("name", name), zap.String("namespace", ns))
		return err
	}

	unstructureObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		c.logger.Error("failed to convert runtime.Object to *unstructured.Unstructured", zap.Error(err), zap.String("name", name), zap.String("namespace", ns))
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructureObj.Object, &result); err != nil {
		c.logger.Error("failed to convert unstructured obj to *k8sgptv1alpha1.Result", zap.Error(err), zap.String("name", name), zap.String("namespace", ns))
		return err
	}

	prompt := result.Spec.Details
	nsName := result.Spec.Name
	podNs, podName, err := cache.SplitMetaNamespaceKey(nsName)
	if err != nil {
		c.logger.Error("error splitting key into namespace and name", zap.Error(err))
		return err
	}

	var pod corev1.Pod
	if err := c.clientset.Get(ctx, apitypes.NamespacedName{Namespace: podNs, Name: podName}, &pod); err != nil {
		c.logger.Error("failed to get pod", zap.Error(err))
		return err
	}

	c.logger.Info("fetched the faulty pod", zap.String("name", nsName))
	// Convert the Pod object to YAML
	serializer := jsonApiMachinery.NewSerializerWithOptions(jsonApiMachinery.DefaultMetaFactory, nil, nil, jsonApiMachinery.SerializerOptions{Yaml: true})
	var podYAML bytes.Buffer
	if err := serializer.Encode(&pod, &podYAML); err != nil {
		c.logger.Error("failed to encode pod to YAML", zap.Error(err))
	}

	// Construct the prompt for the AI agent
	aiPrompt := fmt.Sprintf("%s\n\nPod YAML:\n%s\n\n%s", prompt, podYAML.String(), extraprompt)

	// Call the AI client to generate content
	remediatedYAML, err := c.aiClient.GenerateContent(ctx, aiPrompt)
	if err != nil {
		c.logger.Error("failed to generate content from AI agent", zap.Error(err))
		return err
	}

	c.logger.Info("got the remediation, remediating faulty pod...", zap.String("pod", nsName))

	// Forward the remediation
	if err := handlers.ForwardRemediation(remediatedYAML); err != nil {
		c.logger.Error("failed to forward remediation to k8s-agent", zap.Error(err))
		return err
	}

	c.logger.Info("remediated faulty pod", zap.String("pod", nsName))
	return nil
}

func (c *controller) handleAdd(obj interface{}) {
	c.queue.Add(obj)
}

func (c *controller) handleDel(obj interface{}) {
	c.logger.Info("delete was called")
}
