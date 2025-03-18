.DEFAULT_GOAL: build
IMAGE?=docker.io/lusciousmaestro/k8s-agent
IMAGE_TAG?=latest

#########
# BUILD #
#########

.PHONY: fmt
fmt: ## Run go fmt against code.
	@echo Running fmt... >&2
	@go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	@echo Running vet... >&2
	@go vet ./...

.PHONY: build
build: fmt
build: vet
build: ## Build local binary.
	@echo "Build..." >&2
	@CGO_ENABLED=0 go build -o k8s-agent -ldflags=$(LD_FLAGS)


docker: build
	@docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 --tag $(IMAGE):$(IMAGE_TAG) . --build-arg LD_FLAGS=$(LD_FLAGS) --build-arg GIT_TOKEN=$(GIT_TOKEN)

docker-build-and-push: build
	@docker buildx build --push --file ${PWD}/Dockerfile --progress plain --platform linux/amd64,linux/arm64,linux/arm/v7 --tag $(IMAGE):$(IMAGE_TAG) ../

########
# TEST #
########

.PHONY: test
test: fmt
test: vet
test: ## Run go test against code
	@echo Running tests... >&2
	@go test ./... -race