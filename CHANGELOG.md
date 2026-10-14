# Changelog

## Unreleased

- New resource `computesphere_cache`: a SphereCache cache (managed Valkey).
  Set `instance_type` (e.g. `sc-0.25c-1g`; a high-availability type, its id
  ending in `-ha`, adds a standby in another zone), which changes in place,
  and `eviction` (`allkeys-lru` or `noeviction`), set when the cache is
  created. The password is never stored in state.
- New resource `computesphere_queue`: a SphereQueue broker (managed
  RabbitMQ). Set `instance_type` (e.g. `sq-1c-2g-s20g`; a high-availability
  type, its id ending in `-ha`, runs three instances); it changes in place.
  The password is never stored in state.
- New data sources `computesphere_cache_instance_types` and
  `computesphere_queue_instance_types`: the size catalogs with prices and
  availability for an environment.
- `computesphere_database`, `computesphere_cache` and `computesphere_queue`
  refuse a `name`, `environment_id`, `engine` or `version` change at plan
  time (a cache's `eviction` too), rather than planning a replacement that
  would start an empty one.

## 1.2.0

- New resource `computesphere_database`: a SphereDB database (managed
  PostgreSQL). Set `instance_type` (e.g. `pg-2c-4g`, Standard 2),
  `high_availability` (`none` or `standby`) and `storage_gb`; instance type,
  high availability and storage change in place. `plan` is accepted but
  deprecated, and conflicts with `instance_type`.
- New data source `computesphere_database_instance_types`: the instance-type
  catalog with vCPU, memory, connection limit, included storage and monthly
  price for an environment's region.
- New resource `computesphere_custom_domain`: a custom domain on a
  deployment, one resource per hostname. Changing the deployment or the
  hostname replaces it.
- Built with Go 1.26.9 (the security update also shipped in 1.1.3). Building
  from source needs Go 1.26 or later.

## 1.1.3

- Security update: built with Go 1.26.9; `golang.org/x/net` v0.60.0 and
  `google.golang.org/grpc` v1.83.2. No changes to resources or data sources.

## 1.1.2

- `api_url` defaults to the production API and is normalized to `/v2`; the
  registry source in the docs is corrected.

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
