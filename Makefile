BINARY_NAME ?= helmtemplate-generator

##@ Tools

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN): ## Ensure that the directory exists
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOVULNCHECK ?= go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

## Tool Versions
GOLANGCI_LINT_VERSION ?= v2.5.0
GOVULNCHECK_VERSION ?= latest

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: build
build:
	go build -o $(BINARY_NAME) .

.PHONY: test
test:
	go test -race ./...

.PHONY: lint
lint: golangci-lint ## Run linter.
	$(GOLANGCI_LINT) run --config=.golangci.yaml

.PHONY: lint-fix
lint-fix: golangci-lint ## Run linter.
	$(GOLANGCI_LINT) run --fix --config=.golangci.yaml

.PHONY: vulncheck
vulncheck:
	@$(GOVULNCHECK) ./...

.PHONY: check
check: lint vulncheck

.PHONY: fmt
fmt:
	@$(GOLANGCI_LINT) fmt --config=.golangci.yaml
	go fmt ./...

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
