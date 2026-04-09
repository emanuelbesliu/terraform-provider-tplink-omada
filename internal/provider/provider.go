package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-tplink-omada/internal/client"
	"github.com/terraform-provider-tplink-omada/internal/resources"
)

var _ provider.Provider = &OmadaProvider{}

// OmadaProvider implements the Terraform provider for TP-Link Omada Controller.
type OmadaProvider struct{}

// OmadaProviderModel maps the provider schema to Go types.
type OmadaProviderModel struct {
	URL           types.String `tfsdk:"url"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	SkipTLSVerify types.Bool   `tfsdk:"skip_tls_verify"`
}

// New creates a new provider instance.
func New() provider.Provider {
	return &OmadaProvider{}
}

func (p *OmadaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "omada"
}

func (p *OmadaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for TP-Link Omada Controller. " +
			"Manages sites, networks, wireless SSIDs, port profiles, and device configuration. " +
			"The provider represents a connection to the controller; " +
			"individual resources specify which site they belong to via the site_id attribute.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "The base URL of the Omada Controller (e.g., https://192.168.1.1:8043). " +
					"Can also be set via OMADA_URL environment variable.",
				Optional: true,
			},
			"username": schema.StringAttribute{
				Description: "The username for the Omada Controller. " +
					"Can also be set via OMADA_USERNAME environment variable.",
				Optional: true,
			},
			"password": schema.StringAttribute{
				Description: "The password for the Omada Controller. " +
					"Can also be set via OMADA_PASSWORD environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"skip_tls_verify": schema.BoolAttribute{
				Description: "Skip TLS certificate verification. Defaults to true since Omada " +
					"controllers typically use self-signed certificates. " +
					"Can also be set via OMADA_SKIP_TLS_VERIFY environment variable.",
				Optional: true,
			},
		},
	}
}

func (p *OmadaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config OmadaProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve configuration from schema or environment variables
	url := stringValueOrEnv(config.URL, "OMADA_URL")
	username := stringValueOrEnv(config.Username, "OMADA_USERNAME")
	password := stringValueOrEnv(config.Password, "OMADA_PASSWORD")

	skipTLSVerify := true
	if !config.SkipTLSVerify.IsNull() && !config.SkipTLSVerify.IsUnknown() {
		skipTLSVerify = config.SkipTLSVerify.ValueBool()
	} else if v := os.Getenv("OMADA_SKIP_TLS_VERIFY"); v == "false" || v == "0" {
		skipTLSVerify = false
	}

	if url == "" {
		resp.Diagnostics.AddError(
			"Missing Omada URL",
			"The Omada Controller URL must be set in the provider configuration or via the OMADA_URL environment variable.",
		)
		return
	}
	if username == "" {
		resp.Diagnostics.AddError(
			"Missing Omada Username",
			"The Omada Controller username must be set in the provider configuration or via the OMADA_USERNAME environment variable.",
		)
		return
	}
	if password == "" {
		resp.Diagnostics.AddError(
			"Missing Omada Password",
			"The Omada Controller password must be set in the provider configuration or via the OMADA_PASSWORD environment variable.",
		)
		return
	}

	c, err := client.NewClient(url, username, password, skipTLSVerify)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Connect to Omada Controller",
			"An error occurred when connecting to the Omada Controller: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *OmadaProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewSiteResource,
		resources.NewNetworkResource,
		resources.NewWirelessNetworkResource,
		resources.NewPortProfileResource,
		resources.NewSiteSettingsResource,
		resources.NewWlanGroupResource,
		resources.NewDeviceAPResource,
		resources.NewDeviceSwitchResource,
		resources.NewACLRuleResource,
		resources.NewIPGroupResource,
		resources.NewMDNSReflectorResource,
		resources.NewSAMLIdPResource,
		resources.NewSAMLRoleResource,
		resources.NewControllerCertificateResource,
	}
}

func (p *OmadaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		resources.NewNetworksDataSource,
		resources.NewWirelessNetworksDataSource,
		resources.NewPortProfilesDataSource,
		resources.NewSitesDataSource,
		resources.NewSiteSettingsDataSource,
		resources.NewDevicesDataSource,
		resources.NewFirewallACLsDataSource,
		resources.NewIPGroupsDataSource,
		resources.NewMDNSReflectorsDataSource,
	}
}

func stringValueOrEnv(val types.String, envKey string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	return os.Getenv(envKey)
}
