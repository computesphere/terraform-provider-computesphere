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
2. **Create a new branch** from `develop` for your changes.
3. **Install dependencies:**
   ```sh
   go mod tidy
   ```
4. **Build the provider:**
   ```sh
   go install
   ```
5. **Set up local provider development override:**
   You can run `make dev-setup` to automate this, or do it manually:
   - Determine your GOPATH:
     ```sh
     go env GOPATH
     ```
   - Add the following block to your `~/.terraformrc` (or `%APPDATA%/terraform.rc` on Windows), replacing `/path/to/gopath` with your actual GOPATH:
     ```hcl
     provider_installation {
       dev_overrides {
         "computesphere.com/computesphere/computesphere" = "/path/to/gopath/bin"
       }
       direct {}
     }
     ```
   - This tells Terraform CLI to use your locally built provider binary.
6. **Run the Makefile for common tasks:**
   ```sh
   make
   ```

---

## Using the Makefile

This project provides a Makefile to automate common development, testing, and setup tasks. The most important targets are:

- `make` or `make all` — Runs the full workflow: dependency check, acceptance tests, formatting, tidy, code generation, and install.
- `make check-deps` — Checks for required local dependencies (Go, Terraform, sed) and prints helpful instructions if any are missing.
- `make test-acceptance` — Runs acceptance tests with the current environment variables.
- `make fmt` — Formats all Go code.
- `make tidy` — Runs `go mod tidy` to clean up dependencies.
- `make generate` — Runs code generation.
- `make install` — Installs the provider binary to your `GOPATH/bin`.
- `make inject-tfvars` — Injects secrets and API values into all example directories using the template in `utils/terraform.tfvars.template`.
- `make cleanup-tfvars` — Removes all generated `terraform.tfvars` files from example directories.
- `make dev-setup` — Sets up your local `~/.terraformrc` (or Windows equivalent) to use the local provider binary for development. If a `provider_installation` block already exists, you will be prompted to update it manually.

**Tip:** You can always run `make <target>`

## Development Workflow

### Running Tests

You can run tests in several ways:

- **Unit and Acceptance Tests:**
  ```sh
  go test ./...
  ```
  Runs all unit and acceptance tests in the project.

- **Acceptance Tests with Cassettes:**
  ```sh
  make test-acceptance
  ```
  Runs acceptance tests using HTTP cassette recordings for fast, repeatable tests. Cassettes are YAML files stored alongside test data in the `internal/provider/<resource>/resource/testdata/` and `internal/provider/<resource>/datasource/testdata/` directories.

  To update cassettes with live API calls, set the following environment variables:
  - `COMPUTESPHERE_API_URL`
  - `COMPUTESPHERE_API_TOKEN`
  - `COMPUTESPHERE_ACCOUNT_ID`
  - `UPDATE_RECORDINGS=true`

- **Run a Specific Resource Test:**
  ```sh
  go test ./internal/provider/<resource>/resource/...
  ```
  Replace `<resource>` with the resource you want to test (e.g., `project`).

**Test Configuration Details:**
- Acceptance tests are configured to use cassette files for HTTP request/response recording and replay, making tests fast and deterministic.
- When `UPDATE_RECORDINGS=true` is set, tests will make real API calls and update the cassette files.
- Test files are located in the `internal/provider/<resource>/resource/` and `internal/provider/<resource>/datasource/` directories, with supporting test data in their respective `testdata/` subdirectories.

---