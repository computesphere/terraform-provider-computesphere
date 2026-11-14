# Changelog

## Unreleased

- Data source `computesphere_region`: new read-only `outbound_ips`, the
  addresses the region's services connect out from, and
  `build_outbound_ips`, the addresses builds connect out from, both as CIDR
  ranges for allowlists. Each is null until the API publishes it.
- `computesphere_database` accepts `engine = "documentdb"`: SphereDB built on
  the open-source DocumentDB engine (where offered). It has its own
  instance types (`docdb-...`; list them with
  `computesphere_database_instance_types` and `engine = "documentdb"`), and
  services get `MONGODB_URI` instead of `DATABASE_URL`.
- `computesphere_project`: a project without a `description` applies and
  re-plans cleanly (it failed with "inconsistent result after apply"), and
  removing a description clears it.
- `computesphere_deployment` changes that the API can apply in place now
  update the deployment instead of replacing it: image, registry auth, build
  args, port, spherelet count and env/secret vars. Removing `env_vars`,
  `secret_vars` or `build_args` clears them, and a `wait_for_ready` change
  alone sends nothing. Changing the spherelet size (`plan_id`), or removing
  `image`, an `image_*` setting, `port` or the spherelet count
  (`sphere_count`), still replaces the deployment.
- `computesphere_service`: new `instance_type`, the spherelet size the
  service runs on (e.g. `sl-1c-2g`, or the aliases `FLX`, `MAX` and `PWR`).
  Changing it resizes the service in place and redeploys its deployments on
  the new size (the account needs room for them). `plan_id` is deprecated and
  can't be set together with `instance_type`. Services can now be imported.
- New data source `computesphere_spherelet_instance_types`: the spherelet
  sizes with vCPU, guaranteed vCPU, memory, monthly and annual price,
  aliases and availability, optionally for one region.
- New resource `computesphere_cache`: a SphereCache cache (managed Valkey).
  Set `instance_type` (e.g. `sc-0.25c-1g`; a high-availability type, its id
  ending in `-ha`, adds a standby in another zone), which changes in place,
  and `eviction` (`allkeys-lru` or `noeviction`), which also changes in
  place on caches created from 15 October 2026, 08:30 UTC. A cache that
  keeps its data on disk can't turn high availability on or off. Read-only `persistence` and
  `internal_host`. The password is never stored in state.
- New resource `computesphere_queue`: a SphereQueue broker (managed
  RabbitMQ). Set `instance_type` (e.g. `sq-1c-2g-s20g`; a high-availability
  type, its id ending in `-ha`, runs three instances); it changes in place
  with the same number of instances, and its disk never shrinks.
  Read-only `instances`, `storage_gb` and `internal_host`. The password is
  never stored in state.
- New data sources `computesphere_cache_instance_types` and
  `computesphere_queue_instance_types`: the size catalogs with prices and
  availability for an environment.
- Changed: `computesphere_database` refuses a `name`, `environment_id`,
  `engine` or `version` change at plan time. It used to replace the
  database, which starts an empty one. To move to a new database, add it as
  a separate resource, move the data, then remove the old one. The new
  `computesphere_cache` and `computesphere_queue` work the same way.
- `computesphere_environment`: changing `project_id` moves an environment
  that runs no services to that project in the same account, in place (it
  failed with "inconsistent result after apply"). While it runs services the
  change is refused at plan time: move it in the console or with the CLI,
  then set the new project on the environment and its services. A `region`
  change is refused at plan time: an environment can't move to another
  region.
- New resource `computesphere_environment_variables`: an environment's
  shared variables, which every deployment in it inherits (a deployment's
  own value for a key wins). It owns the whole set; an empty map removes
  them all. Import by environment id.

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
