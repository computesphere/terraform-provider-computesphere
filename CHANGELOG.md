# Changelog

## Unreleased

### New resources

- New resources, one per compute type: `computesphere_web_service`,
  `computesphere_private_service`, `computesphere_worker`,
  `computesphere_cron_job` and `computesphere_static_site`. Each takes only
  its type's fields. A field the API can set only at create forces a new
  resource (the pre-deploy command); everything else changes in place, and
  an apply rolls the service out once, however many settings it changes.
  A git-built service rolls out through a new build when its build settings
  change; its first build starts when it's created.
  - Set `service_id` to run an existing service, or `name` to create a new
    service with the resource. A private service can only be created with
    `name`. Deleting a resource also deletes its service when it's the
    service's last deployment, with `service_id` too; set
    `create_before_destroy` on a `service_id` resource whose changes replace
    it.
  - `env_vars` and `secret_vars`, once set, own the whole set: a variable
    set elsewhere and not listed is removed.
  - `instance_type` (all but static sites): the spherelet size, e.g.
    `sl-1c-2g` or `FLX`. It defaults to the service's size and changes in
    place, except on a cron job, where a new size creates a new cron job.
  - `git_repo_id` (needed for GitLab sources) is sent at create and read
    back. Setting it on a service that has none only records it, removing
    it from the configuration keeps the recorded value, and a value that
    differs from the service's is refused at plan time.
  - `image_password` is write-only (Terraform 1.11 or later): never stored
    in state. Increase `image_password_wo_version` to rotate it in place; it
    is also sent with every image change, since the API keeps a stored
    password only for the same registry.
  - A cron job needs `schedule` and `command`, and a static site `git_url`
    and `build_command`.
  - Removing `network_access` opens the service to everyone again.
    `default_url_disabled` can't be set when the service is created: add
    its custom domain first.
  - Warnings the API returns (for example a size it picked, or a plain
    variable that looks like a credential) show as Terraform warnings.
- Switching a git-built web service, private service, worker or cron job to
  an image in place is refused at plan time for now. Create a new resource
  for the image instead.

### Deprecated

- `computesphere_deployment`. A `moved` block carries a deployment over to
  the resource for its type (Terraform 1.8 or later); its size carries over
  as `instance_type`. Keep `service_id` in the new block. The move itself
  doesn't redeploy; settings the old resource didn't track (for example a
  health check, or a cron job's schedule) apply as one in-place update,
  which rolls the service out.

### Upgrade notes

- Destroying a deployment also deletes its service when it's the
  service's last deployment. That's how the API works, and it's the same
  for `computesphere_deployment` and for the new resources with
  `service_id`. A `computesphere_service` managed beside them then shows
  as missing and is created again on the next apply. To replace a
  deployment without losing its service, set
  `lifecycle { create_before_destroy = true }`.
- With `wait_for_ready`, an apply that builds a git-built service waits
  for the build to finish, then for the rollout. A failed build fails the
  apply.

## 1.4.0

### New resources and data sources

- New resource `computesphere_secret_source`: a secret manager connection
  for the account (for example Doppler), which deployments and environments
  read secrets from. Its `credential` is write-only, never stored in state;
  change `credential_version` to send a rotated one. `all_projects` or
  `project_ids` say which projects may use it. Needs Terraform 1.11 or later.
- New data sources `computesphere_cache_connection` and
  `computesphere_queue_connection`: host, port, user (and a queue's virtual
  host), a TLS URI and the CA file. The password is masked unless
  `reveal = true`, which stores it in state and reveals it (recorded in the
  audit log) on every plan and apply.

### New attributes

- `computesphere_deployment`: new `instance_type`, the spherelet size the
  deployment runs on (e.g. `sl-1c-2g`, or `FLX`, `MAX`, `PWR`); it defaults
  to the service's size and changes in place. `plan_id` is deprecated and
  can't be set together with `instance_type`.
- `computesphere_cache`: new `injection` block: which services in the
  environment get the cache's URL as a variable (`mode` `all`, `selected`
  with `service_ids`, or `none`) and its `variable_name` (default
  `REDIS_URL`). Changing it restarts the services it affects; leaving it
  out keeps it as it is.
- `computesphere_alert`: `threshold` and `evaluation_period` are optional,
  for the scheduled-job alert types (`job_run_failed`, `job_failing_streak`,
  `job_run_missed`) that don't use them. New read-only
  `effective_evaluation_period` and `status`.

### Changed

- `computesphere_service` and `computesphere_deployment`: set
  `instance_type` on the service or on its deployments, not both: a service
  size change resizes every deployment of the service.

## 1.3.0

### New resources and data sources

- New resource `computesphere_cache`: a SphereCache cache (managed Valkey).
  Set `instance_type` (e.g. `sc-0.25c-1g`; a high-availability type, its id
  ending in `-ha`, adds a standby in another zone), which changes in place,
  and `eviction` (`allkeys-lru` or `noeviction`), which also changes in
  place on caches created from 15 October 2026, 08:30 UTC. A cache that
  keeps its data on disk can't turn high availability on or off. Read-only
  `persistence` and `internal_host`. The password is never stored in state.
- New resource `computesphere_queue`: a SphereQueue broker (managed
  RabbitMQ). Set `instance_type` (e.g. `sq-1c-2g-s20g`; a high-availability
  type, its id ending in `-ha`, runs three instances); it changes in place
  with the same number of instances, and its disk never shrinks.
  Read-only `instances`, `storage_gb` and `internal_host`. The password is
  never stored in state.
- New data sources `computesphere_cache_instance_types` and
  `computesphere_queue_instance_types`: the size catalogs with prices and
  availability for an environment.
- New resource `computesphere_environment_variables`: an environment's
  shared variables, which every deployment in it inherits (a deployment's
  own value for a key wins). It owns the whole set; an empty map removes
  them all. Import by environment id.
- New resource `computesphere_environment_secrets`: an environment's shared
  secrets, which every deployment in it inherits. Values are write-only, so
  a value changed outside Terraform isn't detected; unchanged values aren't
  sent again. It owns the set of names; an empty map removes them all.
  Import by environment id.
- New data source `computesphere_database_connection`: the pooled and
  direct routes of a database (host, port, database, user, URI), the public
  routes when public access is on, and the CA details. The password is
  masked unless `reveal = true`, which stores it in state and reveals it
  (recorded in the activity log) on every plan and apply.
- New data source `computesphere_spherelet_instance_types`: the spherelet
  sizes with vCPU, guaranteed vCPU, memory, monthly and annual price,
  aliases and availability, optionally for one region.

### New attributes

- `computesphere_service`: new `instance_type`, the spherelet size the
  service runs on (e.g. `sl-1c-2g`, or the aliases `FLX`, `MAX` and `PWR`).
  Changing it resizes the service in place and redeploys its deployments on
  the new size (the account needs room for them). `plan_id` is deprecated and
  can't be set together with `instance_type`. Services can now be imported.
- `computesphere_database` accepts `engine = "documentdb"`: SphereDB built on
  the open-source DocumentDB engine (where offered). It has its own
  instance types (`docdb-...`; list them with
  `computesphere_database_instance_types` and `engine = "documentdb"`), and
  services get `MONGODB_URI` instead of `DATABASE_URL`.
- `computesphere_environment`: new `deploy_policy` (`auto` or
  `approval`), `default_url_disabled` and `ip_allowlist` (IP addresses or
  CIDR ranges; an empty list makes the environment public), changed in
  place, and read-only
  `network_policy_enabled`, `sub_domain` and `kind`. Leaving a setting out
  keeps it as it is.
- `computesphere_notification_setting`: new `webhooks` (URL, body format and
  token). Left out, the webhooks and email addresses set in the console are
  kept; before, every apply removed the webhooks. Leaving out `emails` no
  longer fails. The resource and its data source need a full-access API
  token, and say so when a scoped token is refused.
- Data source `computesphere_region`: new read-only `outbound_ips`, the
  addresses the region's services connect out from, and
  `build_outbound_ips`, the addresses builds connect out from, both as CIDR
  ranges for allowlists. Each is null until the API publishes it.

### Changed

- Changed: `computesphere_database` refuses a `name`, `environment_id`,
  `engine` or `version` change at plan time. It used to replace the
  database, which starts an empty one. To move to a new database, add it as
  a separate resource, move the data, then remove the old one. The new
  `computesphere_cache` and `computesphere_queue` work the same way.
- `computesphere_deployment` changes that the API can apply in place now
  update the deployment instead of replacing it: image, registry auth, build
  args, port, spherelet count and env/secret vars. Removing `env_vars`,
  `secret_vars` or `build_args` clears them, and a `wait_for_ready` change
  alone sends nothing. Changing the spherelet size (`plan_id`), or removing
  `image`, an `image_*` setting, `port` or the spherelet count
  (`sphere_count`), still replaces the deployment.

### Fixed

- `computesphere_project`: a project without a `description` applies and
  re-plans cleanly (it failed with "inconsistent result after apply"), and
  removing a description clears it.
- `computesphere_environment`: changing `project_id` moves an environment
  that runs no services to that project in the same account, in place (it
  failed with "inconsistent result after apply"). While it runs services the
  change is refused at plan time: move it in the console or with the CLI,
  then set the new project on the environment and its services. A `region`
  change is refused at plan time: an environment can't move to another
  region.
- `computesphere_deployment`: deployment variable writes replace the whole
  set, so a variable removed from the configuration is removed.

### Deprecated

- Deprecated: the `computesphere_guardrail` resource and data source.
  Guardrails are no longer part of the public API: set them in the console.
  Both will stop working and be removed in a future release.

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
