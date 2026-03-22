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
	// Identity — MAC is the ID (immutable)
	MAC types.String `tfsdk:"mac"`

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
	IPSettingMode         types.String `tfsdk:"ip_setting_mode"`
	IPSettingFallback     types.Bool   `tfsdk:"ip_setting_fallback"`
	IPSettingFallbackIP   types.String `tfsdk:"ip_setting_fallback_ip"`
	IPSettingFallbackMask types.String `tfsdk:"ip_setting_fallback_mask"`
	IPSettingFallbackGate types.String `tfsdk:"ip_setting_fallback_gate"`

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
			"APs cannot be created or deleted via the API — this resource manages the configuration " +
			"of an already-adopted AP. Import by MAC address. Delete removes from Terraform state only.",
		Attributes: map[string]schema.Attribute{
			"mac": schema.StringAttribute{
				Description: "The AP MAC address (unique identifier). Used for import.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
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
			"radio_2g_enable": schema.BoolAttribute{
				Description: "Enable 2.4GHz radio.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"radio_2g_channel_width": schema.StringAttribute{
				Description: "2.4GHz channel width. RADIO_20=2; RADIO_40=3; RADIO_40_20=4 (Auto).",
				Optional:    true,
				Computed:    true,
			},
			"radio_2g_channel": schema.StringAttribute{
				Description: "2.4GHz channel (0 for auto).",
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
			"radio_5g_enable": schema.BoolAttribute{
				Description: "Enable 5GHz radio. Null on 2.4GHz-only APs.",
				Optional:    true,
				Computed:    true,
			},
			"radio_5g_channel_width": schema.StringAttribute{
				Description: "5GHz channel width. RADIO_80=5; RADIO_80_40_20=6 (Auto); RADIO_160=7.",
				Optional:    true,
				Computed:    true,
			},
			"radio_5g_channel": schema.StringAttribute{
				Description: "5GHz channel (0 for auto).",
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
			"ip_setting_mode": schema.StringAttribute{
				Description: "IP address mode: 'dhcp' or 'static'.",
				Optional:    true,
				Computed:    true,
			},
			"ip_setting_fallback": schema.BoolAttribute{
				Description: "Enable fallback static IP when DHCP fails (dhcp mode only).",
				Optional:    true,
				Computed:    true,
			},
			"ip_setting_fallback_ip": schema.StringAttribute{
				Description: "Fallback static IP address (used when ip_setting_fallback = true).",
				Optional:    true,
				Computed:    true,
			},
			"ip_setting_fallback_mask": schema.StringAttribute{
				Description: "Fallback subnet mask (used when ip_setting_fallback = true).",
				Optional:    true,
				Computed:    true,
			},
			"ip_setting_fallback_gate": schema.StringAttribute{
				Description: "Fallback gateway IP (used when ip_setting_fallback = true).",
				Optional:    true,
				Computed:    true,
			},
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
			"model": schema.StringAttribute{
				Description: "The AP model. Read-only.",
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

// applyAPPlanToRaw merges general plan values into rawConfig for the main PATCH call.
// Note: radio settings, OFDMA, LB, RSSI, LLDP, and L3 access are handled by
// separate dedicated endpoints and must NOT be included here — the Omada API
// silently ignores these fields on the main PATCH endpoint.
func applyAPPlanToRaw(rawConfig map[string]interface{}, plan *DeviceAPResourceModel) {
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		rawConfig["name"] = plan.Name.ValueString()
	}
	if !plan.WlanID.IsNull() && !plan.WlanID.IsUnknown() {
		rawConfig["wlanId"] = plan.WlanID.ValueString()
	}
	if !plan.LEDSetting.IsNull() && !plan.LEDSetting.IsUnknown() {
		rawConfig["ledSetting"] = plan.LEDSetting.ValueInt64()
	}
	if !plan.MVlanEnable.IsNull() && !plan.MVlanEnable.IsUnknown() {
		rawConfig["mvlanEnable"] = plan.MVlanEnable.ValueBool()
	}
	if !plan.LoopbackDetectEnable.IsNull() && !plan.LoopbackDetectEnable.IsUnknown() {
		rawConfig["loopbackDetectEnable"] = plan.LoopbackDetectEnable.ValueBool()
	}

	// Management VLAN
	if !plan.MVlanNetworkID.IsNull() && !plan.MVlanNetworkID.IsUnknown() {
		rawConfig["mvlanSetting"] = map[string]interface{}{
			"mode":         1,
			"lanNetworkId": plan.MVlanNetworkID.ValueString(),
		}
	}

	// IP setting
	if !plan.IPSettingMode.IsNull() && !plan.IPSettingMode.IsUnknown() {
		mode := plan.IPSettingMode.ValueString()
		if mode == "dhcp" {
			fallback := !plan.IPSettingFallback.IsNull() && !plan.IPSettingFallback.IsUnknown() && plan.IPSettingFallback.ValueBool()
			ipSetting := map[string]interface{}{
				"mode":         "dhcp",
				"fallback":     fallback,
				"useFixedAddr": false,
			}
			// Only include fallback IP fields when fallback is enabled.
			// The Omada API rejects empty strings for these IP fields.
			if fallback {
				if !plan.IPSettingFallbackIP.IsNull() && !plan.IPSettingFallbackIP.IsUnknown() {
					ipSetting["fallbackIp"] = plan.IPSettingFallbackIP.ValueString()
				}
				if !plan.IPSettingFallbackMask.IsNull() && !plan.IPSettingFallbackMask.IsUnknown() {
					ipSetting["fallbackMask"] = plan.IPSettingFallbackMask.ValueString()
				}
				if !plan.IPSettingFallbackGate.IsNull() && !plan.IPSettingFallbackGate.IsUnknown() {
					ipSetting["fallbackGate"] = plan.IPSettingFallbackGate.ValueString()
				}
			}
			rawConfig["ipSetting"] = ipSetting
		}
	}
}

// buildAPRadioConfig builds the radio config payload for PUT /eaps/{mac}/config/radios.
func buildAPRadioConfig(plan *DeviceAPResourceModel) *client.APRadioConfig {
	config := &client.APRadioConfig{}

	has2g := !plan.Radio2gEnable.IsNull() || !plan.Radio2gChannelWidth.IsNull() ||
		!plan.Radio2gChannel.IsNull() || !plan.Radio2gTxPower.IsNull() || !plan.Radio2gTxPowerLevel.IsNull()
	has5g := !plan.Radio5gEnable.IsNull() || !plan.Radio5gChannelWidth.IsNull() ||
		!plan.Radio5gChannel.IsNull() || !plan.Radio5gTxPower.IsNull() || !plan.Radio5gTxPowerLevel.IsNull()

	if has2g {
		r := &client.APRadioSetting{}
		if !plan.Radio2gEnable.IsNull() && !plan.Radio2gEnable.IsUnknown() {
			r.RadioEnable = plan.Radio2gEnable.ValueBool()
		}
		if !plan.Radio2gChannelWidth.IsNull() && !plan.Radio2gChannelWidth.IsUnknown() {
			r.ChannelWidth = plan.Radio2gChannelWidth.ValueString()
		}
		if !plan.Radio2gChannel.IsNull() && !plan.Radio2gChannel.IsUnknown() {
			r.Channel = plan.Radio2gChannel.ValueString()
		}
		if !plan.Radio2gTxPower.IsNull() && !plan.Radio2gTxPower.IsUnknown() {
			r.TxPower = int(plan.Radio2gTxPower.ValueInt64())
		}
		if !plan.Radio2gTxPowerLevel.IsNull() && !plan.Radio2gTxPowerLevel.IsUnknown() {
			r.TxPowerLevel = int(plan.Radio2gTxPowerLevel.ValueInt64())
		}
		config.RadioSetting2g = r
	}

	if has5g {
		r := &client.APRadioSetting{}
		if !plan.Radio5gEnable.IsNull() && !plan.Radio5gEnable.IsUnknown() {
			r.RadioEnable = plan.Radio5gEnable.ValueBool()
		}
		if !plan.Radio5gChannelWidth.IsNull() && !plan.Radio5gChannelWidth.IsUnknown() {
			r.ChannelWidth = plan.Radio5gChannelWidth.ValueString()
		}
		if !plan.Radio5gChannel.IsNull() && !plan.Radio5gChannel.IsUnknown() {
			r.Channel = plan.Radio5gChannel.ValueString()
		}
		if !plan.Radio5gTxPower.IsNull() && !plan.Radio5gTxPower.IsUnknown() {
			r.TxPower = int(plan.Radio5gTxPower.ValueInt64())
		}
		if !plan.Radio5gTxPowerLevel.IsNull() && !plan.Radio5gTxPowerLevel.IsUnknown() {
			r.TxPowerLevel = int(plan.Radio5gTxPowerLevel.ValueInt64())
		}
		config.RadioSetting5g = r
	}

	return config
}

// buildAPAdvancedConfig builds the advanced config payload for PUT /eaps/{mac}/config/advanced.
func buildAPAdvancedConfig(plan *DeviceAPResourceModel) *client.APAdvancedConfig {
	config := &client.APAdvancedConfig{}

	if !plan.OFDMAEnable2g.IsNull() && !plan.OFDMAEnable2g.IsUnknown() {
		v := plan.OFDMAEnable2g.ValueBool()
		config.OFDMAEnable2g = &v
	}
	if !plan.OFDMAEnable5g.IsNull() && !plan.OFDMAEnable5g.IsUnknown() {
		v := plan.OFDMAEnable5g.ValueBool()
		config.OFDMAEnable5g = &v
	}
	if !plan.LB2gEnable.IsNull() && !plan.LB2gEnable.IsUnknown() {
		lb := &client.APLBSetting{LBEnable: plan.LB2gEnable.ValueBool()}
		if !plan.LB2gMaxClients.IsNull() && !plan.LB2gMaxClients.IsUnknown() {
			lb.MaxClients = int(plan.LB2gMaxClients.ValueInt64())
		}
		config.LBSetting2g = lb
	}
	if !plan.LB5gEnable.IsNull() && !plan.LB5gEnable.IsUnknown() {
		lb := &client.APLBSetting{LBEnable: plan.LB5gEnable.ValueBool()}
		if !plan.LB5gMaxClients.IsNull() && !plan.LB5gMaxClients.IsUnknown() {
			lb.MaxClients = int(plan.LB5gMaxClients.ValueInt64())
		}
		config.LBSetting5g = lb
	}
	if !plan.RSSI2gEnable.IsNull() && !plan.RSSI2gEnable.IsUnknown() {
		rssi := &client.APRSSISetting{RSSIEnable: plan.RSSI2gEnable.ValueBool()}
		if !plan.RSSI2gThreshold.IsNull() && !plan.RSSI2gThreshold.IsUnknown() {
			rssi.Threshold = int(plan.RSSI2gThreshold.ValueInt64())
		}
		config.RSSISetting2g = rssi
	}
	if !plan.RSSI5gEnable.IsNull() && !plan.RSSI5gEnable.IsUnknown() {
		rssi := &client.APRSSISetting{RSSIEnable: plan.RSSI5gEnable.ValueBool()}
		if !plan.RSSI5gThreshold.IsNull() && !plan.RSSI5gThreshold.IsUnknown() {
			rssi.Threshold = int(plan.RSSI5gThreshold.ValueInt64())
		}
		config.RSSISetting5g = rssi
	}

	return config
}

// buildAPServicesConfig builds the services config payload for PUT /eaps/{mac}/config/services.
func buildAPServicesConfig(plan *DeviceAPResourceModel) *client.APServicesConfig {
	config := &client.APServicesConfig{}

	if !plan.LLDPEnable.IsNull() && !plan.LLDPEnable.IsUnknown() {
		v := int(plan.LLDPEnable.ValueInt64())
		config.LLDPEnable = &v
	}
	if !plan.L3AccessEnable.IsNull() && !plan.L3AccessEnable.IsUnknown() {
		config.L3AccessSetting = &client.APL3AccessSetting{Enable: plan.L3AccessEnable.ValueBool()}
	}

	return config
}

// applyAPConfig applies all plan changes across the main PATCH and three dedicated endpoints.
func (r *DeviceAPResource) applyAPConfig(ctx context.Context, mac string, plan *DeviceAPResourceModel) error {
	// 1. Main PATCH — name, wlanId, ledSetting, ipSetting, mvlanEnable, loopbackDetectEnable
	rawConfig, err := r.client.GetAPConfigRaw(ctx, mac)
	if err != nil {
		return fmt.Errorf("reading AP config: %w", err)
	}
	applyAPPlanToRaw(rawConfig, plan)
	if _, err := r.client.UpdateAPConfig(ctx, mac, rawConfig); err != nil {
		return fmt.Errorf("updating AP config: %w", err)
	}

	// 2. Radio settings — PUT /eaps/{mac}/config/radios
	if err := r.client.UpdateAPRadioConfig(ctx, mac, buildAPRadioConfig(plan)); err != nil {
		return fmt.Errorf("updating AP radio config: %w", err)
	}

	// 3. Advanced settings — PUT /eaps/{mac}/config/advanced
	// (OFDMA, load balancing, RSSI — silently ignored by main PATCH)
	if err := r.client.UpdateAPAdvancedConfig(ctx, mac, buildAPAdvancedConfig(plan)); err != nil {
		return fmt.Errorf("updating AP advanced config: %w", err)
	}

	// 4. Services settings — PUT /eaps/{mac}/config/services
	// (LLDP, L3 access — silently ignored by main PATCH)
	if err := r.client.UpdateAPServicesConfig(ctx, mac, buildAPServicesConfig(plan)); err != nil {
		return fmt.Errorf("updating AP services config: %w", err)
	}

	return nil
}

func (r *DeviceAPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DeviceAPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mac := plan.MAC.ValueString()

	if err := r.applyAPConfig(ctx, mac, &plan); err != nil {
		resp.Diagnostics.AddError("Error creating AP config", err.Error())
		return
	}

	apConfig, err := r.client.GetAPConfig(ctx, mac)
	if err != nil {
		resp.Diagnostics.AddError("Error reading AP config after create", err.Error())
		return
	}

	apConfigToState(apConfig, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DeviceAPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeviceAPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apConfig, err := r.client.GetAPConfig(ctx, state.MAC.ValueString())
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

	mac := plan.MAC.ValueString()

	if err := r.applyAPConfig(ctx, mac, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating AP config", err.Error())
		return
	}

	apConfig, err := r.client.GetAPConfig(ctx, mac)
	if err != nil {
		resp.Diagnostics.AddError("Error reading AP config after update", err.Error())
		return
	}

	apConfigToState(apConfig, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete for an AP just removes from state — can't unadopt via API.
func (r *DeviceAPResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: APs cannot be deleted/unadopted via the API.
	// Removing from Terraform state is sufficient.
}

func (r *DeviceAPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	mac := req.ID

	apConfig, err := r.client.GetAPConfig(ctx, mac)
	if err != nil {
		resp.Diagnostics.AddError("Error importing AP config",
			fmt.Sprintf("Could not read AP with MAC %q: %s", mac, err.Error()))
		return
	}

	var state DeviceAPResourceModel
	state.MAC = types.StringValue(mac)
	apConfigToState(apConfig, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// apConfigToState maps an APConfig from the API to the Terraform state model.
func apConfigToState(cfg *client.APConfig, state *DeviceAPResourceModel) {
	state.MAC = types.StringValue(cfg.MAC)
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
		state.IPSettingFallback = types.BoolValue(cfg.IPSetting.Fallback)
		if cfg.IPSetting.Fallback {
			state.IPSettingFallbackIP = types.StringValue(cfg.IPSetting.FallbackIP)
			state.IPSettingFallbackMask = types.StringValue(cfg.IPSetting.FallbackMask)
			state.IPSettingFallbackGate = types.StringValue(cfg.IPSetting.FallbackGate)
		} else {
			state.IPSettingFallbackIP = types.StringNull()
			state.IPSettingFallbackMask = types.StringNull()
			state.IPSettingFallbackGate = types.StringNull()
		}
	} else {
		state.IPSettingMode = types.StringValue("dhcp")
		state.IPSettingFallback = types.BoolValue(false)
		state.IPSettingFallbackIP = types.StringNull()
		state.IPSettingFallbackMask = types.StringNull()
		state.IPSettingFallbackGate = types.StringNull()
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
