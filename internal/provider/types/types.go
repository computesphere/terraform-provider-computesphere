package types

import (
	"fmt"

	csv2 "github.com/computesphere/computesphere-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Data is the provider-wide configuration handed to every resource and
// datasource via terraform-plugin-framework's ConfigureRequest.ProviderData.
type Data struct {
	// V2Client is the generated public v2 API client
	// (github.com/computesphere/computesphere-go), used by every resource and
	// datasource.
	V2Client *csv2.ClientWithResponses

	// AccountID is the active account UUID, required on every v2 request
	// via the x-account-id header. Sourced from the `account_id` provider
	// attribute or the COMPUTESPHERE_ACCOUNT_ID environment variable.
	AccountID string
}

// ConfigureDatasource extracts provider data for datasources, with type assertion and error handling.
func ConfigureDatasource(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *Data {
	if req.ProviderData == nil {
		return nil
	}
	d, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *types.Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}
	return d
}

// ConfigureResource extracts provider data for resources, with type assertion and error handling.
func ConfigureResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *Data {
	if req.ProviderData == nil {
		return nil
	}
	d, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *types.Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}
	return d
}
