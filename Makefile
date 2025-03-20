TOOLS_DIR                          ?= $(PWD)/.tools
HELM_DOCS                          ?= $(TOOLS_DIR)/helm-docs
HELM_DOCS_VERSION                  ?= v1.11.0
TOOLS                              := $(HELM_DOCS)

$(HELM_DOCS):
	@echo Install helm-docs... >&2
	@GOBIN=$(TOOLS_DIR) go install github.com/norwoodj/helm-docs/cmd/helm-docs@$(HELM_DOCS_VERSION)

.PHONY: install-tools
install-tools: $(TOOLS) ## Install tools

.PHONY: clean-tools
clean-tools: ## Remove installed tools
	@echo Clean tools... >&2
	@rm -rf $(TOOLS_DIR)

.PHONY: codegen-helm-docs
codegen-helm-docs: ## Generate helm docs
	@echo Generate helm docs... >&2
	@docker run -v ${PWD}/charts:/work -w /work jnorwood/helm-docs:v1.11.0 -s file

# .PHONY: verify-helm
# verify-helm: ## Check Helm charts are up to date
# verify-helm: codegen-helm-docs
# 	@echo Checking helm charts are up to date... >&2
# 	@git --no-pager diff charts
# 	@echo 'If this test fails, it is because the git diff is non-empty after running "make codegen-helm-docs".' >&2
# 	@echo 'To correct this, locally run "make codegen-helm-docs", commit the changes, and re-run tests.' >&2
# 	@git diff --quiet --exit-code charts