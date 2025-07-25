<p align="right">
    <a href="https://computesphere.com/"><img src="https://pepublicassets.blob.core.windows.net/public-assets/computesphere-favicon.svg" width="50px" /></a>
</p>

# ComputeSphere Terraform Provider

This is the official Terraform provider for managing resources on [ComputeSphere](https://computesphere.com).

---

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23

---

## Quickstart

1. **Install Requirements:**  
   Make sure you have Terraform and Go installed (see above).

2. **Install the Provider:**  
   See [CONTRIBUTING.md](./CONTRIBUTING.md) for instructions on building and installing the provider locally.

3. **Configure the Provider:**  
   Add the following to your Terraform configuration:
   ```hcl
   terraform {
     required_providers {
       computesphere = {
         source = "computesphere.com/computesphere/computesphere"
       }
     }
   }

   provider "computesphere" {
     api_token  = "<YOUR_API_TOKEN>"      # or set COMPUTESPHERE_API_TOKEN env variable
     account_id = "<YOUR_ACCOUNT_ID>"    # or set COMPUTESPHERE_ACCOUNT_ID env variable
   }
   ```

4. **Start Using Resources:**  
   See the [examples/resources/computesphere_team](./examples/resources/computesphere_team) for usage.

---

## Provider Configuration

You can configure the provider using variables in the provider block or via environment variables:

- `api_token` (or `COMPUTESPHERE_API_TOKEN`)
- `account_id` (or `COMPUTESPHERE_ACCOUNT_ID`)

---

## Supported Resources and Data Sources

| Type         | Name / Address                       | Provider Folder                                               | Example |
|--------------|--------------------------------------|--------------------------------------------------------------|---------|
| Resource     | computesphere_team                   | [team/resource](internal/provider/team/resource/)             | [examples/resources/computesphere_team](./examples/resources/computesphere_team) |
| Resource     | computesphere_project                | [project/resource](internal/provider/project/resource/)       | [examples/resources/computesphere_project](./examples/resources/computesphere_project) |
| Resource     | computesphere_api_token              | [api_token/resource](internal/provider/api_token/resource/)   | [examples/resources/computesphere_api_token](./examples/resources/computesphere_api_token) |
| Resource     | computesphere_environment            | [environment/resource](internal/provider/environment/resource/)| [examples/resources/computesphere_environment](./examples/resources/computesphere_environment) |
| Resource     | computesphere_alert                  | [alert/resource](internal/provider/alert/resource/)           | [examples/resources/computesphere_alert](./examples/resources/computesphere_alert) |
| Resource     | computesphere_guardrail              | [guardrail/resource](internal/provider/guardrail/resource/)   | [examples/resources/computesphere_guardrail](./examples/resources/computesphere_guardrail) |
| Resource     | computesphere_notification_setting   | [notification_setting/resource](internal/provider/notification_setting/resource/) | [examples/resources/computesphere_notification_setting](./examples/resources/computesphere_notification_setting) |
| Resource     | computesphere_plan                   | [plan/resource](internal/provider/plan/resource/)             | [examples/resources/computesphere_plan](./examples/resources/computesphere_plan) |
| Resource     | computesphere_service                | [service/resource](internal/provider/service/resource/)       | [examples/resources/computesphere_service](./examples/resources/computesphere_service) |
| Resource     | computesphere_subscription           | [subscription/resource](internal/provider/subscription/resource/) | [examples/resources/computesphere_subscription](./examples/resources/computesphere_subscription) |
| Data Source  | computesphere_team                   | [team/datasource](internal/provider/team/datasource/)         | [examples/data-sources/computesphere_team](./examples/data-sources/computesphere_team) |
| Data Source  | computesphere_teams                  | [team/datasource](internal/provider/team/datasource/)         | [examples/data-sources/computesphere_teams](./examples/data-sources/computesphere_teams) |
| Data Source  | computesphere_project                | [project/datasource](internal/provider/project/datasource/)   | [examples/data-sources/computesphere_project](./examples/data-sources/computesphere_project) |
| Data Source  | computesphere_projects               | [project/datasource](internal/provider/project/datasource/)   | [examples/data-sources/computesphere_projects](./examples/data-sources/computesphere_projects) |
| Data Source  | computesphere_environment            | [environment/datasource](internal/provider/environment/datasource/) | [examples/data-sources/computesphere_environment](./examples/data-sources/computesphere_environment) |
| Data Source  | computesphere_environments           | [environment/datasource](internal/provider/environment/datasource/) | [examples/data-sources/computesphere_environments](./examples/data-sources/computesphere_environments) |
| Data Source  | computesphere_alert                  | [alert/datasource](internal/provider/alert/datasource/)       | [examples/data-sources/computesphere_alert](./examples/data-sources/computesphere_alert) |
| Data Source  | computesphere_alerts                 | [alert/datasource](internal/provider/alert/datasource/)       | [examples/data-sources/computesphere_alerts](./examples/data-sources/computesphere_alerts) |
| Data Source  | computesphere_guardrail              | [guardrail/datasource](internal/provider/guardrail/datasource/) | [examples/data-sources/computesphere_guardrail](./examples/data-sources/computesphere_guardrail) |
| Data Source  | computesphere_guardrails             | [guardrail/datasource](internal/provider/guardrail/datasource/) | [examples/data-sources/computesphere_guardrails](./examples/data-sources/computesphere_guardrails) |
| Data Source  | computesphere_notification_setting   | [notification_setting/datasource](internal/provider/notification_setting/datasource/) | [examples/data-sources/computesphere_notification_setting](./examples/data-sources/computesphere_notification_setting) |
| Data Source  | computesphere_notification_settings  | [notification_setting/datasource](internal/provider/notification_setting/datasource/) | [examples/data-sources/computesphere_notification_settings](./examples/data-sources/computesphere_notification_settings) |
| Data Source  | computesphere_plan                   | [plan/datasource](internal/provider/plan/datasource/)         | [examples/data-sources/computesphere_plan](./examples/data-sources/computesphere_plan) |
| Data Source  | computesphere_plans                  | [plan/datasource](internal/provider/plan/datasource/)         | [examples/data-sources/computesphere_plans](./examples/data-sources/computesphere_plans) |
| Data Source  | computesphere_service                | [service/datasource](internal/provider/service/datasource/)   | [examples/data-sources/computesphere_service](./examples/data-sources/computesphere_service) |
| Data Source  | computesphere_services               | [service/datasource](internal/provider/service/datasource/)   | [examples/data-sources/computesphere_services](./examples/data-sources/computesphere_services) |
| Data Source  | computesphere_subscription           | [subscription/datasource](internal/provider/subscription/datasource/) | [examples/data-sources/computesphere_subscription](./examples/data-sources/computesphere_subscription) |
| Data Source  | computesphere_subscriptions          | [subscription/datasource](internal/provider/subscription/datasource/) | [examples/data-sources/computesphere_subscriptions](./examples/data-sources/computesphere_subscriptions) |

---

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines on how to build, test, and contribute to this provider.

<!-- Check if this is the right link to the dashboard -->
<a href="https://console.computesphere.com"> <img src="https://pepublicassets.blob.core.windows.net/public-assets/computesphere-full-logo.png" width="350px" alt="ComputeSphere Logo"> </a>

---
[Explore ComputeSphere Documentation](https://docs.computesphere.com)

**Contact Us:**  
[support@computesphere.com](mailto:support@computesphere.com)  
[Support Portal](https://support.computesphere.com/portal)

&copy; 2025 ComputeSphere LLC. All Rights Reserved.

---