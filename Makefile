.DEFAULT_GOAL: build

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


########
# TEST #
########

.PHONY: test
test: fmt
test: vet
test: ## Run go test against code
	@echo Running tests... >&2
	@go test ./... -race