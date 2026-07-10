# =============================================================================
# ComputeSphere Terraform Provider Makefile
# =============================================================================
#
# This Makefile provides targets for developing and testing the ComputeSphere
# Terraform provider. It handles local development setup, testing, and cleanup.
#
# USAGE:
#   make [target] [VARIABLE=value]
#
# COMMON TARGETS:
#   all              - Run complete development workflow (setup, format, test, install)
#   dev-setup        - Set up local development environment (go.mod + terraform.rc)
#   clean            - Clean up development artifacts (go.mod + terraform.tfvars)
#   test-acceptance  - Run acceptance tests with environment variables
#   inject-tfvars    - Inject API credentials into example directories
#   cleanup-tfvars   - Remove generated terraform.tfvars files
#
# ENVIRONMENT VARIABLES:
#   API_TOKEN        - Your ComputeSphere API token (default: your-api-token-here)
#   ACCOUNT_ID       - Your ComputeSphere account ID (default: your-account-id-here)
#   API_URL          - ComputeSphere API URL (default: your-api-url-here)
#
# EXAMPLES:
#   make dev-setup API_TOKEN=abc123 ACCOUNT_ID=xyz789
#   make test-acceptance API_URL=https://api.computesphere.com
#   make inject-tfvars API_TOKEN=abc123 ACCOUNT_ID=xyz789 API_URL=https://api.computesphere.com
#
# =============================================================================

# Variables (at the top for discoverability)
EXAMPLE_RESOURCE_DIRS := $(shell find examples/resources -type d -name 'computesphere_*')
EXAMPLE_DATASOURCE_DIRS := $(shell find examples/data-sources -type d -name 'computesphere_*')
EXAMPLE_DIRS := $(EXAMPLE_RESOURCE_DIRS) $(EXAMPLE_DATASOURCE_DIRS)
API_TOKEN := $(if $(API_TOKEN),$(API_TOKEN),your-api-token-here)
ACCOUNT_ID := $(if $(ACCOUNT_ID),$(ACCOUNT_ID),your-account-id-here)
API_URL := $(if $(API_URL),$(API_URL),your-api-url-here)
TFVARS_TEMPLATE := utils/terraform.tfvars.template
TERRAFORMRC_TEMPLATE := utils/terraformrc.template

# =============================================================================
# MAIN TARGETS
# =============================================================================

# Main target: run all key steps (default target)
.PHONY: all
all: setup-replace check-deps fmt tidy test-acceptance generate install
	@echo "Ran setup-replace, go fmt, go mod tidy, test-acceptance, go generate, and go install."

# =============================================================================
# DEPENDENCY MANAGEMENT
# =============================================================================

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

# =============================================================================
# DEVELOPMENT SETUP
# =============================================================================

# Add replace directives to go.mod for local development.
# Both the legacy v1 client (cli/cs) and the generated v2 client
# (computesphere-api/sdk/go) need sibling clones to resolve from
# module paths until both are published via a registry/tag.
.PHONY: setup-replace
setup-replace:
	@echo "Setting up local development environment..."
	@if ! grep -q "replace github.com/computesphere/cli/cs => ../cli/cs" go.mod; then \
		echo "Adding cli/cs replace directive to go.mod..."; \
		echo "" >> go.mod; \
		echo "replace github.com/computesphere/cli/cs => ../cli/cs" >> go.mod; \
	else \
		echo "cli/cs replace directive already exists in go.mod"; \
	fi

# Complete development environment setup
.PHONY: dev-setup
dev-setup: setup-replace setup-terraformrc

# =============================================================================
# CODE QUALITY & BUILD
# =============================================================================

# Format Go code
.PHONY: fmt
fmt:
	go fmt ./...

# Tidy Go module dependencies
.PHONY: tidy
tidy:
	go mod tidy

# Generate code from templates
.PHONY: generate
generate:
	go generate ./...

# Install the provider binary
.PHONY: install
install:
	go install ./...

# =============================================================================
# TESTING
# =============================================================================

# Run acceptance tests with ComputeSphere environment variables
.PHONY: test-acceptance
test-acceptance:
	@echo "Running acceptance tests with ComputeSphere environment variables..."
	COMPUTESPHERE_API_URL=$(API_URL) \
	COMPUTESPHERE_API_TOKEN=$(API_TOKEN) \
	COMPUTESPHERE_ACCOUNT_ID=$(ACCOUNT_ID) \
	UPDATE_RECORDINGS=true \
	go test ./internal/provider/... 

# =============================================================================
# EXAMPLE MANAGEMENT
# =============================================================================

# Inject API credentials into example terraform.tfvars files
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

# Remove generated terraform.tfvars files from examples
.PHONY: cleanup-tfvars
cleanup-tfvars:
	@find examples/resources -type f -name 'terraform.tfvars' -delete
	@find examples/data-sources -type f -name 'terraform.tfvars' -delete
	@echo "Cleaned up all generated terraform.tfvars files in examples." 

# =============================================================================
# CLEANUP
# =============================================================================

# Remove both replace directives from go.mod
.PHONY: clean-gomod
clean-gomod:
	@echo "Cleaning up go.mod..."
	@if grep -q "replace github.com/computesphere/cli/cs => ../cli/cs" go.mod; then \
		echo "Removing cli/cs replace directive from go.mod..."; \
		sed -i '' '/replace github.com\/computesphere\/cli\/cs => ..\/cli\/cs/d' go.mod; \
	fi
	@if grep -q "replace github.com/computesphere/computesphere-api/sdk/go => ../computesphere-api/sdk/go" go.mod; then \
		echo "Removing computesphere-api/sdk/go replace directive from go.mod..."; \
		sed -i '' '/replace github.com\/computesphere\/computesphere-api\/sdk\/go => ..\/computesphere-api\/sdk\/go/d' go.mod; \
	fi
	@echo "Replace directives removed from go.mod"

# Clean up all development artifacts
.PHONY: clean
clean: clean-gomod cleanup-tfvars
	@echo "Cleaned up development environment and terraform.tfvars files."

# =============================================================================
# TERRAFORM CONFIGURATION
# =============================================================================

# Set up Terraform provider development override in terraform.rc
.PHONY: setup-terraformrc
setup-terraformrc:
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