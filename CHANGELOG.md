# Changelog

## Unreleased

- New resource `computesphere_database`: a SphereDB database (managed
  PostgreSQL). Set `instance_type` (e.g. `pg-2c-4g`, Standard 2),
  `high_availability` (`none`, `standby`, `three_zone`) and `storage_gb`;
  instance type, high availability and storage change in place. `plan` is
  accepted but deprecated, and conflicts with `instance_type`.
- New data source `computesphere_database_instance_types`: the instance-type
  catalog with vCPU, memory, connection limit, included storage and monthly
  price for an environment's region.
- New resources, one per compute type: `computesphere_web_service`,
  `computesphere_private_service`, `computesphere_worker`,
  `computesphere_cron_job` and `computesphere_static_site`. Each takes only
  its type's fields and sends the API's closed create and change bodies; a
  field the API can set only at create forces a new resource (the
  pre-deploy command, and an image worker's command). The plan always comes
  from the service. `git_repo_id` (needed for GitLab sources) is on every
  type; it's sent at create and read back. Setting it on a service that has
  none only records it, removing it from the configuration keeps the
  recorded value, and a value that differs from the service's is refused at
  plan time.
- `image_password` on the per-type resources is write-only (Terraform 1.11
  or later): never stored in state. Increase `image_password_wo_version` to
  rotate it in place on every type. A first version on an imported or moved
  resource only records it.
- `computesphere_deployment` is deprecated, and so is its `plan_id`. A
  `moved` block carries a deployment over to the resource for its type
  without redeploying it (Terraform 1.8 or later).
- Needs the `computesphere-go` SDK release that carries the SphereDB
  operations (newer than v0.3.0).

## 1.1.1

- Re-release of the build_args + private-image auth change under a clean
  version (1.1.0 was withdrawn). No functional difference from the withdrawn
  1.1.0.

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
