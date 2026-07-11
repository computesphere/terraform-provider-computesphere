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

Full per-attribute documentation for every resource and data source lives under
[`docs/`](./docs) (generated) and on the Terraform Registry.

| Type         | Name / Address                       | Provider Folder                                               |
|--------------|--------------------------------------|--------------------------------------------------------------|
| Resource     | computesphere_project                | [project/resource](internal/provider/project/resource/)      |
| Resource     | computesphere_environment            | [environment/resource](internal/provider/environment/resource/) |
| Resource     | computesphere_service                | [service/resource](internal/provider/service/resource/)      |
| Resource     | computesphere_deployment             | [deployment/resource](internal/provider/deployment/resource/) |
| Resource     | computesphere_team                   | [team/resource](internal/provider/team/resource/)            |
| Resource     | computesphere_api_token              | [api_token/resource](internal/provider/api_token/resource/)  |
| Resource     | computesphere_alert                  | [alert/resource](internal/provider/alert/resource/)          |
| Resource     | computesphere_guardrail              | [guardrail/resource](internal/provider/guardrail/resource/)  |
| Resource     | computesphere_notification_setting   | [notification_setting/resource](internal/provider/notification_setting/resource/) |
| Data Source  | computesphere_project / _projects    | [project/datasource](internal/provider/project/datasource/)  |
| Data Source  | computesphere_environment / _environments | [environment/datasource](internal/provider/environment/datasource/) |
| Data Source  | computesphere_environment_variables  | [environment/datasource](internal/provider/environment/datasource/) |
| Data Source  | computesphere_environment_secrets    | [environment/datasource](internal/provider/environment/datasource/) |
| Data Source  | computesphere_service / _services    | [service/datasource](internal/provider/service/datasource/)  |
| Data Source  | computesphere_region / _regions      | [region/datasource](internal/provider/region/datasource/)    |
| Data Source  | computesphere_team / _teams          | [team/datasource](internal/provider/team/datasource/)        |
| Data Source  | computesphere_alert / _alerts        | [alert/datasource](internal/provider/alert/datasource/)      |
| Data Source  | computesphere_guardrail              | [guardrail/datasource](internal/provider/guardrail/datasource/) |
| Data Source  | computesphere_notification_setting   | [notification_setting/datasource](internal/provider/notification_setting/datasource/) |
| Data Source  | computesphere_plan / _plans          | [plan/datasource](internal/provider/plan/datasource/)        |
| Data Source  | computesphere_subscription / _subscriptions | [subscription/datasource](internal/provider/subscription/datasource/) |

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