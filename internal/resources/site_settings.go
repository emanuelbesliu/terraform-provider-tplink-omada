package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-tplink-omada/internal/client"
)

var _ resource.Resource = &SiteSettingsResource{}
var _ resource.ResourceWithImportState = &SiteSettingsResource{}

// SiteSettingsResource manages the settings for an Omada site.
// This is a singleton resource — each site has exactly one settings object.
// Create adopts the existing settings; Delete is a no-op.
type SiteSettingsResource struct {
	client *client.Client
}

// SiteSettingsResourceModel maps the Terraform schema to Go types.
type SiteSettingsResourceModel struct {
	ID types.String `tfsdk:"id"`

	// Site identity
	SiteName types.String `tfsdk:"site_name"`
	Region   types.String `tfsdk:"region"`
	TimeZone types.String `tfsdk:"timezone"`
	Scenario types.String `tfsdk:"scenario"`

	// Auto upgrade
	AutoUpgradeEnable types.Bool `tfsdk:"auto_upgrade_enable"`

	// Mesh
	MeshEnable       types.Bool `tfsdk:"mesh_enable"`
	MeshAutoFailover types.Bool `tfsdk:"mesh_auto_failover"`
	MeshDefGateway   types.Bool `tfsdk:"mesh_default_gateway"`
	MeshFullSector   types.Bool `tfsdk:"mesh_full_sector"`

	// LED
	LEDEnable types.Bool `tfsdk:"led_enable"`

	// Roaming
	FastRoamingEnable         types.Bool `tfsdk:"fast_roaming_enable"`
	AiRoamingEnable           types.Bool `tfsdk:"ai_roaming_enable"`
	DualBand11kReportEnable   types.Bool `tfsdk:"dual_band_11k_report_enable"`
	ForceDisassociationEnable types.Bool `tfsdk:"force_disassociation_enable"`
	NonStickRoamingEnable     types.Bool `tfsdk:"non_stick_roaming_enable"`
	NonPingPongRoamingEnable  types.Bool `tfsdk:"non_ping_pong_roaming_enable"`

	// Band steering
	BandSteeringEnable              types.Bool  `tfsdk:"band_steering_enable"`
	BandSteeringConnectionThreshold types.Int64 `tfsdk:"band_steering_connection_threshold"`
	BandSteeringDifferenceThreshold types.Int64 `tfsdk:"band_steering_difference_threshold"`
	BandSteeringMaxFailures         types.Int64 `tfsdk:"band_steering_max_failures"`
	BandSteeringMultiBandMode       types.Int64 `tfsdk:"band_steering_multi_band_mode"`

	// Airtime fairness
	AirtimeFairness2g types.Bool `tfsdk:"airtime_fairness_2g"`
	AirtimeFairness5g types.Bool `tfsdk:"airtime_fairness_5g"`
	AirtimeFairness6g types.Bool `tfsdk:"airtime_fairness_6g"`

	// Beacon control
	BeaconIntvMode2g types.Int64 `tfsdk:"beacon_interval_mode_2g"`
	DtimPeriod2g     types.Int64 `tfsdk:"dtim_period_2g"`
	RtsThreshold2g   types.Int64 `tfsdk:"rts_threshold_2g"`
	FragThreshold2g  types.Int64 `tfsdk:"fragmentation_threshold_2g"`
	BeaconIntvMode5g types.Int64 `tfsdk:"beacon_interval_mode_5g"`
	DtimPeriod5g     types.Int64 `tfsdk:"dtim_period_5g"`
	RtsThreshold5g   types.Int64 `tfsdk:"rts_threshold_5g"`
	FragThreshold5g  types.Int64 `tfsdk:"fragmentation_threshold_5g"`
	BeaconInterval6g types.Int64 `tfsdk:"beacon_interval_6g"`
	BeaconIntvMode6g types.Int64 `tfsdk:"beacon_interval_mode_6g"`
	DtimPeriod6g     types.Int64 `tfsdk:"dtim_period_6g"`
	RtsThreshold6g   types.Int64 `tfsdk:"rts_threshold_6g"`
	FragThreshold6g  types.Int64 `tfsdk:"fragmentation_threshold_6g"`

	// LLDP
	LLDPEnable types.Bool `tfsdk:"lldp_enable"`

	// Advanced features
	AdvancedFeatureEnable types.Bool `tfsdk:"advanced_feature_enable"`

	// Speed test
	SpeedTestEnable   types.Bool  `tfsdk:"speed_test_enable"`
	SpeedTestInterval types.Int64 `tfsdk:"speed_test_interval"`

	// Alert
	AlertEnable      types.Bool  `tfsdk:"alert_enable"`
	AlertDelayEnable types.Bool  `tfsdk:"alert_delay_enable"`
	AlertDelay       types.Int64 `tfsdk:"alert_delay"`

	// Remote log
	RemoteLogEnable        types.Bool   `tfsdk:"remote_log_enable"`
	RemoteLogServer        types.String `tfsdk:"remote_log_server"`
	RemoteLogPort          types.Int64  `tfsdk:"remote_log_port"`
	RemoteLogMoreClientLog types.Bool   `tfsdk:"remote_log_more_client_log"`

	// Device account
	DeviceAccountUsername types.String `tfsdk:"device_account_username"`
	DeviceAccountPassword types.String `tfsdk:"device_account_password"`

	// Remember device
	RememberDeviceEnable types.Bool `tfsdk:"remember_device_enable"`
}

func NewSiteSettingsResource() resource.Resource {
	return &SiteSettingsResource{}
}

func (r *SiteSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_settings"
}

func (r *SiteSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the settings for an Omada site. This is a singleton resource — " +
			"each site has exactly one settings object. Creating this resource adopts the existing " +
			"settings and applies your configuration. Destroying it removes the resource from " +
			"Terraform state without changing settings on the controller.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The site ID (used as the resource identifier).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// --- Site identity ---
			"site_name": schema.StringAttribute{
				Description: "The display name of the site.",
				Optional:    true,
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region/country for the site (e.g., 'Romania').",
				Optional:    true,
				Computed:    true,
			},
			"timezone": schema.StringAttribute{
				Description: "The timezone for the site (e.g., 'Europe/Athens').",
				Optional:    true,
				Computed:    true,
			},
			"scenario": schema.StringAttribute{
				Description: "The site scenario: 'Home', 'Office', 'Hotel', etc.",
				Optional:    true,
				Computed:    true,
			},

			// --- Auto upgrade ---
			"auto_upgrade_enable": schema.BoolAttribute{
				Description: "Enable automatic firmware upgrade for devices.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			// --- Mesh ---
			"mesh_enable": schema.BoolAttribute{
				Description: "Enable mesh networking.",
				Optional:    true,
				Computed:    true,
			},
			"mesh_auto_failover": schema.BoolAttribute{
				Description: "Enable automatic mesh failover.",
				Optional:    true,
				Computed:    true,
			},
			"mesh_default_gateway": schema.BoolAttribute{
				Description: "Enable default gateway for mesh.",
				Optional:    true,
				Computed:    true,
			},
			"mesh_full_sector": schema.BoolAttribute{
				Description: "Enable full sector for mesh.",
				Optional:    true,
				Computed:    true,
			},

			// --- LED ---
			"led_enable": schema.BoolAttribute{
				Description: "Enable AP LED lights.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			// --- Roaming ---
			"fast_roaming_enable": schema.BoolAttribute{
				Description: "Enable 802.11r Fast BSS Transition (fast roaming).",
				Optional:    true,
				Computed:    true,
			},
			"ai_roaming_enable": schema.BoolAttribute{
				Description: "Enable AI-based roaming optimization.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"dual_band_11k_report_enable": schema.BoolAttribute{
				Description: "Enable dual-band 802.11k neighbor reports.",
				Optional:    true,
				Computed:    true,
			},
			"force_disassociation_enable": schema.BoolAttribute{
				Description: "Enable forced disassociation for roaming.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"non_stick_roaming_enable": schema.BoolAttribute{
				Description: "Enable non-sticky client roaming.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"non_ping_pong_roaming_enable": schema.BoolAttribute{
				Description: "Enable non-ping-pong roaming prevention.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			// --- Band steering ---
			"band_steering_enable": schema.BoolAttribute{
				Description: "Enable band steering to push clients to 5GHz.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"band_steering_connection_threshold": schema.Int64Attribute{
				Description: "Band steering connection threshold.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
			},
			"band_steering_difference_threshold": schema.Int64Attribute{
				Description: "Band steering difference threshold.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(4),
			},
			"band_steering_max_failures": schema.Int64Attribute{
				Description: "Band steering maximum failures before giving up.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5),
			},
			"band_steering_multi_band_mode": schema.Int64Attribute{
				Description: "Multi-band steering mode: 0=disabled, 1=prefer 5GHz.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
			},

			// --- Airtime fairness ---
			"airtime_fairness_2g": schema.BoolAttribute{
				Description: "Enable airtime fairness for 2.4GHz band.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"airtime_fairness_5g": schema.BoolAttribute{
				Description: "Enable airtime fairness for 5GHz band.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"airtime_fairness_6g": schema.BoolAttribute{
				Description: "Enable airtime fairness for 6GHz band.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			// --- Beacon control ---
			"beacon_interval_mode_2g": schema.Int64Attribute{
				Description: "Beacon interval mode for 2.4GHz (0=default).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"dtim_period_2g": schema.Int64Attribute{
				Description: "DTIM period for 2.4GHz.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
			},
			"rts_threshold_2g": schema.Int64Attribute{
				Description: "RTS/CTS threshold for 2.4GHz.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2347),
			},
			"fragmentation_threshold_2g": schema.Int64Attribute{
				Description: "Fragmentation threshold for 2.4GHz.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2346),
			},
			"beacon_interval_mode_5g": schema.Int64Attribute{
				Description: "Beacon interval mode for 5GHz (0=default).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"dtim_period_5g": schema.Int64Attribute{
				Description: "DTIM period for 5GHz.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
			},
			"rts_threshold_5g": schema.Int64Attribute{
				Description: "RTS/CTS threshold for 5GHz.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2347),
			},
			"fragmentation_threshold_5g": schema.Int64Attribute{
				Description: "Fragmentation threshold for 5GHz.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2346),
			},
			"beacon_interval_6g": schema.Int64Attribute{
				Description: "Beacon interval for 6GHz (in TUs, e.g. 100).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(100),
			},
			"beacon_interval_mode_6g": schema.Int64Attribute{
				Description: "Beacon interval mode for 6GHz (0=default).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"dtim_period_6g": schema.Int64Attribute{
				Description: "DTIM period for 6GHz.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
			},
			"rts_threshold_6g": schema.Int64Attribute{
				Description: "RTS/CTS threshold for 6GHz.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2347),
			},
			"fragmentation_threshold_6g": schema.Int64Attribute{
				Description: "Fragmentation threshold for 6GHz.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2346),
			},

			// --- LLDP ---
			"lldp_enable": schema.BoolAttribute{
				Description: "Enable LLDP (Link Layer Discovery Protocol).",
				Optional:    true,
				Computed:    true,
			},

			// --- Advanced features ---
			"advanced_feature_enable": schema.BoolAttribute{
				Description: "Enable advanced features.",
				Optional:    true,
				Computed:    true,
			},

			// --- Speed test ---
			"speed_test_enable": schema.BoolAttribute{
				Description: "Enable scheduled speed tests.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"speed_test_interval": schema.Int64Attribute{
				Description: "Speed test interval in minutes.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(120),
			},

			// --- Alert ---
			"alert_enable": schema.BoolAttribute{
				Description: "Enable alert notifications.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"alert_delay_enable": schema.BoolAttribute{
				Description: "Enable alert delay (suppress alerts for a period after triggering).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"alert_delay": schema.Int64Attribute{
				Description: "Alert delay in seconds.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(60),
			},

			// --- Remote log ---
			"remote_log_enable": schema.BoolAttribute{
				Description: "Enable remote syslog logging.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"remote_log_server": schema.StringAttribute{
				Description: "Remote syslog server address.",
				Optional:    true,
				Computed:    true,
			},
			"remote_log_port": schema.Int64Attribute{
				Description: "Remote syslog server port.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(514),
			},
			"remote_log_more_client_log": schema.BoolAttribute{
				Description: "Enable additional client logging to syslog.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			// --- Device account ---
			"device_account_username": schema.StringAttribute{
				Description: "Username for SSH/management access to devices.",
				Optional:    true,
				Computed:    true,
			},
			"device_account_password": schema.StringAttribute{
				Description: "Password for SSH/management access to devices.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
			},

			// --- Remember device ---
			"remember_device_enable": schema.BoolAttribute{
				Description: "Enable remember device feature.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *SiteSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *SiteSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SiteSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the config to determine if device_account was explicitly set by the user.
	var config SiteSettingsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the settings object from the plan
	settings := planToSiteSettings(&plan)

	// Only include deviceAccount if the user explicitly configured both fields.
	if config.DeviceAccountUsername.IsNull() || config.DeviceAccountPassword.IsNull() {
		settings.DeviceAccount = nil
	}

	// PATCH the settings onto the existing site settings
	updated, err := r.client.UpdateSiteSettings(ctx, settings)
	if err != nil {
		resp.Diagnostics.AddError("Error creating site settings", err.Error())
		return
	}

	// Use the site ID as the resource ID
	plan.ID = types.StringValue(r.client.GetSiteID())
	siteSettingsToState(updated, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SiteSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SiteSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.client.GetSiteSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading site settings", err.Error())
		return
	}

	state.ID = types.StringValue(r.client.GetSiteID())
	siteSettingsToState(settings, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SiteSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SiteSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SiteSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the config to determine if device_account was explicitly set by the user.
	// The plan merges state into Optional+Computed attributes, so we can't rely on
	// plan values alone — we must check the config to know user intent.
	var config SiteSettingsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := planToSiteSettings(&plan)

	// Only include deviceAccount in the PATCH if the user explicitly configured both
	// username and password. The Omada API requires the password on every PATCH that
	// includes deviceAccount, but the GET response never returns the real password.
	if config.DeviceAccountUsername.IsNull() || config.DeviceAccountPassword.IsNull() {
		settings.DeviceAccount = nil
	}

	updated, err := r.client.UpdateSiteSettings(ctx, settings)
	if err != nil {
		resp.Diagnostics.AddError("Error updating site settings", err.Error())
		return
	}

	plan.ID = state.ID
	siteSettingsToState(updated, &plan)

	// Preserve the device account password from state — the API returns "***".
	if !state.DeviceAccountPassword.IsNull() {
		plan.DeviceAccountPassword = state.DeviceAccountPassword
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SiteSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Site settings are a singleton — they always exist.
	// Destroying this resource just removes it from Terraform state.
	// No API call needed.
}

func (r *SiteSettingsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	settings, err := r.client.GetSiteSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error importing site settings", err.Error())
		return
	}

	var state SiteSettingsResourceModel
	state.ID = types.StringValue(r.client.GetSiteID())
	siteSettingsToState(settings, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// planToSiteSettings converts the Terraform plan into a client.SiteSettings for PATCH.
func planToSiteSettings(plan *SiteSettingsResourceModel) *client.SiteSettings {
	s := &client.SiteSettings{}

	// Site identity — only include if any site fields are set
	if !plan.SiteName.IsNull() || !plan.Region.IsNull() || !plan.TimeZone.IsNull() || !plan.Scenario.IsNull() {
		s.Site = &client.SiteSettingsSite{}
		if !plan.SiteName.IsNull() && !plan.SiteName.IsUnknown() {
			s.Site.Name = plan.SiteName.ValueString()
		}
		if !plan.Region.IsNull() && !plan.Region.IsUnknown() {
			s.Site.Region = plan.Region.ValueString()
		}
		if !plan.TimeZone.IsNull() && !plan.TimeZone.IsUnknown() {
			s.Site.TimeZone = plan.TimeZone.ValueString()
		}
		if !plan.Scenario.IsNull() && !plan.Scenario.IsUnknown() {
			s.Site.Scenario = plan.Scenario.ValueString()
		}
	}

	// Auto upgrade
	s.AutoUpgrade = &client.AutoUpgrade{
		Enable: plan.AutoUpgradeEnable.ValueBool(),
	}

	// Mesh
	s.Mesh = &client.MeshSettings{
		MeshEnable:         plan.MeshEnable.ValueBool(),
		AutoFailoverEnable: plan.MeshAutoFailover.ValueBool(),
		DefGatewayEnable:   plan.MeshDefGateway.ValueBool(),
		FullSector:         plan.MeshFullSector.ValueBool(),
	}

	// LED
	s.LED = &client.LEDSettings{
		Enable: plan.LEDEnable.ValueBool(),
	}

	// Roaming
	s.Roaming = &client.RoamingSettings{
		FastRoamingEnable:         plan.FastRoamingEnable.ValueBool(),
		AiRoamingEnable:           plan.AiRoamingEnable.ValueBool(),
		DualBand11kReportEnable:   plan.DualBand11kReportEnable.ValueBool(),
		ForceDisassociationEnable: plan.ForceDisassociationEnable.ValueBool(),
		NonStickRoamingEnable:     plan.NonStickRoamingEnable.ValueBool(),
		NonPingPongRoamingEnable:  plan.NonPingPongRoamingEnable.ValueBool(),
	}

	// Band steering
	s.BandSteering = &client.BandSteering{
		Enable:              plan.BandSteeringEnable.ValueBool(),
		ConnectionThreshold: int(plan.BandSteeringConnectionThreshold.ValueInt64()),
		DifferenceThreshold: int(plan.BandSteeringDifferenceThreshold.ValueInt64()),
		MaxFailures:         int(plan.BandSteeringMaxFailures.ValueInt64()),
	}
	s.BandSteeringForMultiBand = &client.BandSteeringForMultiBand{
		Mode: int(plan.BandSteeringMultiBandMode.ValueInt64()),
	}

	// Airtime fairness
	s.AirtimeFairness = &client.AirtimeFairness{
		Enable2g: plan.AirtimeFairness2g.ValueBool(),
		Enable5g: plan.AirtimeFairness5g.ValueBool(),
		Enable6g: plan.AirtimeFairness6g.ValueBool(),
	}

	// Beacon control
	s.BeaconControl = &client.BeaconControl{
		BeaconIntvMode2g:         int(plan.BeaconIntvMode2g.ValueInt64()),
		DtimPeriod2g:             int(plan.DtimPeriod2g.ValueInt64()),
		RtsThreshold2g:           int(plan.RtsThreshold2g.ValueInt64()),
		FragmentationThreshold2g: int(plan.FragThreshold2g.ValueInt64()),
		BeaconIntvMode5g:         int(plan.BeaconIntvMode5g.ValueInt64()),
		DtimPeriod5g:             int(plan.DtimPeriod5g.ValueInt64()),
		RtsThreshold5g:           int(plan.RtsThreshold5g.ValueInt64()),
		FragmentationThreshold5g: int(plan.FragThreshold5g.ValueInt64()),
		BeaconInterval6g:         int(plan.BeaconInterval6g.ValueInt64()),
		BeaconIntvMode6g:         int(plan.BeaconIntvMode6g.ValueInt64()),
		DtimPeriod6g:             int(plan.DtimPeriod6g.ValueInt64()),
		RtsThreshold6g:           int(plan.RtsThreshold6g.ValueInt64()),
		FragmentationThreshold6g: int(plan.FragThreshold6g.ValueInt64()),
	}

	// LLDP
	s.LLDP = &client.LLDPSettings{
		Enable: plan.LLDPEnable.ValueBool(),
	}

	// Advanced features
	s.AdvancedFeature = &client.AdvancedFeature{
		Enable: plan.AdvancedFeatureEnable.ValueBool(),
	}

	// Speed test
	s.SpeedTest = &client.SpeedTest{
		Enable:   plan.SpeedTestEnable.ValueBool(),
		Interval: int(plan.SpeedTestInterval.ValueInt64()),
	}

	// Alert
	s.Alert = &client.AlertSettings{
		Enable:      plan.AlertEnable.ValueBool(),
		DelayEnable: plan.AlertDelayEnable.ValueBool(),
		Delay:       int(plan.AlertDelay.ValueInt64()),
	}

	// Remote log
	s.RemoteLog = &client.RemoteLog{
		Enable:        plan.RemoteLogEnable.ValueBool(),
		Server:        plan.RemoteLogServer.ValueString(),
		Port:          int(plan.RemoteLogPort.ValueInt64()),
		MoreClientLog: plan.RemoteLogMoreClientLog.ValueBool(),
	}

	// Device account — only include if set
	if !plan.DeviceAccountUsername.IsNull() || !plan.DeviceAccountPassword.IsNull() {
		s.DeviceAccount = &client.DeviceAccount{
			Username: plan.DeviceAccountUsername.ValueString(),
			Password: plan.DeviceAccountPassword.ValueString(),
		}
	}

	// Remember device
	s.RememberDevice = &client.RememberDevice{
		Enable: plan.RememberDeviceEnable.ValueBool(),
	}

	return s
}

// siteSettingsToState maps the API response into the Terraform state model.
func siteSettingsToState(s *client.SiteSettings, state *SiteSettingsResourceModel) {
	// Site identity
	if s.Site != nil {
		state.SiteName = types.StringValue(s.Site.Name)
		state.Region = types.StringValue(s.Site.Region)
		state.TimeZone = types.StringValue(s.Site.TimeZone)
		state.Scenario = types.StringValue(s.Site.Scenario)
	}

	// Auto upgrade
	if s.AutoUpgrade != nil {
		state.AutoUpgradeEnable = types.BoolValue(s.AutoUpgrade.Enable)
	}

	// Mesh
	if s.Mesh != nil {
		state.MeshEnable = types.BoolValue(s.Mesh.MeshEnable)
		state.MeshAutoFailover = types.BoolValue(s.Mesh.AutoFailoverEnable)
		state.MeshDefGateway = types.BoolValue(s.Mesh.DefGatewayEnable)
		state.MeshFullSector = types.BoolValue(s.Mesh.FullSector)
	}

	// LED
	if s.LED != nil {
		state.LEDEnable = types.BoolValue(s.LED.Enable)
	}

	// Roaming
	if s.Roaming != nil {
		state.FastRoamingEnable = types.BoolValue(s.Roaming.FastRoamingEnable)
		state.AiRoamingEnable = types.BoolValue(s.Roaming.AiRoamingEnable)
		state.DualBand11kReportEnable = types.BoolValue(s.Roaming.DualBand11kReportEnable)
		state.ForceDisassociationEnable = types.BoolValue(s.Roaming.ForceDisassociationEnable)
		state.NonStickRoamingEnable = types.BoolValue(s.Roaming.NonStickRoamingEnable)
		state.NonPingPongRoamingEnable = types.BoolValue(s.Roaming.NonPingPongRoamingEnable)
	}

	// Band steering
	if s.BandSteering != nil {
		state.BandSteeringEnable = types.BoolValue(s.BandSteering.Enable)
		state.BandSteeringConnectionThreshold = types.Int64Value(int64(s.BandSteering.ConnectionThreshold))
		state.BandSteeringDifferenceThreshold = types.Int64Value(int64(s.BandSteering.DifferenceThreshold))
		state.BandSteeringMaxFailures = types.Int64Value(int64(s.BandSteering.MaxFailures))
	}
	if s.BandSteeringForMultiBand != nil {
		state.BandSteeringMultiBandMode = types.Int64Value(int64(s.BandSteeringForMultiBand.Mode))
	}

	// Airtime fairness
	if s.AirtimeFairness != nil {
		state.AirtimeFairness2g = types.BoolValue(s.AirtimeFairness.Enable2g)
		state.AirtimeFairness5g = types.BoolValue(s.AirtimeFairness.Enable5g)
		state.AirtimeFairness6g = types.BoolValue(s.AirtimeFairness.Enable6g)
	}

	// Beacon control
	if s.BeaconControl != nil {
		state.BeaconIntvMode2g = types.Int64Value(int64(s.BeaconControl.BeaconIntvMode2g))
		state.DtimPeriod2g = types.Int64Value(int64(s.BeaconControl.DtimPeriod2g))
		state.RtsThreshold2g = types.Int64Value(int64(s.BeaconControl.RtsThreshold2g))
		state.FragThreshold2g = types.Int64Value(int64(s.BeaconControl.FragmentationThreshold2g))
		state.BeaconIntvMode5g = types.Int64Value(int64(s.BeaconControl.BeaconIntvMode5g))
		state.DtimPeriod5g = types.Int64Value(int64(s.BeaconControl.DtimPeriod5g))
		state.RtsThreshold5g = types.Int64Value(int64(s.BeaconControl.RtsThreshold5g))
		state.FragThreshold5g = types.Int64Value(int64(s.BeaconControl.FragmentationThreshold5g))
		state.BeaconInterval6g = types.Int64Value(int64(s.BeaconControl.BeaconInterval6g))
		state.BeaconIntvMode6g = types.Int64Value(int64(s.BeaconControl.BeaconIntvMode6g))
		state.DtimPeriod6g = types.Int64Value(int64(s.BeaconControl.DtimPeriod6g))
		state.RtsThreshold6g = types.Int64Value(int64(s.BeaconControl.RtsThreshold6g))
		state.FragThreshold6g = types.Int64Value(int64(s.BeaconControl.FragmentationThreshold6g))
	}

	// LLDP
	if s.LLDP != nil {
		state.LLDPEnable = types.BoolValue(s.LLDP.Enable)
	}

	// Advanced features
	if s.AdvancedFeature != nil {
		state.AdvancedFeatureEnable = types.BoolValue(s.AdvancedFeature.Enable)
	}

	// Speed test
	if s.SpeedTest != nil {
		state.SpeedTestEnable = types.BoolValue(s.SpeedTest.Enable)
		state.SpeedTestInterval = types.Int64Value(int64(s.SpeedTest.Interval))
	}

	// Alert
	if s.Alert != nil {
		state.AlertEnable = types.BoolValue(s.Alert.Enable)
		state.AlertDelayEnable = types.BoolValue(s.Alert.DelayEnable)
		state.AlertDelay = types.Int64Value(int64(s.Alert.Delay))
	}

	// Remote log
	if s.RemoteLog != nil {
		state.RemoteLogEnable = types.BoolValue(s.RemoteLog.Enable)
		state.RemoteLogServer = types.StringValue(s.RemoteLog.Server)
		state.RemoteLogPort = types.Int64Value(int64(s.RemoteLog.Port))
		state.RemoteLogMoreClientLog = types.BoolValue(s.RemoteLog.MoreClientLog)
	}

	// Device account
	if s.DeviceAccount != nil {
		state.DeviceAccountUsername = types.StringValue(s.DeviceAccount.Username)
		// Password is returned as "***" from the API — preserve existing state if masked
		if s.DeviceAccount.Password != "" && s.DeviceAccount.Password != "***" {
			state.DeviceAccountPassword = types.StringValue(s.DeviceAccount.Password)
		}
		// If password is "***", keep whatever is in state already (don't overwrite with masked value)
	}

	// Remember device
	if s.RememberDevice != nil {
		state.RememberDeviceEnable = types.BoolValue(s.RememberDevice.Enable)
	}
}
