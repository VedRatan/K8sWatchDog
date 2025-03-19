# Tutorial to try out functionality of K8sWatchDog

- Step 1: Install the results crd of k8sgpt operator.
    ```console
    kubectl apply -f https://raw.githubusercontent.com/k8sgpt-ai/k8sgpt-operator/refs/heads/main/config/crd/bases/core.k8sgpt.ai_resu
    lts.yaml
    ```
- Step 2: Apply sample faulty pod
    ```console
    kubectl apply -f https://raw.githubusercontent.com/VedRatan/K8sWatchdog/refs/heads/main/manifests/faulty-pod.yaml
    ```
- Step 3: Apply the sample result CR in your local cluster
    ```console
    kubectl apply -f https://raw.githubusercontent.com/VedRatan/K8sWatchdog/refs/heads/main/manifests/result.yaml
    ```
- Step 4: Verify the status of the `faulty-pod` in the cluster, it should be in running state


 **_NOTE:_**  Remediation-server internally uses gemini ai backend to get the remediation, so it might take some time for the pods to get remediated, as ai backend can provide some flaky responses sometimes.