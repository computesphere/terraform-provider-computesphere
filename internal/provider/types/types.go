package types

import (
	"fmt"

	cs "github.com/computesphere/cli/cs"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type Data struct {
	Client *cs.APIClient
	// Add other provider-wide config/state fields here as needed
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
