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

var _ resource.Resource = &WirelessNetworkResource{}
var _ resource.ResourceWithImportState = &WirelessNetworkResource{}

// WirelessNetworkResource manages an Omada SSID/WLAN.
type WirelessNetworkResource struct {
	client *client.Client
}

// WirelessNetworkResourceModel maps the resource schema to Go types.
type WirelessNetworkResourceModel struct {
	ID          types.String `tfsdk:"id"`
	SiteID      types.String `tfsdk:"site_id"`
	WlanGroupID types.String `tfsdk:"wlan_group_id"`
	Name        types.String `tfsdk:"name"`
	Band        types.Int64  `tfsdk:"band"`
	Security    types.Int64  `tfsdk:"security"`
	Passphrase  types.String `tfsdk:"passphrase"`
	Broadcast   types.Bool   `tfsdk:"broadcast"`
	VlanID      types.Int64  `tfsdk:"vlan_id"`
	Enable11r   types.Bool   `tfsdk:"enable_11r"`
	PmfMode     types.Int64  `tfsdk:"pmf_mode"`
}

func NewWirelessNetworkResource() resource.Resource {
	return &WirelessNetworkResource{}
}

func (r *WirelessNetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wireless_network"
}

func (r *WirelessNetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a wireless network (SSID) on the Omada Controller.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the SSID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": siteIDResourceSchema(),
			"wlan_group_id": schema.StringAttribute{
				Description: "The WLAN group ID. If not set, the default WLAN group is used.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The SSID name (broadcast name).",
				Required:    true,
			},
			"band": schema.Int64Attribute{
				Description: "Radio band bitmask: 1=2.4GHz, 2=5GHz, 3=both.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3),
			},
			"security": schema.Int64Attribute{
				Description: "Security mode: 0=Open, 3=WPA2/WPA3. Note: do NOT use 2 (WPA2-only fails on v6).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3),
			},
			"passphrase": schema.StringAttribute{
				Description: "The Wi-Fi password (WPA pre-shared key). Required when security > 0.",
				Optional:    true,
				Sensitive:   true,
			},
			"broadcast": schema.BoolAttribute{
				Description: "Whether the SSID is broadcast (visible). Set false for hidden networks.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"vlan_id": schema.Int64Attribute{
				Description: "Custom VLAN ID to assign to this SSID. If not set, uses the default VLAN.",
				Optional:    true,
			},
			"enable_11r": schema.BoolAttribute{
				Description: "Enable 802.11r Fast Roaming.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"pmf_mode": schema.Int64Attribute{
				Description: "Protected Management Frames mode: 1=disabled, 2=optional, 3=required.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
		},
	}
}

func (r *WirelessNetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WirelessNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WirelessNetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := plan.SiteID.ValueString()

	// Resolve WLAN group ID
	wlanGroupID := plan.WlanGroupID.ValueString()
	if wlanGroupID == "" {
		gid, err := r.client.GetDefaultWlanGroupID(ctx, siteID)
		if err != nil {
			resp.Diagnostics.AddError("Error getting default WLAN group", err.Error())
			return
		}
		wlanGroupID = gid
	}

	ssid := &client.WirelessNetwork{
		Name:               plan.Name.ValueString(),
		Band:               int(plan.Band.ValueInt64()),
		Security:           int(plan.Security.ValueInt64()),
		Broadcast:          plan.Broadcast.ValueBool(),
		Enable11r:          plan.Enable11r.ValueBool(),
		PmfMode:            int(plan.PmfMode.ValueInt64()),
		WlanScheduleEnable: false,
		MacFilterEnable:    false,
		RateLimit:          &client.RateLimit{},
		SSIDRateLimit:      &client.RateLimit{},
		RateAndBeaconCtrl:  &client.RateAndBeaconCtrl{},
		MultiCastSetting: &client.MultiCastSetting{
			MultiCastEnable: true,
			ChannelUtil:     100,
			ArpCastEnable:   true,
			Ipv6CastEnable:  true,
		},
	}

	if !plan.Passphrase.IsNull() && !plan.Passphrase.IsUnknown() {
		ssid.PSKSetting = &client.PSKSetting{
			VersionPsk:    2,
			EncryptionPsk: 3,
			SecurityKey:   plan.Passphrase.ValueString(),
		}
	}

	if !plan.VlanID.IsNull() && !plan.VlanID.IsUnknown() {
		vlanID := int(plan.VlanID.ValueInt64())
		customConfig := &client.CustomConfig{
			BridgeVlan: vlanID,
		}
		// Look up the network ID for this VLAN to populate lanNetworkId
		// (required by the Omada API for SSID creation).
		networks, err := r.client.ListNetworks(ctx, siteID)
		if err == nil {
			for _, n := range networks {
				if n.Vlan == vlanID {
					customConfig.LanNetworkID = n.ID
					customConfig.LanNetworkVlanIds = map[string][]int{
						n.ID: {vlanID},
					}
					break
				}
			}
		}
		ssid.VlanSetting = &client.VlanSetting{
			Mode:         1,
			CustomConfig: customConfig,
		}
	} else {
		ssid.VlanSetting = &client.VlanSetting{Mode: 0}
	}

	created, err := r.client.CreateWirelessNetwork(ctx, siteID, wlanGroupID, ssid)
	if err != nil {
		resp.Diagnostics.AddError("Error creating wireless network", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.WlanGroupID = types.StringValue(wlanGroupID)
	plan.Band = types.Int64Value(int64(created.Band))
	plan.Security = types.Int64Value(int64(created.Security))
	plan.Broadcast = types.BoolValue(created.Broadcast)
	plan.Enable11r = types.BoolValue(created.Enable11r)
	plan.PmfMode = types.Int64Value(int64(created.PmfMode))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WirelessNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WirelessNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()
	wlanGroupID := state.WlanGroupID.ValueString()

	ssid, err := r.client.GetWirelessNetwork(ctx, siteID, wlanGroupID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading wireless network", err.Error())
		return
	}

	state.Name = types.StringValue(ssid.Name)
	state.Band = types.Int64Value(int64(ssid.Band))
	state.Security = types.Int64Value(int64(ssid.Security))
	state.Broadcast = types.BoolValue(ssid.Broadcast)
	state.Enable11r = types.BoolValue(ssid.Enable11r)
	state.PmfMode = types.Int64Value(int64(ssid.PmfMode))

	// Read vlan_id from VlanSetting
	if ssid.VlanSetting != nil && ssid.VlanSetting.CustomConfig != nil && ssid.VlanSetting.CustomConfig.BridgeVlan != 0 {
		state.VlanID = types.Int64Value(int64(ssid.VlanSetting.CustomConfig.BridgeVlan))
	} else if ssid.VlanSetting != nil && ssid.VlanSetting.CurrentVlanId != 0 {
		state.VlanID = types.Int64Value(int64(ssid.VlanSetting.CurrentVlanId))
	}

	// Read passphrase from PSKSetting — the v6 API does return securityKey
	if ssid.PSKSetting != nil && ssid.PSKSetting.SecurityKey != "" {
		state.Passphrase = types.StringValue(ssid.PSKSetting.SecurityKey)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WirelessNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WirelessNetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WirelessNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()
	wlanGroupID := state.WlanGroupID.ValueString()
	ssidID := state.ID.ValueString()

	// Fetch the full raw SSID object (PATCH requires the complete object)
	rawSSID, err := r.client.GetWirelessNetworkRaw(ctx, siteID, wlanGroupID, ssidID)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching wireless network for update", err.Error())
		return
	}

	// Apply changes from the plan onto the raw object
	rawSSID["name"] = plan.Name.ValueString()
	rawSSID["band"] = int(plan.Band.ValueInt64())
	rawSSID["security"] = int(plan.Security.ValueInt64())
	rawSSID["broadcast"] = plan.Broadcast.ValueBool()
	rawSSID["enable11r"] = plan.Enable11r.ValueBool()
	rawSSID["pmfMode"] = int(plan.PmfMode.ValueInt64())

	if !plan.Passphrase.IsNull() && !plan.Passphrase.IsUnknown() {
		rawSSID["pskSetting"] = map[string]interface{}{
			"versionPsk":    2,
			"encryptionPsk": 3,
			"securityKey":   plan.Passphrase.ValueString(),
		}
	}

	if !plan.VlanID.IsNull() && !plan.VlanID.IsUnknown() {
		rawSSID["vlanSetting"] = map[string]interface{}{
			"mode": 1,
			"customConfig": map[string]interface{}{
				"bridgeVlan": int(plan.VlanID.ValueInt64()),
			},
		}
	}

	updated, err := r.client.UpdateWirelessNetwork(ctx, siteID, wlanGroupID, ssidID, rawSSID)
	if err != nil {
		resp.Diagnostics.AddError("Error updating wireless network", err.Error())
		return
	}

	plan.ID = state.ID
	plan.SiteID = state.SiteID
	plan.WlanGroupID = state.WlanGroupID
	plan.Band = types.Int64Value(int64(updated.Band))
	plan.Security = types.Int64Value(int64(updated.Security))
	plan.Broadcast = types.BoolValue(updated.Broadcast)
	plan.Enable11r = types.BoolValue(updated.Enable11r)
	plan.PmfMode = types.Int64Value(int64(updated.PmfMode))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WirelessNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WirelessNetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWirelessNetwork(ctx, state.SiteID.ValueString(), state.WlanGroupID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting wireless network", err.Error())
		return
	}
}

func (r *WirelessNetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "siteID/wlanGroupID/ssidID"
	siteID, wlanGroupID, ssidID, ok := parseImportID3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'siteID/wlanGroupID/ssidID'.",
		)
		return
	}

	ssid, err := r.client.GetWirelessNetwork(ctx, siteID, wlanGroupID, ssidID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing wireless network", err.Error())
		return
	}

	state := WirelessNetworkResourceModel{
		ID:          types.StringValue(ssid.ID),
		SiteID:      types.StringValue(siteID),
		WlanGroupID: types.StringValue(wlanGroupID),
		Name:        types.StringValue(ssid.Name),
		Band:        types.Int64Value(int64(ssid.Band)),
		Security:    types.Int64Value(int64(ssid.Security)),
		Broadcast:   types.BoolValue(ssid.Broadcast),
		Enable11r:   types.BoolValue(ssid.Enable11r),
		PmfMode:     types.Int64Value(int64(ssid.PmfMode)),
	}

	// Read vlan_id from VlanSetting
	if ssid.VlanSetting != nil && ssid.VlanSetting.CustomConfig != nil && ssid.VlanSetting.CustomConfig.BridgeVlan != 0 {
		state.VlanID = types.Int64Value(int64(ssid.VlanSetting.CustomConfig.BridgeVlan))
	} else if ssid.VlanSetting != nil && ssid.VlanSetting.CurrentVlanId != 0 {
		state.VlanID = types.Int64Value(int64(ssid.VlanSetting.CurrentVlanId))
	} else {
		state.VlanID = types.Int64Null()
	}

	// Read passphrase from PSKSetting — the v6 API returns securityKey
	if ssid.PSKSetting != nil && ssid.PSKSetting.SecurityKey != "" {
		state.Passphrase = types.StringValue(ssid.PSKSetting.SecurityKey)
	} else {
		state.Passphrase = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
