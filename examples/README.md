# ComputeSphere Terraform Examples

This directory contains example configurations for using the ComputeSphere Terraform provider. Each subdirectory demonstrates how to use a specific resource or data source, following best practices for security, reusability, and clarity.

## Structure

- `resources/` — Example configurations for ComputeSphere resources (e.g., `computesphere_project`, `computesphere_team`, etc.).
- `data-sources/` — Example configurations for ComputeSphere data sources (e.g., `computesphere_project`, `computesphere_team`, etc.).

Each example typically includes:
- `resource.tf` or `data-source.tf`: The main example configuration.
- `terraform.tfvars`: Example variable values for local testing (should not be committed with real secrets).
- `import-by-identity.tf`, `import-by-string-id.tf`, `import.sh`: Examples for importing resources into Terraform state.

## Usage

1. **Copy the example directory** for the resource or data source you want to use.
2. **Edit the variable values** in `terraform.tfvars` or directly in the `.tf` files as needed.
3. **Initialize Terraform:**
   ```sh
   terraform init
   ```
4. **Apply the configuration:**
   ```sh
   terraform apply
   ```
5. **Import existing resources** using the provided import examples if needed.

## Best Practices

- **Never commit real secrets** (API tokens, account IDs, etc.) to version control. Use environment variables or `terraform.tfvars` (which should be git-ignored).
- **Use variables** for all sensitive or environment-specific values.
- **Review and customize** the examples to fit your actual infrastructure and security requirements.

## Additional Resources

- [ComputeSphere Provider Documentation](https://registry.terraform.io/providers/computesphere/computesphere/latest/docs)
- [Terraform Documentation](https://developer.hashicorp.com/terraform/docs)

---

If you have questions or need more advanced examples, please open an issue or consult the provider documentation. 