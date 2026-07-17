# Changelog

## Unreleased

- `computesphere_deployment`: add public docker build args and private-image
  registry auth. New optional attributes: `build_args` (map of string, PUBLIC
  values only — baked into image layers, never secrets), `image_type`
  (`public`|`private`), `image_provider`, `image_username`, `image_password`
  (sensitive), and `image_url`. Wired into both create and update. Bumps the
  `computesphere-go` SDK to v0.3.0.

## 1.0.2

- Relicense under the Mozilla Public License 2.0 (MPL-2.0), the standard
  license used by HashiCorp and partner-tier Terraform providers. No changes
  to resources or data sources.

## 1.0.1

- Add this CHANGELOG. No functional changes to resources or data sources.

## 1.0.0

- Initial release of the ComputeSphere Terraform provider: resources and data
  sources for accounts, projects, environments, services, deployments,
  API tokens, guardrails, and related platform objects.
