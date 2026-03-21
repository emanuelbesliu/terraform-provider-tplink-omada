package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-tplink-omada/internal/client"
)

var _ resource.Resource = &NetworkResource{}
var _ resource.ResourceWithImportState = &NetworkResource{}

// NetworkResource manages an Omada LAN network / VLAN.
type NetworkResource struct {
	client *client.Client
}

// NetworkResourceModel maps the resource schema to Go types.
type NetworkResourceModel struct {
	ID            types.String `tfsdk:"id"`
	SiteID        types.String `tfsdk:"site_id"`
	Name          types.String `tfsdk:"name"`
	Purpose       types.String `tfsdk:"purpose"`
	VlanID        types.Int64  `tfsdk:"vlan_id"`
	GatewaySubnet types.String `tfsdk:"gateway_subnet"`
	DHCPEnabled   types.Bool   `tfsdk:"dhcp_enabled"`
	DHCPStart     types.String `tfsdk:"dhcp_start"`
	DHCPEnd       types.String `tfsdk:"dhcp_end"`
}

func NewNetworkResource() resource.Resource {
	return &NetworkResource{}
}

func (r *NetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *NetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LAN network (VLAN) on the Omada Controller.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the network.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": siteIDResourceSchema(),
			"name": schema.StringAttribute{
				Description: "The name of the network.",
				Required:    true,
			},
			"purpose": schema.StringAttribute{
				Description: "The purpose of the network ('interface' for gateway networks, 'vlan' for VLAN-only).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("vlan"),
			},
			"vlan_id": schema.Int64Attribute{
				Description: "The VLAN ID for the network (1-4094).",
				Required:    true,
			},
			"gateway_subnet": schema.StringAttribute{
				Description: "The gateway IP and subnet in CIDR notation (e.g., '192.168.0.1/24'). Only applicable for 'interface' purpose networks.",
				Optional:    true,
				Computed:    true,
			},
			"dhcp_enabled": schema.BoolAttribute{
				Description: "Whether DHCP is enabled on this network. Only applicable for 'interface' purpose networks.",
				Optional:    true,
				Computed:    true,
			},
			"dhcp_start": schema.StringAttribute{
				Description: "The start of the DHCP range. Only applicable when DHCP is enabled.",
				Optional:    true,
				Computed:    true,
			},
			"dhcp_end": schema.StringAttribute{
				Description: "The end of the DHCP range. Only applicable when DHCP is enabled.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *NetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := plan.SiteID.ValueString()

	network := &client.Network{
		Name:          plan.Name.ValueString(),
		Vlan:          int(plan.VlanID.ValueInt64()),
		GatewaySubnet: plan.GatewaySubnet.ValueString(),
	}
	if !plan.Purpose.IsNull() && !plan.Purpose.IsUnknown() {
		network.Purpose = plan.Purpose.ValueString()
	}
	// Build DHCPSettings if any DHCP attributes are set
	if !plan.DHCPEnabled.IsNull() && !plan.DHCPEnabled.IsUnknown() {
		network.DHCPSettings = &client.DHCPSettings{
			Enable:      plan.DHCPEnabled.ValueBool(),
			IPAddrStart: plan.DHCPStart.ValueString(),
			IPAddrEnd:   plan.DHCPEnd.ValueString(),
		}
	}

	created, err := r.client.CreateNetwork(ctx, siteID, network)
	if err != nil {
		resp.Diagnostics.AddError("Error creating network", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.Purpose = types.StringValue(created.Purpose)
	plan.GatewaySubnet = types.StringValue(created.GatewaySubnet)
	if created.DHCPSettings != nil {
		plan.DHCPEnabled = types.BoolValue(created.DHCPSettings.Enable)
		plan.DHCPStart = types.StringValue(created.DHCPSettings.IPAddrStart)
		plan.DHCPEnd = types.StringValue(created.DHCPSettings.IPAddrEnd)
	} else {
		plan.DHCPEnabled = types.BoolNull()
		plan.DHCPStart = types.StringNull()
		plan.DHCPEnd = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()

	network, err := r.client.GetNetwork(ctx, siteID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading network", err.Error())
		return
	}

	state.Name = types.StringValue(network.Name)
	state.Purpose = types.StringValue(network.Purpose)
	state.VlanID = types.Int64Value(int64(network.Vlan))
	state.GatewaySubnet = types.StringValue(network.GatewaySubnet)
	if network.DHCPSettings != nil {
		state.DHCPEnabled = types.BoolValue(network.DHCPSettings.Enable)
		state.DHCPStart = types.StringValue(network.DHCPSettings.IPAddrStart)
		state.DHCPEnd = types.StringValue(network.DHCPSettings.IPAddrEnd)
	} else {
		state.DHCPEnabled = types.BoolNull()
		state.DHCPStart = types.StringNull()
		state.DHCPEnd = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()

	network := &client.Network{
		ID:            state.ID.ValueString(),
		Name:          plan.Name.ValueString(),
		Vlan:          int(plan.VlanID.ValueInt64()),
		GatewaySubnet: plan.GatewaySubnet.ValueString(),
	}
	if !plan.Purpose.IsNull() {
		network.Purpose = plan.Purpose.ValueString()
	}
	// Build DHCPSettings if any DHCP attributes are set
	if !plan.DHCPEnabled.IsNull() && !plan.DHCPEnabled.IsUnknown() {
		network.DHCPSettings = &client.DHCPSettings{
			Enable:      plan.DHCPEnabled.ValueBool(),
			IPAddrStart: plan.DHCPStart.ValueString(),
			IPAddrEnd:   plan.DHCPEnd.ValueString(),
		}
	}

	updated, err := r.client.UpdateNetwork(ctx, siteID, state.ID.ValueString(), network)
	if err != nil {
		resp.Diagnostics.AddError("Error updating network", err.Error())
		return
	}

	plan.ID = state.ID
	plan.SiteID = state.SiteID
	plan.Purpose = types.StringValue(updated.Purpose)
	plan.GatewaySubnet = types.StringValue(updated.GatewaySubnet)
	if updated.DHCPSettings != nil {
		plan.DHCPEnabled = types.BoolValue(updated.DHCPSettings.Enable)
		plan.DHCPStart = types.StringValue(updated.DHCPSettings.IPAddrStart)
		plan.DHCPEnd = types.StringValue(updated.DHCPSettings.IPAddrEnd)
	} else {
		plan.DHCPEnabled = types.BoolNull()
		plan.DHCPStart = types.StringNull()
		plan.DHCPEnd = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteNetwork(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting network", err.Error())
		return
	}
}

func (r *NetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	siteID, networkID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'siteID/networkID'.",
		)
		return
	}

	network, err := r.client.GetNetwork(ctx, siteID, networkID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing network", err.Error())
		return
	}

	state := NetworkResourceModel{
		ID:            types.StringValue(network.ID),
		SiteID:        types.StringValue(siteID),
		Name:          types.StringValue(network.Name),
		Purpose:       types.StringValue(network.Purpose),
		VlanID:        types.Int64Value(int64(network.Vlan)),
		GatewaySubnet: types.StringValue(network.GatewaySubnet),
	}
	if network.DHCPSettings != nil {
		state.DHCPEnabled = types.BoolValue(network.DHCPSettings.Enable)
		state.DHCPStart = types.StringValue(network.DHCPSettings.IPAddrStart)
		state.DHCPEnd = types.StringValue(network.DHCPSettings.IPAddrEnd)
	} else {
		state.DHCPEnabled = types.BoolNull()
		state.DHCPStart = types.StringNull()
		state.DHCPEnd = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
