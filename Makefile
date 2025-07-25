# Variables (at the top for discoverability)
EXAMPLE_RESOURCE_DIRS := $(shell find examples/resources -type d -name 'computesphere_*')
EXAMPLE_DATASOURCE_DIRS := $(shell find examples/data-sources -type d -name 'computesphere_*')
EXAMPLE_DIRS := $(EXAMPLE_RESOURCE_DIRS) $(EXAMPLE_DATASOURCE_DIRS)
API_TOKEN := $(if $(API_TOKEN),$(API_TOKEN),your-api-token-here)
ACCOUNT_ID := $(if $(ACCOUNT_ID),$(ACCOUNT_ID),your-account-id-here)
API_URL := $(if $(API_URL),$(API_URL),your-api-url-here)
TFVARS_TEMPLATE := utils/terraform.tfvars.template
TERRAFORMRC_TEMPLATE := utils/terraformrc.template

# Main target: run all key steps (default target)
.PHONY: all
all: check-deps test-acceptance fmt tidy generate install
	@echo "Ran test-acceptance, go fmt, go mod tidy, go generate, and go install."

# Dependency check (supports macOS, Linux, and Windows)
.PHONY: check-deps
check-deps:
	@if [ "$(OS)" = "Windows_NT" ]; then \
	  which go.exe >/dev/null 2>&1 || { echo >&2 "Go is not installed. Please install Go for Windows. See CONTRIBUTING.md#local-development-requirements"; exit 1; }; \
	  which terraform.exe >/dev/null 2>&1 || { echo >&2 "Terraform is not installed. Please install Terraform for Windows. See CONTRIBUTING.md#local-development-requirements"; exit 1; }; \
	  which sed.exe >/dev/null 2>&1 || { echo >&2 "sed is not installed. Please install sed for Windows (e.g., via Git Bash or GnuWin32). See CONTRIBUTING.md#local-development-requirements"; exit 1; }; \
	else \
	  command -v go >/dev/null 2>&1 || { echo >&2 "Go is not installed. Please install Go. See CONTRIBUTING.md#local-development-requirements"; exit 1; }; \
	  command -v terraform >/dev/null 2>&1 || { echo >&2 "Terraform is not installed. Please install Terraform. See CONTRIBUTING.md#local-development-requirements"; exit 1; }; \
	  command -v sed >/dev/null 2>&1 || { echo >&2 "sed is not installed. Please install sed. See CONTRIBUTING.md#local-development-requirements"; exit 1; }; \
	fi
	@echo "All required dependencies are installed."

# Testing
.PHONY: test-acceptance
# Acceptance tests
# (Set your own environment variables or override as needed)
test-acceptance:
	@echo "Running acceptance tests with ComputeSphere environment variables..."
	COMPUTESPHERE_API_URL=$(API_URL) \
	COMPUTESPHERE_API_TOKEN=$(API_TOKEN) \
	COMPUTESPHERE_ACCOUNT_ID=$(ACCOUNT_ID) \
	UPDATE_RECORDINGS=true \
	go test ./internal/provider/... 

# Formatting
.PHONY: fmt
fmt:
	go fmt ./...

# Dependency management
.PHONY: tidy
tidy:
	go mod tidy

# Code generation
.PHONY: generate
generate:
	go generate ./...

# Install the provider
.PHONY: install
install:
	go install ./...

# Example tfvars injection/cleanup
.PHONY: inject-tfvars
inject-tfvars:
	@for dir in $(EXAMPLE_DIRS); do \
	  sed \
	    -e 's|\$${API_TOKEN}|$(API_TOKEN)|g' \
	    -e 's|\$${ACCOUNT_ID}|$(ACCOUNT_ID)|g' \
	    -e 's|\$${API_URL}|$(API_URL)|g' \
	    $(TFVARS_TEMPLATE) > $$dir/terraform.tfvars; \
	  echo "Injected terraform.tfvars into $$dir"; \
	done 

.PHONY: cleanup-tfvars
cleanup-tfvars:
	@find examples/resources -type f -name 'terraform.tfvars' -delete
	@find examples/data-sources -type f -name 'terraform.tfvars' -delete
	@echo "Cleaned up all generated terraform.tfvars files in examples." 

.PHONY: dev-setup
dev-setup:
	@echo "Setting up local Terraform provider development override..."
	@GOPATH=$$(go env GOPATH); \
	if [ -z "$$GOPATH" ]; then \
	  echo "Could not determine GOPATH. Please set GOPATH and try again."; \
	  echo "See https://golang.org/doc/gopath_code.html for help."; \
	  exit 1; \
	fi; \
	TFRC_PATH=""; \
	if [ "$$OS" = "Windows_NT" ]; then \
	  TFRC_PATH="$$APPDATA/terraform.rc"; \
	else \
	  TFRC_PATH="$$HOME/.terraformrc"; \
	fi; \
	echo "Preparing provider_installation block..."; \
	TMPFILE=$$(mktemp); \
	sed 's|\$${GOPATH}|'$$GOPATH'|g' $(TERRAFORMRC_TEMPLATE) > $$TMPFILE; \
	if [ -f "$$TFRC_PATH" ] && grep -q 'provider_installation' "$$TFRC_PATH"; then \
	  echo "A provider_installation block already exists in $$TFRC_PATH."; \
	  echo "Please manually add or update the computesphere dev_overrides entry as shown below:"; \
	  echo ""; \
	  cat $$TMPFILE; \
	  echo ""; \
	  echo "No changes made to $$TFRC_PATH."; \
	else \
	  if [ -f "$$TFRC_PATH" ] && grep -q 'computesphere.com/computesphere/computesphere' "$$TFRC_PATH"; then \
	    echo "Provider installation block for computesphere already exists in $$TFRC_PATH. Skipping."; \
	  else \
	    echo "Appending provider_installation block to $$TFRC_PATH"; \
	    cat $$TMPFILE >> "$$TFRC_PATH"; \
	  fi; \
	fi; \
	rm -f $$TMPFILE; \
	echo "Done! Your Terraform CLI will now use the local provider binary from $$GOPATH/bin." 