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

var _ resource.Resource = &DeviceAPResource{}
var _ resource.ResourceWithImportState = &DeviceAPResource{}

// DeviceAPResource manages an Omada AP device configuration.
type DeviceAPResource struct {
	client *client.Client
}

// DeviceAPResourceModel maps the resource schema to Go types.
type DeviceAPResourceModel struct {
	// Identity — ID is the MAC address, set via import
	ID     types.String `tfsdk:"id"`
	SiteID types.String `tfsdk:"site_id"`

	// Configurable fields
	Name   types.String `tfsdk:"name"`
	WlanID types.String `tfsdk:"wlan_group_id"`

	// Radio 2.4GHz
	Radio2gEnable       types.Bool   `tfsdk:"radio_2g_enable"`
	Radio2gChannelWidth types.String `tfsdk:"radio_2g_channel_width"`
	Radio2gChannel      types.String `tfsdk:"radio_2g_channel"`
	Radio2gTxPower      types.Int64  `tfsdk:"radio_2g_tx_power"`
	Radio2gTxPowerLevel types.Int64  `tfsdk:"radio_2g_tx_power_level"`

	// Radio 5GHz
	Radio5gEnable       types.Bool   `tfsdk:"radio_5g_enable"`
	Radio5gChannelWidth types.String `tfsdk:"radio_5g_channel_width"`
	Radio5gChannel      types.String `tfsdk:"radio_5g_channel"`
	Radio5gTxPower      types.Int64  `tfsdk:"radio_5g_tx_power"`
	Radio5gTxPowerLevel types.Int64  `tfsdk:"radio_5g_tx_power_level"`

	// IP Settings
	IPSettingMode types.String `tfsdk:"ip_setting_mode"`

	// LED / LLDP
	LEDSetting types.Int64 `tfsdk:"led_setting"`
	LLDPEnable types.Int64 `tfsdk:"lldp_enable"`

	// Management VLAN
	MVlanEnable    types.Bool   `tfsdk:"management_vlan_enable"`
	MVlanNetworkID types.String `tfsdk:"management_vlan_network_id"`

	// Feature toggles
	OFDMAEnable2g        types.Bool `tfsdk:"ofdma_enable_2g"`
	OFDMAEnable5g        types.Bool `tfsdk:"ofdma_enable_5g"`
	LoopbackDetectEnable types.Bool `tfsdk:"loopback_detect_enable"`
	L3AccessEnable       types.Bool `tfsdk:"l3_access_enable"`

	// Load balancing
	LB2gEnable     types.Bool  `tfsdk:"lb_2g_enable"`
	LB2gMaxClients types.Int64 `tfsdk:"lb_2g_max_clients"`
	LB5gEnable     types.Bool  `tfsdk:"lb_5g_enable"`
	LB5gMaxClients types.Int64 `tfsdk:"lb_5g_max_clients"`

	// RSSI
	RSSI2gEnable    types.Bool  `tfsdk:"rssi_2g_enable"`
	RSSI2gThreshold types.Int64 `tfsdk:"rssi_2g_threshold"`
	RSSI5gEnable    types.Bool  `tfsdk:"rssi_5g_enable"`
	RSSI5gThreshold types.Int64 `tfsdk:"rssi_5g_threshold"`

	// Read-only computed fields
	Model           types.String `tfsdk:"model"`
	IP              types.String `tfsdk:"ip"`
	FirmwareVersion types.String `tfsdk:"firmware_version"`
}

func NewDeviceAPResource() resource.Resource {
	return &DeviceAPResource{}
}

func (r *DeviceAPResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_ap"
}

func (r *DeviceAPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the configuration of an Omada AP device. " +
			"APs must be adopted through the controller UI before they can be managed. " +
			"Use 'terraform import omada_device_ap.<name> <siteID>/<mac>' to bring an adopted AP into state.",
		Attributes: map[string]schema.Attribute{
			// Identity — set by import, not user config
			"id": schema.StringAttribute{
				Description: "The AP MAC address (set by import).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID this device belongs to. Set by import, not configurable.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Configurable
			"name": schema.StringAttribute{
				Description: "The display name of the AP.",
				Optional:    true,
				Computed:    true,
			},
			"wlan_group_id": schema.StringAttribute{
				Description: "The WLAN group ID assigned to this AP.",
				Optional:    true,
				Computed:    true,
			},

			// Radio 2.4GHz
			"radio_2g_enable": schema.BoolAttribute{
				Description: "Enable 2.4GHz radio.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"radio_2g_channel_width": schema.StringAttribute{
				Description: "2.4GHz channel width (e.g., '0' for auto, '1' for 20MHz, '2' for 40MHz).",
				Optional:    true,
				Computed:    true,
			},
			"radio_2g_channel": schema.StringAttribute{
				Description: "2.4GHz channel (e.g., '0' for auto, '1'-'11' for specific).",
				Optional:    true,
				Computed:    true,
			},
			"radio_2g_tx_power": schema.Int64Attribute{
				Description: "2.4GHz TX power in dBm (used when tx_power_level=3/Custom).",
				Optional:    true,
				Computed:    true,
			},
			"radio_2g_tx_power_level": schema.Int64Attribute{
				Description: "2.4GHz TX power level: 0=Low, 1=Medium, 2=High, 3=Custom, 4=Auto.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(4),
			},

			// Radio 5GHz — absent on 2.4GHz-only APs (e.g., EAP115)
			"radio_5g_enable": schema.BoolAttribute{
				Description: "Enable 5GHz radio. Null on 2.4GHz-only APs.",
				Optional:    true,
				Computed:    true,
			},
			"radio_5g_channel_width": schema.StringAttribute{
				Description: "5GHz channel width (e.g., '0' for auto, '2' for 40MHz, '4' for 80MHz).",
				Optional:    true,
				Computed:    true,
			},
			"radio_5g_channel": schema.StringAttribute{
				Description: "5GHz channel (e.g., '0' for auto, specific channel numbers).",
				Optional:    true,
				Computed:    true,
			},
			"radio_5g_tx_power": schema.Int64Attribute{
				Description: "5GHz TX power in dBm (used when tx_power_level=3/Custom).",
				Optional:    true,
				Computed:    true,
			},
			"radio_5g_tx_power_level": schema.Int64Attribute{
				Description: "5GHz TX power level: 0=Low, 1=Medium, 2=High, 3=Custom, 4=Auto.",
				Optional:    true,
				Computed:    true,
			},

			// IP Settings
			"ip_setting_mode": schema.StringAttribute{
				Description: "IP address mode: 'dhcp' or 'static'.",
				Optional:    true,
				Computed:    true,
			},

			// LED / LLDP
			"led_setting": schema.Int64Attribute{
				Description: "LED setting: 0=Off, 1=On, 2=Follow site setting.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"lldp_enable": schema.Int64Attribute{
				Description: "LLDP: 0=Off, 1=On, 2=Follow site setting. Null on APs that don't support LLDP.",
				Optional:    true,
				Computed:    true,
			},

			// Management VLAN
			"management_vlan_enable": schema.BoolAttribute{
				Description: "Enable management VLAN.",
				Optional:    true,
				Computed:    true,
			},
			"management_vlan_network_id": schema.StringAttribute{
				Description: "The LAN network ID for the management VLAN.",
				Optional:    true,
				Computed:    true,
			},

			// Feature toggles
			"ofdma_enable_2g": schema.BoolAttribute{
				Description: "Enable OFDMA on 2.4GHz.",
				Optional:    true,
				Computed:    true,
			},
			"ofdma_enable_5g": schema.BoolAttribute{
				Description: "Enable OFDMA on 5GHz.",
				Optional:    true,
				Computed:    true,
			},
			"loopback_detect_enable": schema.BoolAttribute{
				Description: "Enable loopback detection.",
				Optional:    true,
				Computed:    true,
			},
			"l3_access_enable": schema.BoolAttribute{
				Description: "Enable L3 management access.",
				Optional:    true,
				Computed:    true,
			},

			// Load balancing
			"lb_2g_enable": schema.BoolAttribute{
				Description: "Enable load balancing on 2.4GHz.",
				Optional:    true,
				Computed:    true,
			},
			"lb_2g_max_clients": schema.Int64Attribute{
				Description: "Max clients for 2.4GHz load balancing.",
				Optional:    true,
				Computed:    true,
			},
			"lb_5g_enable": schema.BoolAttribute{
				Description: "Enable load balancing on 5GHz.",
				Optional:    true,
				Computed:    true,
			},
			"lb_5g_max_clients": schema.Int64Attribute{
				Description: "Max clients for 5GHz load balancing.",
				Optional:    true,
				Computed:    true,
			},

			// RSSI
			"rssi_2g_enable": schema.BoolAttribute{
				Description: "Enable RSSI threshold on 2.4GHz.",
				Optional:    true,
				Computed:    true,
			},
			"rssi_2g_threshold": schema.Int64Attribute{
				Description: "RSSI threshold for 2.4GHz (negative dBm).",
				Optional:    true,
				Computed:    true,
			},
			"rssi_5g_enable": schema.BoolAttribute{
				Description: "Enable RSSI threshold on 5GHz.",
				Optional:    true,
				Computed:    true,
			},
			"rssi_5g_threshold": schema.Int64Attribute{
				Description: "RSSI threshold for 5GHz (negative dBm).",
				Optional:    true,
				Computed:    true,
			},

			// Read-only
			"model": schema.StringAttribute{
				Description: "The AP model (e.g., 'EAP655-Wall'). Read-only.",
				Computed:    true,
			},
			"ip": schema.StringAttribute{
				Description: "The AP IP address. Read-only.",
				Computed:    true,
			},
			"firmware_version": schema.StringAttribute{
				Description: "The AP firmware version. Read-only.",
				Computed:    true,
			},
		},
	}
}

func (r *DeviceAPResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create always fails — APs must be adopted through the controller UI
// and imported into Terraform state.
func (r *DeviceAPResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"Cannot create AP device",
		"APs must be adopted through the Omada controller UI, then imported into Terraform:\n\n"+
			"  terraform import omada_device_ap.<name> <siteID>/<mac>",
	)
}

func (r *DeviceAPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeviceAPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apConfig, err := r.client.GetAPConfig(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading AP config", err.Error())
		return
	}

	apConfigToState(apConfig, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DeviceAPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeviceAPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DeviceAPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mac := state.ID.ValueString()
	siteID := state.SiteID.ValueString()

	// Fetch full raw config for PATCH
	rawConfig, err := r.client.GetAPConfigRaw(ctx, siteID, mac)
	if err != nil {
		resp.Diagnostics.AddError("Error reading AP config for update", err.Error())
		return
	}

	// Apply all plan values to raw config
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		rawConfig["name"] = plan.Name.ValueString()
	}
	if !plan.WlanID.IsNull() && !plan.WlanID.IsUnknown() {
		rawConfig["wlanId"] = plan.WlanID.ValueString()
	}
	if !plan.LEDSetting.IsNull() && !plan.LEDSetting.IsUnknown() {
		rawConfig["ledSetting"] = plan.LEDSetting.ValueInt64()
	}
	if !plan.LLDPEnable.IsNull() && !plan.LLDPEnable.IsUnknown() {
		rawConfig["lldpEnable"] = plan.LLDPEnable.ValueInt64()
	}
	if !plan.MVlanEnable.IsNull() && !plan.MVlanEnable.IsUnknown() {
		rawConfig["mvlanEnable"] = plan.MVlanEnable.ValueBool()
	}
	if !plan.OFDMAEnable2g.IsNull() && !plan.OFDMAEnable2g.IsUnknown() {
		rawConfig["ofdmaEnable2g"] = plan.OFDMAEnable2g.ValueBool()
	}
	if !plan.OFDMAEnable5g.IsNull() && !plan.OFDMAEnable5g.IsUnknown() {
		rawConfig["ofdmaEnable5g"] = plan.OFDMAEnable5g.ValueBool()
	}
	if !plan.LoopbackDetectEnable.IsNull() && !plan.LoopbackDetectEnable.IsUnknown() {
		rawConfig["loopbackDetectEnable"] = plan.LoopbackDetectEnable.ValueBool()
	}

	// Apply radio settings
	dummy := true
	applyRadioSettingsToRaw(rawConfig, "radioSetting2g", plan.Radio2gEnable, plan.Radio2gChannelWidth, plan.Radio2gChannel, plan.Radio2gTxPower, plan.Radio2gTxPowerLevel, &dummy)
	applyRadioSettingsToRaw(rawConfig, "radioSetting5g", plan.Radio5gEnable, plan.Radio5gChannelWidth, plan.Radio5gChannel, plan.Radio5gTxPower, plan.Radio5gTxPowerLevel, &dummy)

	// Apply IP setting
	if !plan.IPSettingMode.IsNull() && !plan.IPSettingMode.IsUnknown() {
		if ipSetting, ok := rawConfig["ipSetting"].(map[string]interface{}); ok {
			ipSetting["mode"] = plan.IPSettingMode.ValueString()
		} else {
			rawConfig["ipSetting"] = map[string]interface{}{"mode": plan.IPSettingMode.ValueString()}
		}
	}

	// Apply management VLAN
	if !plan.MVlanNetworkID.IsNull() && !plan.MVlanNetworkID.IsUnknown() {
		rawConfig["mvlanSetting"] = map[string]interface{}{
			"mode":         1,
			"lanNetworkId": plan.MVlanNetworkID.ValueString(),
		}
	}

	// Apply L3 access
	if !plan.L3AccessEnable.IsNull() && !plan.L3AccessEnable.IsUnknown() {
		rawConfig["l3AccessSetting"] = map[string]interface{}{"enable": plan.L3AccessEnable.ValueBool()}
	}

	// Apply load balancing
	applyLBSettingsToRaw(rawConfig, "lbSetting2g", plan.LB2gEnable, plan.LB2gMaxClients, &dummy)
	applyLBSettingsToRaw(rawConfig, "lbSetting5g", plan.LB5gEnable, plan.LB5gMaxClients, &dummy)

	// Apply RSSI
	applyRSSISettingsToRaw(rawConfig, "rssiSetting2g", plan.RSSI2gEnable, plan.RSSI2gThreshold, &dummy)
	applyRSSISettingsToRaw(rawConfig, "rssiSetting5g", plan.RSSI5gEnable, plan.RSSI5gThreshold, &dummy)

	_, err = r.client.UpdateAPConfig(ctx, siteID, mac, rawConfig)
	if err != nil {
		resp.Diagnostics.AddError("Error updating AP config", err.Error())
		return
	}

	// Read back fresh state
	apConfig, err := r.client.GetAPConfig(ctx, siteID, mac)
	if err != nil {
		resp.Diagnostics.AddError("Error reading AP config after update", err.Error())
		return
	}

	apConfigToState(apConfig, &plan)
	plan.SiteID = state.SiteID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete for an AP just removes from state — can't unadopt via API.
func (r *DeviceAPResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: APs cannot be deleted/unadopted via the API.
	// Removing from Terraform state is sufficient.
}

func (r *DeviceAPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	siteID, mac, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'siteID/mac'.",
		)
		return
	}

	apConfig, err := r.client.GetAPConfig(ctx, siteID, mac)
	if err != nil {
		resp.Diagnostics.AddError("Error importing AP config",
			fmt.Sprintf("Could not read AP with MAC %q: %s", mac, err.Error()))
		return
	}

	var state DeviceAPResourceModel
	state.SiteID = types.StringValue(siteID)
	apConfigToState(apConfig, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// apConfigToState maps an APConfig from the API to the Terraform state model.
func apConfigToState(cfg *client.APConfig, state *DeviceAPResourceModel) {
	state.ID = types.StringValue(cfg.MAC)
	state.Name = types.StringValue(cfg.Name)
	state.WlanID = types.StringValue(cfg.WlanID)

	// Radio 2.4GHz
	if cfg.RadioSetting2g != nil {
		state.Radio2gEnable = types.BoolValue(cfg.RadioSetting2g.RadioEnable)
		state.Radio2gChannelWidth = types.StringValue(cfg.RadioSetting2g.ChannelWidth)
		state.Radio2gChannel = types.StringValue(cfg.RadioSetting2g.Channel)
		state.Radio2gTxPower = types.Int64Value(int64(cfg.RadioSetting2g.TxPower))
		state.Radio2gTxPowerLevel = types.Int64Value(int64(cfg.RadioSetting2g.TxPowerLevel))
	}

	// Radio 5GHz — nil on 2.4GHz-only APs (e.g., EAP115)
	if cfg.RadioSetting5g != nil {
		state.Radio5gEnable = types.BoolValue(cfg.RadioSetting5g.RadioEnable)
		state.Radio5gChannelWidth = types.StringValue(cfg.RadioSetting5g.ChannelWidth)
		state.Radio5gChannel = types.StringValue(cfg.RadioSetting5g.Channel)
		state.Radio5gTxPower = types.Int64Value(int64(cfg.RadioSetting5g.TxPower))
		state.Radio5gTxPowerLevel = types.Int64Value(int64(cfg.RadioSetting5g.TxPowerLevel))
	} else {
		state.Radio5gEnable = types.BoolNull()
		state.Radio5gChannelWidth = types.StringNull()
		state.Radio5gChannel = types.StringNull()
		state.Radio5gTxPower = types.Int64Null()
		state.Radio5gTxPowerLevel = types.Int64Null()
	}

	// IP Settings
	if cfg.IPSetting != nil {
		state.IPSettingMode = types.StringValue(cfg.IPSetting.Mode)
	} else {
		state.IPSettingMode = types.StringValue("dhcp")
	}

	// LED
	state.LEDSetting = types.Int64Value(int64(cfg.LEDSetting))

	// LLDP — nil on APs that don't support it (e.g., EAP115)
	if cfg.LLDPEnable != nil {
		state.LLDPEnable = types.Int64Value(int64(*cfg.LLDPEnable))
	} else {
		state.LLDPEnable = types.Int64Null()
	}

	// Management VLAN
	state.MVlanEnable = types.BoolValue(cfg.MVlanEnable)
	if cfg.MVlanSetting != nil && cfg.MVlanSetting.LanNetworkID != "" {
		state.MVlanNetworkID = types.StringValue(cfg.MVlanSetting.LanNetworkID)
	} else {
		state.MVlanNetworkID = types.StringNull()
	}

	// Feature toggles — nil on APs that don't support them
	if cfg.OFDMAEnable2g != nil {
		state.OFDMAEnable2g = types.BoolValue(*cfg.OFDMAEnable2g)
	} else {
		state.OFDMAEnable2g = types.BoolNull()
	}
	if cfg.OFDMAEnable5g != nil {
		state.OFDMAEnable5g = types.BoolValue(*cfg.OFDMAEnable5g)
	} else {
		state.OFDMAEnable5g = types.BoolNull()
	}
	if cfg.LoopbackDetectEnable != nil {
		state.LoopbackDetectEnable = types.BoolValue(*cfg.LoopbackDetectEnable)
	} else {
		state.LoopbackDetectEnable = types.BoolNull()
	}
	if cfg.L3AccessSetting != nil {
		state.L3AccessEnable = types.BoolValue(cfg.L3AccessSetting.Enable)
	} else {
		state.L3AccessEnable = types.BoolNull()
	}

	// Load balancing 2g
	if cfg.LBSetting2g != nil {
		state.LB2gEnable = types.BoolValue(cfg.LBSetting2g.LBEnable)
		state.LB2gMaxClients = types.Int64Value(int64(cfg.LBSetting2g.MaxClients))
	} else {
		state.LB2gEnable = types.BoolNull()
		state.LB2gMaxClients = types.Int64Null()
	}

	// Load balancing 5g — nil on 2.4GHz-only APs
	if cfg.LBSetting5g != nil {
		state.LB5gEnable = types.BoolValue(cfg.LBSetting5g.LBEnable)
		state.LB5gMaxClients = types.Int64Value(int64(cfg.LBSetting5g.MaxClients))
	} else {
		state.LB5gEnable = types.BoolNull()
		state.LB5gMaxClients = types.Int64Null()
	}

	// RSSI 2g
	if cfg.RSSISetting2g != nil {
		state.RSSI2gEnable = types.BoolValue(cfg.RSSISetting2g.RSSIEnable)
		state.RSSI2gThreshold = types.Int64Value(int64(cfg.RSSISetting2g.Threshold))
	} else {
		state.RSSI2gEnable = types.BoolNull()
		state.RSSI2gThreshold = types.Int64Null()
	}

	// RSSI 5g — nil on 2.4GHz-only APs
	if cfg.RSSISetting5g != nil {
		state.RSSI5gEnable = types.BoolValue(cfg.RSSISetting5g.RSSIEnable)
		state.RSSI5gThreshold = types.Int64Value(int64(cfg.RSSISetting5g.Threshold))
	} else {
		state.RSSI5gEnable = types.BoolNull()
		state.RSSI5gThreshold = types.Int64Null()
	}

	// Read-only
	state.Model = types.StringValue(cfg.Model)
	state.IP = types.StringValue(cfg.IP)
	state.FirmwareVersion = types.StringValue(cfg.FirmwareVersion)
}

// applyRadioSettingsToRaw applies radio settings from plan to raw JSON config.
func applyRadioSettingsToRaw(raw map[string]interface{}, key string, enable types.Bool, channelWidth types.String, channel types.String, txPower types.Int64, txPowerLevel types.Int64, needsUpdate *bool) {
	anySet := !enable.IsNull() || !channelWidth.IsNull() || !channel.IsNull() || !txPower.IsNull() || !txPowerLevel.IsNull()
	if !anySet {
		return
	}

	radio, ok := raw[key].(map[string]interface{})
	if !ok {
		radio = map[string]interface{}{}
		raw[key] = radio
	}

	if !enable.IsNull() && !enable.IsUnknown() {
		radio["radioEnable"] = enable.ValueBool()
		*needsUpdate = true
	}
	if !channelWidth.IsNull() && !channelWidth.IsUnknown() {
		radio["channelWidth"] = channelWidth.ValueString()
		*needsUpdate = true
	}
	if !channel.IsNull() && !channel.IsUnknown() {
		radio["channel"] = channel.ValueString()
		*needsUpdate = true
	}
	if !txPower.IsNull() && !txPower.IsUnknown() {
		radio["txPower"] = txPower.ValueInt64()
		*needsUpdate = true
	}
	if !txPowerLevel.IsNull() && !txPowerLevel.IsUnknown() {
		radio["txPowerLevel"] = txPowerLevel.ValueInt64()
		*needsUpdate = true
	}
}

// applyLBSettingsToRaw applies load balancing settings from plan to raw JSON config.
func applyLBSettingsToRaw(raw map[string]interface{}, key string, enable types.Bool, maxClients types.Int64, needsUpdate *bool) {
	if enable.IsNull() && maxClients.IsNull() {
		return
	}

	lb, ok := raw[key].(map[string]interface{})
	if !ok {
		lb = map[string]interface{}{}
		raw[key] = lb
	}

	if !enable.IsNull() && !enable.IsUnknown() {
		lb["lbEnable"] = enable.ValueBool()
		*needsUpdate = true
	}
	if !maxClients.IsNull() && !maxClients.IsUnknown() {
		lb["maxClients"] = maxClients.ValueInt64()
		*needsUpdate = true
	}
}

// applyRSSISettingsToRaw applies RSSI settings from plan to raw JSON config.
func applyRSSISettingsToRaw(raw map[string]interface{}, key string, enable types.Bool, threshold types.Int64, needsUpdate *bool) {
	if enable.IsNull() && threshold.IsNull() {
		return
	}

	rssi, ok := raw[key].(map[string]interface{})
	if !ok {
		rssi = map[string]interface{}{}
		raw[key] = rssi
	}

	if !enable.IsNull() && !enable.IsUnknown() {
		rssi["rssiEnable"] = enable.ValueBool()
		*needsUpdate = true
	}
	if !threshold.IsNull() && !threshold.IsUnknown() {
		rssi["threshold"] = threshold.ValueInt64()
		*needsUpdate = true
	}
}
