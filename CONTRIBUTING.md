# Contributing to ComputeSphere Terraform Provider

Thank you for your interest in contributing to the ComputeSphere Terraform Provider! We welcome contributions of all kinds—bug reports, feature requests, code, documentation, and tests.

## Local Development Requirements

To build, test, and contribute to this provider, you need:
- [Go](https://golang.org/dl/) (1.23+)
- [Terraform](https://developer.hashicorp.com/terraform/downloads) (1.0+)
- `sed` (for Makefile automation)
  - **macOS/Linux:** Pre-installed
  - **Windows:** Install via [Git for Windows](https://gitforwindows.org/) (includes Git Bash and sed), [GnuWin32 sed](http://gnuwin32.sourceforge.net/packages/sed.htm), or [WSL](https://docs.microsoft.com/en-us/windows/wsl/)
- [GNU Make](https://www.gnu.org/software/make/)
  - **macOS:** Install via [Homebrew](https://brew.sh/) with `brew install make`
  - **Linux:** Usually pre-installed
  - **Windows:** Use [Git Bash](https://gitforwindows.org/) or [WSL](https://docs.microsoft.com/en-us/windows/wsl/)

---

## Getting Started

1. **Fork the repository** and clone your fork.
2. **Clone the sibling repos** next to this one — both must live at sibling paths because `go.mod` relies on `replace` directives pointing at `../`:
   ```sh
   # Expected layout
   # ~/code/github.com/computesphere/
   # ├── terraform-provider-computesphere  (this repo)
   # ├── cli                                (legacy v1 API client)
   # └── computesphere-api                  (v2 API + sdk/go)
   git clone git@github.com:computesphere/cli.git
   git clone git@github.com:computesphere/computesphere-api.git
   ```
3. **Create a new branch** from `develop` for your changes.
4. **Set up the development environment:**
   ```sh
   make dev-setup
   ```
   This automatically:
   - Adds the necessary `replace` directives to `go.mod` for local development (both `cli/cs` and `computesphere-api/sdk/go`)
   - Sets up your `~/.terraformrc` (or `%APPDATA%/terraform.rc` on Windows) to use the local provider binary
4. **Run the complete development workflow:**
   ```sh
   make all
   ```
   This runs dependency checks, formatting, testing, code generation, and installation.

**Alternative Manual Setup:**
If you prefer to set up manually instead of using `make dev-setup`:
- Add the following block to your `~/.terraformrc` (or `%APPDATA%/terraform.rc` on Windows), replacing `/path/to/gopath` with your actual GOPATH:
  ```hcl
  provider_installation {
    dev_overrides {
      "computesphere.com/computesphere/computesphere" = "/path/to/gopath/bin"
    }
    direct {}
  }
  ```

---

## Using the Makefile

This project provides a Makefile to automate development, testing, and setup tasks. Here are the key targets organized by purpose:

### Development Setup
- `make dev-setup` — Complete development environment setup (go.mod + terraform.rc)
- `make check-deps` — Verify required dependencies (Go, Terraform, sed) are installed

### Main Workflow
- `make` or `make all` — Complete development workflow: setup, format, test, generate, install
- `make fmt` — Format all Go code
- `make tidy` — Clean up Go module dependencies
- `make generate` — Generate code from templates
- `make install` — Install the provider binary to your `GOPATH/bin`

### Testing
- `make test-acceptance` — Run acceptance tests with environment variables
  ```sh
  make test-acceptance API_TOKEN=your-token ACCOUNT_ID=your-account API_URL=https://api.computesphere.com
  ```

### Example Management
- `make inject-tfvars` — Inject API credentials into example terraform.tfvars files
  ```sh
  make inject-tfvars API_TOKEN=your-token ACCOUNT_ID=your-account API_URL=https://api.computesphere.com
  ```
- `make cleanup-tfvars` — Remove generated terraform.tfvars files from examples

### Cleanup
- `make clean` — Clean up development artifacts (go.mod + terraform.tfvars)

### Environment Variables
You can pass environment variables to override defaults:
- `API_TOKEN` — Your ComputeSphere API token
- `ACCOUNT_ID` — Your ComputeSphere account ID  
- `API_URL` — ComputeSphere API URL

**Examples:**
```sh
# Set up development environment with your credentials
make dev-setup API_TOKEN=abc123 ACCOUNT_ID=xyz789

# Run tests with live API
make test-acceptance API_TOKEN=abc123 ACCOUNT_ID=xyz789 API_URL=https://api.computesphere.com

# Inject credentials into examples
make inject-tfvars API_TOKEN=abc123 ACCOUNT_ID=xyz789 API_URL=https://api.computesphere.com
```

## Development Workflow

### Running Tests

You can run tests in several ways:

- **Complete Test Suite:**
  ```sh
  go test ./...
  ```
  Runs all unit and acceptance tests in the project.

- **Acceptance Tests (Recommended):**
  ```sh
  make test-acceptance
  ```
  Runs acceptance tests using HTTP cassette recordings for fast, repeatable tests. Cassettes are YAML files stored alongside test data in the `internal/provider/<resource>/resource/testdata/` and `internal/provider/<resource>/datasource/testdata/` directories.

  To update cassettes with live API calls:
  ```sh
  make test-acceptance API_TOKEN=your-token ACCOUNT_ID=your-account API_URL=https://api.computesphere.com UPDATE_RECORDINGS=true
  ```

- **Specific Resource Tests:**
  ```sh
  go test ./internal/provider/<resource>/resource/...
  ```
  Replace `<resource>` with the resource you want to test (e.g., `project`).

**Test Configuration Details:**
- Acceptance tests use cassette files for HTTP request/response recording and replay, making tests fast and deterministic
- When `UPDATE_RECORDINGS=true` is set, tests make real API calls and update the cassette files
- Test files are located in `internal/provider/<resource>/resource/` and `internal/provider/<resource>/datasource/` directories
- Supporting test data is stored in respective `testdata/` subdirectories

### Cleanup

When you're done developing, you can clean up the development environment:

```sh
make clean
```

This removes:
- The replace directive from `go.mod`
- Generated `terraform.tfvars` files from examples

### Best Practices

1. **Always use `make dev-setup`** when starting development to ensure proper environment configuration
2. **Use `make test-acceptance`** for testing with proper environment variables
3. **Clean up after development** with `make clean` to avoid committing development artifacts
4. **Use environment variables** instead of hardcoding credentials in examples
5. **Update cassettes carefully** - only when you need to refresh test data with live API calls

---