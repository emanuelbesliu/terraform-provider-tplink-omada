package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-tplink-omada/internal/client"
)

var _ datasource.DataSource = &NetworksDataSource{}

// NetworksDataSource lists all networks on the Omada Controller.
type NetworksDataSource struct {
	client *client.Client
}

type NetworksDataSourceModel struct {
	SiteID   types.String       `tfsdk:"site_id"`
	Networks []NetworkDataModel `tfsdk:"networks"`
}

type NetworkDataModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Purpose       types.String `tfsdk:"purpose"`
	VlanID        types.Int64  `tfsdk:"vlan_id"`
	GatewaySubnet types.String `tfsdk:"gateway_subnet"`
	DHCPEnabled   types.Bool   `tfsdk:"dhcp_enabled"`
}

func NewNetworksDataSource() datasource.DataSource {
	return &NetworksDataSource{}
}

func (d *NetworksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_networks"
}

func (d *NetworksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all LAN networks on the Omada Controller for the given site.",
		Attributes: map[string]schema.Attribute{
			"site_id": siteIDDataSourceSchema(),
			"networks": schema.ListNestedAttribute{
				Description: "List of networks.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             schema.StringAttribute{Description: "The network ID.", Computed: true},
						"name":           schema.StringAttribute{Description: "The network name.", Computed: true},
						"purpose":        schema.StringAttribute{Description: "The network purpose ('interface' or 'vlan').", Computed: true},
						"vlan_id":        schema.Int64Attribute{Description: "The VLAN ID.", Computed: true},
						"gateway_subnet": schema.StringAttribute{Description: "The gateway IP and subnet in CIDR notation.", Computed: true},
						"dhcp_enabled":   schema.BoolAttribute{Description: "Whether DHCP is enabled.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *NetworksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *NetworksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config NetworksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := config.SiteID.ValueString()

	networks, err := d.client.ListNetworks(ctx, siteID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing networks", err.Error())
		return
	}

	state := NetworksDataSourceModel{
		SiteID: config.SiteID,
	}
	for _, n := range networks {
		dm := NetworkDataModel{
			ID:            types.StringValue(n.ID),
			Name:          types.StringValue(n.Name),
			Purpose:       types.StringValue(n.Purpose),
			VlanID:        types.Int64Value(int64(n.Vlan)),
			GatewaySubnet: types.StringValue(n.GatewaySubnet),
		}
		if n.DHCPSettings != nil {
			dm.DHCPEnabled = types.BoolValue(n.DHCPSettings.Enable)
		} else {
			dm.DHCPEnabled = types.BoolNull()
		}
		state.Networks = append(state.Networks, dm)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- Wireless Networks Data Source ---

var _ datasource.DataSource = &WirelessNetworksDataSource{}

type WirelessNetworksDataSource struct {
	client *client.Client
}

type WirelessNetworksDataSourceModel struct {
	SiteID           types.String               `tfsdk:"site_id"`
	WlanGroupID      types.String               `tfsdk:"wlan_group_id"`
	WirelessNetworks []WirelessNetworkDataModel `tfsdk:"wireless_networks"`
}

type WirelessNetworkDataModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Band      types.Int64  `tfsdk:"band"`
	Security  types.Int64  `tfsdk:"security"`
	Broadcast types.Bool   `tfsdk:"broadcast"`
	VlanID    types.Int64  `tfsdk:"vlan_id"`
	Enable11r types.Bool   `tfsdk:"enable_11r"`
	PmfMode   types.Int64  `tfsdk:"pmf_mode"`
}

func NewWirelessNetworksDataSource() datasource.DataSource {
	return &WirelessNetworksDataSource{}
}

func (d *WirelessNetworksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wireless_networks"
}

func (d *WirelessNetworksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all wireless networks (SSIDs) for a WLAN group on the Omada Controller.",
		Attributes: map[string]schema.Attribute{
			"site_id": siteIDDataSourceSchema(),
			"wlan_group_id": schema.StringAttribute{
				Description: "The WLAN group ID to list SSIDs from. If not set, the default WLAN group is used.",
				Optional:    true,
				Computed:    true,
			},
			"wireless_networks": schema.ListNestedAttribute{
				Description: "List of wireless networks (SSIDs).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{Description: "The SSID ID.", Computed: true},
						"name":       schema.StringAttribute{Description: "The SSID name.", Computed: true},
						"band":       schema.Int64Attribute{Description: "Radio band: 1=2.4GHz, 2=5GHz, 3=both.", Computed: true},
						"security":   schema.Int64Attribute{Description: "Security mode: 0=Open, 3=WPA2/WPA3.", Computed: true},
						"broadcast":  schema.BoolAttribute{Description: "Whether the SSID is broadcast (visible).", Computed: true},
						"vlan_id":    schema.Int64Attribute{Description: "The VLAN ID assigned to this SSID.", Computed: true},
						"enable_11r": schema.BoolAttribute{Description: "Whether 802.11r Fast Roaming is enabled.", Computed: true},
						"pmf_mode":   schema.Int64Attribute{Description: "Protected Management Frames mode.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *WirelessNetworksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *WirelessNetworksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WirelessNetworksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := config.SiteID.ValueString()

	wlanGroupID := config.WlanGroupID.ValueString()
	if wlanGroupID == "" {
		gid, err := d.client.GetDefaultWlanGroupID(ctx, siteID)
		if err != nil {
			resp.Diagnostics.AddError("Error getting default WLAN group", err.Error())
			return
		}
		wlanGroupID = gid
	}

	ssids, err := d.client.ListWirelessNetworks(ctx, siteID, wlanGroupID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing wireless networks", err.Error())
		return
	}

	state := WirelessNetworksDataSourceModel{
		SiteID:      config.SiteID,
		WlanGroupID: types.StringValue(wlanGroupID),
	}
	for _, s := range ssids {
		dm := WirelessNetworkDataModel{
			ID:        types.StringValue(s.ID),
			Name:      types.StringValue(s.Name),
			Band:      types.Int64Value(int64(s.Band)),
			Security:  types.Int64Value(int64(s.Security)),
			Broadcast: types.BoolValue(s.Broadcast),
			Enable11r: types.BoolValue(s.Enable11r),
			PmfMode:   types.Int64Value(int64(s.PmfMode)),
		}
		// Extract VLAN ID from vlanSetting
		if s.VlanSetting != nil && s.VlanSetting.CustomConfig != nil && s.VlanSetting.CustomConfig.BridgeVlan > 0 {
			dm.VlanID = types.Int64Value(int64(s.VlanSetting.CustomConfig.BridgeVlan))
		} else if s.VlanSetting != nil && s.VlanSetting.CurrentVlanId > 0 {
			dm.VlanID = types.Int64Value(int64(s.VlanSetting.CurrentVlanId))
		} else {
			dm.VlanID = types.Int64Null()
		}
		state.WirelessNetworks = append(state.WirelessNetworks, dm)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- Port Profiles Data Source ---

var _ datasource.DataSource = &PortProfilesDataSource{}

type PortProfilesDataSource struct {
	client *client.Client
}

type PortProfilesDataSourceModel struct {
	SiteID       types.String           `tfsdk:"site_id"`
	PortProfiles []PortProfileDataModel `tfsdk:"port_profiles"`
}

type PortProfileDataModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	NativeNetworkID types.String `tfsdk:"native_network_id"`
	TagNetworkIDs   types.List   `tfsdk:"tag_network_ids"`
	POE             types.Int64  `tfsdk:"poe"`
	Dot1x           types.Int64  `tfsdk:"dot1x"`
	Type            types.Int64  `tfsdk:"type"`
}

func NewPortProfilesDataSource() datasource.DataSource {
	return &PortProfilesDataSource{}
}

func (d *PortProfilesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_profiles"
}

func (d *PortProfilesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all switch port profiles on the Omada Controller for the given site.",
		Attributes: map[string]schema.Attribute{
			"site_id": siteIDDataSourceSchema(),
			"port_profiles": schema.ListNestedAttribute{
				Description: "List of port profiles.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                schema.StringAttribute{Description: "The port profile ID.", Computed: true},
						"name":              schema.StringAttribute{Description: "The port profile name.", Computed: true},
						"native_network_id": schema.StringAttribute{Description: "The native (untagged) network ID.", Computed: true},
						"tag_network_ids":   schema.ListAttribute{Description: "Tagged network IDs.", Computed: true, ElementType: types.StringType},
						"poe":               schema.Int64Attribute{Description: "PoE setting.", Computed: true},
						"dot1x":             schema.Int64Attribute{Description: "802.1X setting.", Computed: true},
						"type":              schema.Int64Attribute{Description: "Profile type.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *PortProfilesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *PortProfilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PortProfilesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := config.SiteID.ValueString()

	profiles, err := d.client.ListPortProfiles(ctx, siteID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing port profiles", err.Error())
		return
	}

	state := PortProfilesDataSourceModel{
		SiteID: config.SiteID,
	}
	for _, p := range profiles {
		tagIDs, diags := types.ListValueFrom(ctx, types.StringType, p.TagNetworkIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.PortProfiles = append(state.PortProfiles, PortProfileDataModel{
			ID:              types.StringValue(p.ID),
			Name:            types.StringValue(p.Name),
			NativeNetworkID: types.StringValue(p.NativeNetworkID),
			TagNetworkIDs:   tagIDs,
			POE:             types.Int64Value(int64(p.POE)),
			Dot1x:           types.Int64Value(int64(p.Dot1x)),
			Type:            types.Int64Value(int64(p.Type)),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- Sites Data Source ---

var _ datasource.DataSource = &SitesDataSource{}

type SitesDataSource struct {
	client *client.Client
}

type SitesDataSourceModel struct {
	Sites []SiteDataModel `tfsdk:"sites"`
}

type SiteDataModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewSitesDataSource() datasource.DataSource {
	return &SitesDataSource{}
}

func (d *SitesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sites"
}

func (d *SitesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all sites on the Omada Controller.",
		Attributes: map[string]schema.Attribute{
			"sites": schema.ListNestedAttribute{
				Description: "List of sites.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Description: "The site ID.", Computed: true},
						"name": schema.StringAttribute{Description: "The site name.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *SitesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *SitesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	sites, err := d.client.ListSites(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing sites", err.Error())
		return
	}

	var state SitesDataSourceModel
	for _, s := range sites {
		state.Sites = append(state.Sites, SiteDataModel{
			ID:   types.StringValue(s.ID),
			Name: types.StringValue(s.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- Site Settings Data Source ---

var _ datasource.DataSource = &SiteSettingsDataSource{}

// SiteSettingsDataSource reads the settings for a site.
type SiteSettingsDataSource struct {
	client *client.Client
}

// SiteSettingsDataSourceModel maps the data source schema.
type SiteSettingsDataSourceModel struct {
	SiteID types.String `tfsdk:"site_id"`

	// Site identity
	SiteName types.String `tfsdk:"site_name"`
	Region   types.String `tfsdk:"region"`
	TimeZone types.String `tfsdk:"timezone"`
	Scenario types.String `tfsdk:"scenario"`

	// Feature toggles
	AutoUpgradeEnable     types.Bool `tfsdk:"auto_upgrade_enable"`
	MeshEnable            types.Bool `tfsdk:"mesh_enable"`
	MeshAutoFailover      types.Bool `tfsdk:"mesh_auto_failover"`
	MeshDefGateway        types.Bool `tfsdk:"mesh_default_gateway"`
	MeshFullSector        types.Bool `tfsdk:"mesh_full_sector"`
	LEDEnable             types.Bool `tfsdk:"led_enable"`
	LLDPEnable            types.Bool `tfsdk:"lldp_enable"`
	AdvancedFeatureEnable types.Bool `tfsdk:"advanced_feature_enable"`

	// Roaming
	FastRoamingEnable         types.Bool `tfsdk:"fast_roaming_enable"`
	AiRoamingEnable           types.Bool `tfsdk:"ai_roaming_enable"`
	DualBand11kReportEnable   types.Bool `tfsdk:"dual_band_11k_report_enable"`
	ForceDisassociationEnable types.Bool `tfsdk:"force_disassociation_enable"`
	NonStickRoamingEnable     types.Bool `tfsdk:"non_stick_roaming_enable"`
	NonPingPongRoamingEnable  types.Bool `tfsdk:"non_ping_pong_roaming_enable"`

	// Band steering
	BandSteeringEnable        types.Bool  `tfsdk:"band_steering_enable"`
	BandSteeringMultiBandMode types.Int64 `tfsdk:"band_steering_multi_band_mode"`

	// Airtime fairness
	AirtimeFairness2g types.Bool `tfsdk:"airtime_fairness_2g"`
	AirtimeFairness5g types.Bool `tfsdk:"airtime_fairness_5g"`
	AirtimeFairness6g types.Bool `tfsdk:"airtime_fairness_6g"`

	// Speed test
	SpeedTestEnable   types.Bool  `tfsdk:"speed_test_enable"`
	SpeedTestInterval types.Int64 `tfsdk:"speed_test_interval"`

	// Alert
	AlertEnable types.Bool `tfsdk:"alert_enable"`

	// Remote log
	RemoteLogEnable types.Bool `tfsdk:"remote_log_enable"`

	// Device account
	DeviceAccountUsername types.String `tfsdk:"device_account_username"`

	// Remember device
	RememberDeviceEnable types.Bool `tfsdk:"remember_device_enable"`
}

func NewSiteSettingsDataSource() datasource.DataSource {
	return &SiteSettingsDataSource{}
}

func (d *SiteSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_settings"
}

func (d *SiteSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the current settings for an Omada site.",
		Attributes: map[string]schema.Attribute{
			"site_id":                       siteIDDataSourceSchema(),
			"site_name":                     schema.StringAttribute{Description: "The site display name.", Computed: true},
			"region":                        schema.StringAttribute{Description: "The site region.", Computed: true},
			"timezone":                      schema.StringAttribute{Description: "The site timezone.", Computed: true},
			"scenario":                      schema.StringAttribute{Description: "The site scenario.", Computed: true},
			"auto_upgrade_enable":           schema.BoolAttribute{Description: "Auto firmware upgrade.", Computed: true},
			"mesh_enable":                   schema.BoolAttribute{Description: "Mesh networking enabled.", Computed: true},
			"mesh_auto_failover":            schema.BoolAttribute{Description: "Mesh auto failover.", Computed: true},
			"mesh_default_gateway":          schema.BoolAttribute{Description: "Mesh default gateway.", Computed: true},
			"mesh_full_sector":              schema.BoolAttribute{Description: "Mesh full sector.", Computed: true},
			"led_enable":                    schema.BoolAttribute{Description: "AP LED enabled.", Computed: true},
			"lldp_enable":                   schema.BoolAttribute{Description: "LLDP enabled.", Computed: true},
			"advanced_feature_enable":       schema.BoolAttribute{Description: "Advanced features enabled.", Computed: true},
			"fast_roaming_enable":           schema.BoolAttribute{Description: "802.11r fast roaming.", Computed: true},
			"ai_roaming_enable":             schema.BoolAttribute{Description: "AI roaming.", Computed: true},
			"dual_band_11k_report_enable":   schema.BoolAttribute{Description: "Dual-band 11k reports.", Computed: true},
			"force_disassociation_enable":   schema.BoolAttribute{Description: "Force disassociation.", Computed: true},
			"non_stick_roaming_enable":      schema.BoolAttribute{Description: "Non-sticky roaming.", Computed: true},
			"non_ping_pong_roaming_enable":  schema.BoolAttribute{Description: "Non-ping-pong roaming.", Computed: true},
			"band_steering_enable":          schema.BoolAttribute{Description: "Band steering.", Computed: true},
			"band_steering_multi_band_mode": schema.Int64Attribute{Description: "Multi-band steering mode.", Computed: true},
			"airtime_fairness_2g":           schema.BoolAttribute{Description: "Airtime fairness 2.4GHz.", Computed: true},
			"airtime_fairness_5g":           schema.BoolAttribute{Description: "Airtime fairness 5GHz.", Computed: true},
			"airtime_fairness_6g":           schema.BoolAttribute{Description: "Airtime fairness 6GHz.", Computed: true},
			"speed_test_enable":             schema.BoolAttribute{Description: "Speed test enabled.", Computed: true},
			"speed_test_interval":           schema.Int64Attribute{Description: "Speed test interval (minutes).", Computed: true},
			"alert_enable":                  schema.BoolAttribute{Description: "Alerts enabled.", Computed: true},
			"remote_log_enable":             schema.BoolAttribute{Description: "Remote syslog enabled.", Computed: true},
			"device_account_username":       schema.StringAttribute{Description: "Device SSH username.", Computed: true},
			"remember_device_enable":        schema.BoolAttribute{Description: "Remember device.", Computed: true},
		},
	}
}

func (d *SiteSettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *SiteSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SiteSettingsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := config.SiteID.ValueString()

	settings, err := d.client.GetSiteSettings(ctx, siteID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading site settings", err.Error())
		return
	}

	state := SiteSettingsDataSourceModel{
		SiteID: config.SiteID,
	}

	if settings.Site != nil {
		state.SiteName = types.StringValue(settings.Site.Name)
		state.Region = types.StringValue(settings.Site.Region)
		state.TimeZone = types.StringValue(settings.Site.TimeZone)
		state.Scenario = types.StringValue(settings.Site.Scenario)
	}
	if settings.AutoUpgrade != nil {
		state.AutoUpgradeEnable = types.BoolValue(settings.AutoUpgrade.Enable)
	}
	if settings.Mesh != nil {
		state.MeshEnable = types.BoolValue(settings.Mesh.MeshEnable)
		state.MeshAutoFailover = types.BoolValue(settings.Mesh.AutoFailoverEnable)
		state.MeshDefGateway = types.BoolValue(settings.Mesh.DefGatewayEnable)
		state.MeshFullSector = types.BoolValue(settings.Mesh.FullSector)
	}
	if settings.LED != nil {
		state.LEDEnable = types.BoolValue(settings.LED.Enable)
	}
	if settings.LLDP != nil {
		state.LLDPEnable = types.BoolValue(settings.LLDP.Enable)
	}
	if settings.AdvancedFeature != nil {
		state.AdvancedFeatureEnable = types.BoolValue(settings.AdvancedFeature.Enable)
	}
	if settings.Roaming != nil {
		state.FastRoamingEnable = types.BoolValue(settings.Roaming.FastRoamingEnable)
		state.AiRoamingEnable = types.BoolValue(settings.Roaming.AiRoamingEnable)
		state.DualBand11kReportEnable = types.BoolValue(settings.Roaming.DualBand11kReportEnable)
		state.ForceDisassociationEnable = types.BoolValue(settings.Roaming.ForceDisassociationEnable)
		state.NonStickRoamingEnable = types.BoolValue(settings.Roaming.NonStickRoamingEnable)
		state.NonPingPongRoamingEnable = types.BoolValue(settings.Roaming.NonPingPongRoamingEnable)
	}
	if settings.BandSteering != nil {
		state.BandSteeringEnable = types.BoolValue(settings.BandSteering.Enable)
	}
	if settings.BandSteeringForMultiBand != nil {
		state.BandSteeringMultiBandMode = types.Int64Value(int64(settings.BandSteeringForMultiBand.Mode))
	}
	if settings.AirtimeFairness != nil {
		state.AirtimeFairness2g = types.BoolValue(settings.AirtimeFairness.Enable2g)
		state.AirtimeFairness5g = types.BoolValue(settings.AirtimeFairness.Enable5g)
		state.AirtimeFairness6g = types.BoolValue(settings.AirtimeFairness.Enable6g)
	}
	if settings.SpeedTest != nil {
		state.SpeedTestEnable = types.BoolValue(settings.SpeedTest.Enable)
		state.SpeedTestInterval = types.Int64Value(int64(settings.SpeedTest.Interval))
	}
	if settings.Alert != nil {
		state.AlertEnable = types.BoolValue(settings.Alert.Enable)
	}
	if settings.RemoteLog != nil {
		state.RemoteLogEnable = types.BoolValue(settings.RemoteLog.Enable)
	}
	if settings.DeviceAccount != nil {
		state.DeviceAccountUsername = types.StringValue(settings.DeviceAccount.Username)
	}
	if settings.RememberDevice != nil {
		state.RememberDeviceEnable = types.BoolValue(settings.RememberDevice.Enable)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- Devices Data Source ---

var _ datasource.DataSource = &DevicesDataSource{}

// DevicesDataSource lists all devices in a site.
type DevicesDataSource struct {
	client *client.Client
}

// DevicesDataSourceModel maps the data source schema.
type DevicesDataSourceModel struct {
	SiteID  types.String      `tfsdk:"site_id"`
	Devices []DeviceDataModel `tfsdk:"devices"`
}

// DeviceDataModel represents a single device in the data source output.
type DeviceDataModel struct {
	Type            types.String  `tfsdk:"type"`
	MAC             types.String  `tfsdk:"mac"`
	Name            types.String  `tfsdk:"name"`
	Model           types.String  `tfsdk:"model"`
	FirmwareVersion types.String  `tfsdk:"firmware_version"`
	IP              types.String  `tfsdk:"ip"`
	Status          types.Int64   `tfsdk:"status"`
	StatusCategory  types.Int64   `tfsdk:"status_category"`
	ClientNum       types.Int64   `tfsdk:"client_num"`
	CPUUtil         types.Float64 `tfsdk:"cpu_util"`
	MemUtil         types.Float64 `tfsdk:"mem_util"`
}

func NewDevicesDataSource() datasource.DataSource {
	return &DevicesDataSource{}
}

func (d *DevicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_devices"
}

func (d *DevicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all devices (APs, switches, gateways) in a site.",
		Attributes: map[string]schema.Attribute{
			"site_id": siteIDDataSourceSchema(),
			"devices": schema.ListNestedAttribute{
				Description: "List of devices.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type":             schema.StringAttribute{Description: "Device type: 'ap', 'switch', or 'gateway'.", Computed: true},
						"mac":              schema.StringAttribute{Description: "The device MAC address.", Computed: true},
						"name":             schema.StringAttribute{Description: "The device display name.", Computed: true},
						"model":            schema.StringAttribute{Description: "The device model.", Computed: true},
						"firmware_version": schema.StringAttribute{Description: "The current firmware version.", Computed: true},
						"ip":               schema.StringAttribute{Description: "The device IP address.", Computed: true},
						"status":           schema.Int64Attribute{Description: "Device status code.", Computed: true},
						"status_category":  schema.Int64Attribute{Description: "Device status category.", Computed: true},
						"client_num":       schema.Int64Attribute{Description: "Number of connected clients.", Computed: true},
						"cpu_util":         schema.Float64Attribute{Description: "CPU utilization percentage.", Computed: true},
						"mem_util":         schema.Float64Attribute{Description: "Memory utilization percentage.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *DevicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *DevicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DevicesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := config.SiteID.ValueString()

	devices, err := d.client.ListDevices(ctx, siteID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing devices", err.Error())
		return
	}

	state := DevicesDataSourceModel{
		SiteID: config.SiteID,
	}
	for _, dev := range devices {
		state.Devices = append(state.Devices, DeviceDataModel{
			Type:            types.StringValue(dev.Type),
			MAC:             types.StringValue(dev.MAC),
			Name:            types.StringValue(dev.Name),
			Model:           types.StringValue(dev.Model),
			FirmwareVersion: types.StringValue(dev.FirmwareVersion),
			IP:              types.StringValue(dev.IP),
			Status:          types.Int64Value(int64(dev.Status)),
			StatusCategory:  types.Int64Value(int64(dev.StatusCategory)),
			ClientNum:       types.Int64Value(int64(dev.ClientNum)),
			CPUUtil:         types.Float64Value(dev.CPUUtil),
			MemUtil:         types.Float64Value(dev.MemUtil),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
