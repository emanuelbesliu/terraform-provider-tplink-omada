package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-tplink-omada/internal/client"
)

var _ resource.Resource = &DeviceSwitchResource{}
var _ resource.ResourceWithImportState = &DeviceSwitchResource{}

// DeviceSwitchResource manages an Omada switch device configuration.
type DeviceSwitchResource struct {
	client *client.Client
}

// SwitchPortModel maps a single switch port to Terraform state.
type SwitchPortModel struct {
	Port                  types.Int64  `tfsdk:"port"`
	Name                  types.String `tfsdk:"name"`
	Disable               types.Bool   `tfsdk:"disable"`
	ProfileID             types.String `tfsdk:"profile_id"`
	ProfileOverrideEnable types.Bool   `tfsdk:"profile_override_enable"`
	NativeNetworkID       types.String `tfsdk:"native_network_id"`
	NetworkTagsSetting    types.Int64  `tfsdk:"network_tags_setting"`
	TagNetworkIDs         types.List   `tfsdk:"tag_network_ids"`
	UntagNetworkIDs       types.List   `tfsdk:"untag_network_ids"`
	Speed                 types.Int64  `tfsdk:"speed"`
	VoiceNetworkEnable    types.Bool   `tfsdk:"voice_network_enable"`
	VoiceDscpEnable       types.Bool   `tfsdk:"voice_dscp_enable"`
}

// DeviceSwitchResourceModel maps the resource schema to Go types.
type DeviceSwitchResourceModel struct {
	// Identity — ID is the MAC address, set via import
	ID     types.String `tfsdk:"id"`
	SiteID types.String `tfsdk:"site_id"`

	// Configurable top-level fields
	Name                  types.String `tfsdk:"name"`
	LEDSetting            types.Int64  `tfsdk:"led_setting"`
	MVlanNetworkID        types.String `tfsdk:"management_vlan_network_id"`
	IPSettingMode         types.String `tfsdk:"ip_setting_mode"`
	IPSettingFallback     types.Bool   `tfsdk:"ip_setting_fallback"`
	IPSettingFallbackIP   types.String `tfsdk:"ip_setting_fallback_ip"`
	IPSettingFallbackMask types.String `tfsdk:"ip_setting_fallback_mask"`
	LoopbackDetectEnable  types.Bool   `tfsdk:"loopback_detect_enable"`

	// STP
	STP          types.Int64 `tfsdk:"stp"`
	Priority     types.Int64 `tfsdk:"stp_priority"`
	HelloTime    types.Int64 `tfsdk:"stp_hello_time"`
	MaxAge       types.Int64 `tfsdk:"stp_max_age"`
	ForwardDelay types.Int64 `tfsdk:"stp_forward_delay"`
	TxHoldCount  types.Int64 `tfsdk:"stp_tx_hold_count"`
	MaxHops      types.Int64 `tfsdk:"stp_max_hops"`

	// SNMP
	SNMPLocation types.String `tfsdk:"snmp_location"`
	SNMPContact  types.String `tfsdk:"snmp_contact"`

	// Misc
	Jumbo      types.Int64 `tfsdk:"jumbo"`
	LagHashAlg types.Int64 `tfsdk:"lag_hash_alg"`

	// Ports
	Ports types.List `tfsdk:"ports"`

	// Read-only
	Model           types.String `tfsdk:"model"`
	IP              types.String `tfsdk:"ip"`
	FirmwareVersion types.String `tfsdk:"firmware_version"`
}

func NewDeviceSwitchResource() resource.Resource {
	return &DeviceSwitchResource{}
}

func (r *DeviceSwitchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_switch"
}

var switchPortSchema = schema.NestedAttributeObject{
	Attributes: map[string]schema.Attribute{
		"port": schema.Int64Attribute{
			Description: "Port number (1-based).",
			Required:    true,
		},
		"name": schema.StringAttribute{
			Description: "Port display name.",
			Optional:    true,
			Computed:    true,
		},
		"disable": schema.BoolAttribute{
			Description: "Whether the port is administratively disabled.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
		"profile_id": schema.StringAttribute{
			Description: "The port profile ID assigned to this port.",
			Optional:    true,
			Computed:    true,
		},
		"profile_override_enable": schema.BoolAttribute{
			Description: "Whether per-port override of the profile is enabled.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
		"native_network_id": schema.StringAttribute{
			Description: "The native (untagged) network ID for this port.",
			Optional:    true,
			Computed:    true,
		},
		"network_tags_setting": schema.Int64Attribute{
			Description: "Network tags setting: 0=from profile, 1=custom.",
			Optional:    true,
			Computed:    true,
		},
		"tag_network_ids": schema.ListAttribute{
			Description: "List of tagged network IDs.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
		},
		"untag_network_ids": schema.ListAttribute{
			Description: "List of untagged network IDs.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
		},
		"speed": schema.Int64Attribute{
			Description: "Port speed: 0=Auto, 1=10M, 2=100M, 3=1000M.",
			Optional:    true,
			Computed:    true,
		},
		"voice_network_enable": schema.BoolAttribute{
			Description: "Enable voice network on this port.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
		"voice_dscp_enable": schema.BoolAttribute{
			Description: "Enable voice DSCP on this port.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
	},
}

func (r *DeviceSwitchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the configuration of an Omada managed switch. " +
			"Switches must be adopted through the controller UI before they can be managed. " +
			"Use 'terraform import omada_device_switch.<name> <siteID>/<mac>' to bring an adopted switch into state.",
		Attributes: map[string]schema.Attribute{
			// Identity — set by import, not user config
			"id": schema.StringAttribute{
				Description: "The switch MAC address (set by import).",
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
				Description: "The display name of the switch.",
				Optional:    true,
				Computed:    true,
			},
			"led_setting": schema.Int64Attribute{
				Description: "LED setting: 0=Off, 1=On, 2=Follow site setting.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"management_vlan_network_id": schema.StringAttribute{
				Description: "The LAN network ID used for management VLAN.",
				Optional:    true,
				Computed:    true,
			},

			// IP Settings
			"ip_setting_mode": schema.StringAttribute{
				Description: "IP address mode: 'dhcp' or 'static'.",
				Optional:    true,
				Computed:    true,
			},
			"ip_setting_fallback": schema.BoolAttribute{
				Description: "Enable fallback IP when DHCP fails.",
				Optional:    true,
				Computed:    true,
			},
			"ip_setting_fallback_ip": schema.StringAttribute{
				Description: "Fallback IP address.",
				Optional:    true,
				Computed:    true,
			},
			"ip_setting_fallback_mask": schema.StringAttribute{
				Description: "Fallback subnet mask.",
				Optional:    true,
				Computed:    true,
			},

			"loopback_detect_enable": schema.BoolAttribute{
				Description: "Enable loopback detection.",
				Optional:    true,
				Computed:    true,
			},

			// STP
			"stp": schema.Int64Attribute{
				Description: "STP mode: 2=Follow site setting.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"stp_priority": schema.Int64Attribute{
				Description: "STP bridge priority (default 32768).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(32768),
			},
			"stp_hello_time": schema.Int64Attribute{
				Description: "STP hello time in seconds (default 2).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"stp_max_age": schema.Int64Attribute{
				Description: "STP max age in seconds (default 20).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20),
			},
			"stp_forward_delay": schema.Int64Attribute{
				Description: "STP forward delay in seconds (default 15).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(15),
			},
			"stp_tx_hold_count": schema.Int64Attribute{
				Description: "STP TX hold count (default 5).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5),
			},
			"stp_max_hops": schema.Int64Attribute{
				Description: "STP max hops (default 20).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20),
			},

			// SNMP
			"snmp_location": schema.StringAttribute{
				Description: "SNMP location string.",
				Optional:    true,
				Computed:    true,
			},
			"snmp_contact": schema.StringAttribute{
				Description: "SNMP contact string.",
				Optional:    true,
				Computed:    true,
			},

			// Misc
			"jumbo": schema.Int64Attribute{
				Description: "Jumbo frame size (default 1518).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1518),
			},
			"lag_hash_alg": schema.Int64Attribute{
				Description: "LAG hash algorithm.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},

			// Ports
			"ports": schema.ListNestedAttribute{
				Description:  "Switch port configurations. All ports must be specified on import.",
				Optional:     true,
				Computed:     true,
				NestedObject: switchPortSchema,
			},

			// Read-only
			"model": schema.StringAttribute{
				Description: "The switch model (e.g., 'TL-SG3428MP'). Read-only.",
				Computed:    true,
			},
			"ip": schema.StringAttribute{
				Description: "The switch IP address. Read-only.",
				Computed:    true,
			},
			"firmware_version": schema.StringAttribute{
				Description: "The switch firmware version. Read-only.",
				Computed:    true,
			},
		},
	}
}

func (r *DeviceSwitchResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create always fails — switches must be adopted through the controller UI
// and imported into Terraform state.
func (r *DeviceSwitchResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"Cannot create switch device",
		"Switches must be adopted through the Omada controller UI, then imported into Terraform:\n\n"+
			"  terraform import omada_device_switch.<name> <siteID>/<mac>",
	)
}

func (r *DeviceSwitchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeviceSwitchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	swConfig, err := r.client.GetSwitchConfig(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading switch config", err.Error())
		return
	}

	switchConfigToState(swConfig, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DeviceSwitchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeviceSwitchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DeviceSwitchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mac := state.ID.ValueString()
	siteID := state.SiteID.ValueString()

	rawConfig, err := r.client.GetSwitchConfigRaw(ctx, siteID, mac)
	if err != nil {
		resp.Diagnostics.AddError("Error reading switch config for update", err.Error())
		return
	}

	dummy := true
	applySwitchPlanToRaw(rawConfig, &plan, &dummy)

	_, err = r.client.UpdateSwitchConfig(ctx, siteID, mac, rawConfig)
	if err != nil {
		resp.Diagnostics.AddError("Error updating switch config", err.Error())
		return
	}

	swConfig, err := r.client.GetSwitchConfig(ctx, siteID, mac)
	if err != nil {
		resp.Diagnostics.AddError("Error reading switch config after update", err.Error())
		return
	}

	switchConfigToState(swConfig, &plan)
	plan.SiteID = state.SiteID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete for a switch just removes from state.
func (r *DeviceSwitchResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: Switches cannot be deleted/unadopted via the API.
}

func (r *DeviceSwitchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	siteID, mac, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'siteID/mac'.",
		)
		return
	}

	swConfig, err := r.client.GetSwitchConfig(ctx, siteID, mac)
	if err != nil {
		resp.Diagnostics.AddError("Error importing switch config",
			fmt.Sprintf("Could not read switch with MAC %q: %s", mac, err.Error()))
		return
	}

	var state DeviceSwitchResourceModel
	state.SiteID = types.StringValue(siteID)
	switchConfigToState(swConfig, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// switchConfigToState maps a SwitchConfig from the API to the Terraform state model.
func switchConfigToState(cfg *client.SwitchConfig, state *DeviceSwitchResourceModel) {
	state.ID = types.StringValue(cfg.MAC)
	state.Name = types.StringValue(cfg.Name)
	state.LEDSetting = types.Int64Value(int64(cfg.LEDSetting))
	state.MVlanNetworkID = types.StringValue(cfg.MVlanNetworkID)

	// IP Settings
	if cfg.IPSetting != nil {
		state.IPSettingMode = types.StringValue(cfg.IPSetting.Mode)
		state.IPSettingFallback = types.BoolValue(cfg.IPSetting.Fallback)
		state.IPSettingFallbackIP = types.StringValue(cfg.IPSetting.FallbackIP)
		state.IPSettingFallbackMask = types.StringValue(cfg.IPSetting.FallbackMask)
	} else {
		state.IPSettingMode = types.StringValue("dhcp")
		state.IPSettingFallback = types.BoolValue(false)
		state.IPSettingFallbackIP = types.StringValue("")
		state.IPSettingFallbackMask = types.StringValue("")
	}

	state.LoopbackDetectEnable = types.BoolValue(cfg.LoopbackDetectEnable)

	// STP
	state.STP = types.Int64Value(int64(cfg.STP))
	state.Priority = types.Int64Value(int64(cfg.Priority))
	state.HelloTime = types.Int64Value(int64(cfg.HelloTime))
	state.MaxAge = types.Int64Value(int64(cfg.MaxAge))
	state.ForwardDelay = types.Int64Value(int64(cfg.ForwardDelay))
	state.TxHoldCount = types.Int64Value(int64(cfg.TxHoldCount))
	state.MaxHops = types.Int64Value(int64(cfg.MaxHops))

	// SNMP
	if cfg.SNMP != nil {
		state.SNMPLocation = types.StringValue(cfg.SNMP.Location)
		state.SNMPContact = types.StringValue(cfg.SNMP.Contact)
	} else {
		state.SNMPLocation = types.StringValue("")
		state.SNMPContact = types.StringValue("")
	}

	// Misc
	state.Jumbo = types.Int64Value(int64(cfg.Jumbo))
	state.LagHashAlg = types.Int64Value(int64(cfg.LagHashAlg))

	// Ports
	portValues := make([]attr.Value, len(cfg.Ports))
	for i, p := range cfg.Ports {
		tagIDs := make([]attr.Value, len(p.TagNetworkIDs))
		for j, id := range p.TagNetworkIDs {
			tagIDs[j] = types.StringValue(id)
		}
		untagIDs := make([]attr.Value, len(p.UntagNetworkIDs))
		for j, id := range p.UntagNetworkIDs {
			untagIDs[j] = types.StringValue(id)
		}

		tagList, _ := types.ListValue(types.StringType, tagIDs)
		untagList, _ := types.ListValue(types.StringType, untagIDs)

		portObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"port":                    types.Int64Type,
				"name":                    types.StringType,
				"disable":                 types.BoolType,
				"profile_id":              types.StringType,
				"profile_override_enable": types.BoolType,
				"native_network_id":       types.StringType,
				"network_tags_setting":    types.Int64Type,
				"tag_network_ids":         types.ListType{ElemType: types.StringType},
				"untag_network_ids":       types.ListType{ElemType: types.StringType},
				"speed":                   types.Int64Type,
				"voice_network_enable":    types.BoolType,
				"voice_dscp_enable":       types.BoolType,
			},
			map[string]attr.Value{
				"port":                    types.Int64Value(int64(p.Port)),
				"name":                    types.StringValue(p.Name),
				"disable":                 types.BoolValue(p.Disable),
				"profile_id":              types.StringValue(p.ProfileID),
				"profile_override_enable": types.BoolValue(p.ProfileOverrideEnable),
				"native_network_id":       types.StringValue(p.NativeNetworkID),
				"network_tags_setting":    types.Int64Value(int64(p.NetworkTagsSetting)),
				"tag_network_ids":         tagList,
				"untag_network_ids":       untagList,
				"speed":                   types.Int64Value(int64(p.Speed)),
				"voice_network_enable":    types.BoolValue(p.VoiceNetworkEnable),
				"voice_dscp_enable":       types.BoolValue(p.VoiceDscpEnable),
			},
		)
		portValues[i] = portObj
	}

	portAttrTypes := map[string]attr.Type{
		"port":                    types.Int64Type,
		"name":                    types.StringType,
		"disable":                 types.BoolType,
		"profile_id":              types.StringType,
		"profile_override_enable": types.BoolType,
		"native_network_id":       types.StringType,
		"network_tags_setting":    types.Int64Type,
		"tag_network_ids":         types.ListType{ElemType: types.StringType},
		"untag_network_ids":       types.ListType{ElemType: types.StringType},
		"speed":                   types.Int64Type,
		"voice_network_enable":    types.BoolType,
		"voice_dscp_enable":       types.BoolType,
	}

	portsList, _ := types.ListValue(
		types.ObjectType{AttrTypes: portAttrTypes},
		portValues,
	)
	state.Ports = portsList

	// Read-only
	state.Model = types.StringValue(cfg.Model)
	state.IP = types.StringValue(cfg.IP)
	state.FirmwareVersion = types.StringValue(cfg.FirmwareVersion)
}

// applySwitchPlanToRaw applies Terraform plan values to a raw JSON switch config.
func applySwitchPlanToRaw(raw map[string]interface{}, plan *DeviceSwitchResourceModel, needsUpdate *bool) {
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		raw["name"] = plan.Name.ValueString()
		*needsUpdate = true
	}
	if !plan.LEDSetting.IsNull() && !plan.LEDSetting.IsUnknown() {
		raw["ledSetting"] = plan.LEDSetting.ValueInt64()
		*needsUpdate = true
	}
	if !plan.MVlanNetworkID.IsNull() && !plan.MVlanNetworkID.IsUnknown() {
		raw["mvlanNetworkId"] = plan.MVlanNetworkID.ValueString()
		*needsUpdate = true
	}

	// IP Settings
	if !plan.IPSettingMode.IsNull() && !plan.IPSettingMode.IsUnknown() {
		ipSetting, ok := raw["ipSetting"].(map[string]interface{})
		if !ok {
			ipSetting = map[string]interface{}{}
			raw["ipSetting"] = ipSetting
		}
		ipSetting["mode"] = plan.IPSettingMode.ValueString()
		if !plan.IPSettingFallback.IsNull() && !plan.IPSettingFallback.IsUnknown() {
			ipSetting["fallback"] = plan.IPSettingFallback.ValueBool()
		}
		if !plan.IPSettingFallbackIP.IsNull() && !plan.IPSettingFallbackIP.IsUnknown() {
			ipSetting["fallbackIp"] = plan.IPSettingFallbackIP.ValueString()
		}
		if !plan.IPSettingFallbackMask.IsNull() && !plan.IPSettingFallbackMask.IsUnknown() {
			ipSetting["fallbackMask"] = plan.IPSettingFallbackMask.ValueString()
		}
		*needsUpdate = true
	}

	if !plan.LoopbackDetectEnable.IsNull() && !plan.LoopbackDetectEnable.IsUnknown() {
		raw["loopbackDetectEnable"] = plan.LoopbackDetectEnable.ValueBool()
		*needsUpdate = true
	}

	// STP
	if !plan.STP.IsNull() && !plan.STP.IsUnknown() {
		raw["stp"] = plan.STP.ValueInt64()
		*needsUpdate = true
	}
	if !plan.Priority.IsNull() && !plan.Priority.IsUnknown() {
		raw["priority"] = plan.Priority.ValueInt64()
		*needsUpdate = true
	}
	if !plan.HelloTime.IsNull() && !plan.HelloTime.IsUnknown() {
		raw["helloTime"] = plan.HelloTime.ValueInt64()
		*needsUpdate = true
	}
	if !plan.MaxAge.IsNull() && !plan.MaxAge.IsUnknown() {
		raw["maxAge"] = plan.MaxAge.ValueInt64()
		*needsUpdate = true
	}
	if !plan.ForwardDelay.IsNull() && !plan.ForwardDelay.IsUnknown() {
		raw["forwardDelay"] = plan.ForwardDelay.ValueInt64()
		*needsUpdate = true
	}
	if !plan.TxHoldCount.IsNull() && !plan.TxHoldCount.IsUnknown() {
		raw["txHoldCount"] = plan.TxHoldCount.ValueInt64()
		*needsUpdate = true
	}
	if !plan.MaxHops.IsNull() && !plan.MaxHops.IsUnknown() {
		raw["maxHops"] = plan.MaxHops.ValueInt64()
		*needsUpdate = true
	}

	// SNMP
	if !plan.SNMPLocation.IsNull() && !plan.SNMPLocation.IsUnknown() {
		snmp, ok := raw["snmp"].(map[string]interface{})
		if !ok {
			snmp = map[string]interface{}{}
			raw["snmp"] = snmp
		}
		snmp["location"] = plan.SNMPLocation.ValueString()
		if !plan.SNMPContact.IsNull() && !plan.SNMPContact.IsUnknown() {
			snmp["contact"] = plan.SNMPContact.ValueString()
		}
		*needsUpdate = true
	}

	// Misc
	if !plan.Jumbo.IsNull() && !plan.Jumbo.IsUnknown() {
		raw["jumbo"] = plan.Jumbo.ValueInt64()
		*needsUpdate = true
	}
	if !plan.LagHashAlg.IsNull() && !plan.LagHashAlg.IsUnknown() {
		raw["lagHashAlg"] = plan.LagHashAlg.ValueInt64()
		*needsUpdate = true
	}
}
