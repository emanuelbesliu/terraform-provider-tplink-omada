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

var _ resource.Resource = &PortProfileResource{}
var _ resource.ResourceWithImportState = &PortProfileResource{}

type PortProfileResource struct {
	client *client.Client
}

type PortProfileResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	SiteID               types.String `tfsdk:"site_id"`
	Name                 types.String `tfsdk:"name"`
	NativeNetworkID      types.String `tfsdk:"native_network_id"`
	TagNetworkIDs        types.List   `tfsdk:"tag_network_ids"`
	POE                  types.Int64  `tfsdk:"poe"`
	Dot1x                types.Int64  `tfsdk:"dot1x"`
	PortIsolationEnable  types.Bool   `tfsdk:"port_isolation_enable"`
	LLDPMedEnable        types.Bool   `tfsdk:"lldp_med_enable"`
	TopoNotifyEnable     types.Bool   `tfsdk:"topo_notify_enable"`
	SpanningTreeEnable   types.Bool   `tfsdk:"spanning_tree_enable"`
	LoopbackDetectEnable types.Bool   `tfsdk:"loopback_detect_enable"`
}

func NewPortProfileResource() resource.Resource {
	return &PortProfileResource{}
}

func (r *PortProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_profile"
}

func (r *PortProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a switch port profile on the Omada Controller.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the port profile.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": siteIDResourceSchema(),
			"name": schema.StringAttribute{
				Description: "The name of the port profile.",
				Required:    true,
			},
			"native_network_id": schema.StringAttribute{
				Description: "The native (untagged) network ID. Required for trunk profiles.",
				Required:    true,
			},
			"tag_network_ids": schema.ListAttribute{
				Description: "List of tagged network IDs for trunk profiles.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"poe": schema.Int64Attribute{
				Description: "PoE setting: 0=disabled, 1=enabled, 2=use profile default.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"dot1x": schema.Int64Attribute{
				Description: "802.1X setting: 0=port-based, 1=mac-based, 2=disabled.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"port_isolation_enable": schema.BoolAttribute{
				Description: "Enable port isolation.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"lldp_med_enable": schema.BoolAttribute{
				Description: "Enable LLDP-MED.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"topo_notify_enable": schema.BoolAttribute{
				Description: "Enable topology change notification.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"spanning_tree_enable": schema.BoolAttribute{
				Description: "Enable Spanning Tree Protocol.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"loopback_detect_enable": schema.BoolAttribute{
				Description: "Enable loopback detection.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *PortProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PortProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PortProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := plan.SiteID.ValueString()

	profile := &client.PortProfile{
		Name:                 plan.Name.ValueString(),
		NativeNetworkID:      plan.NativeNetworkID.ValueString(),
		POE:                  int(plan.POE.ValueInt64()),
		Dot1x:                int(plan.Dot1x.ValueInt64()),
		PortIsolationEnable:  plan.PortIsolationEnable.ValueBool(),
		LLDPMedEnable:        plan.LLDPMedEnable.ValueBool(),
		TopoNotifyEnable:     plan.TopoNotifyEnable.ValueBool(),
		SpanningTreeEnable:   plan.SpanningTreeEnable.ValueBool(),
		LoopbackDetectEnable: plan.LoopbackDetectEnable.ValueBool(),
		SpanningTreeSetting: &client.SpanningTreeSetting{
			Priority:    128,
			BpduForward: true,
		},
		DhcpL2RelaySettings: &client.DhcpL2RelaySettings{},
	}

	if !plan.TagNetworkIDs.IsNull() && !plan.TagNetworkIDs.IsUnknown() {
		var tagIDs []string
		resp.Diagnostics.Append(plan.TagNetworkIDs.ElementsAs(ctx, &tagIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		profile.TagNetworkIDs = tagIDs
	}

	created, err := r.client.CreatePortProfile(ctx, siteID, profile)
	if err != nil {
		resp.Diagnostics.AddError("Error creating port profile", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PortProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PortProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()

	profile, err := r.client.GetPortProfile(ctx, siteID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading port profile", err.Error())
		return
	}

	state.Name = types.StringValue(profile.Name)
	state.NativeNetworkID = types.StringValue(profile.NativeNetworkID)
	state.POE = types.Int64Value(int64(profile.POE))
	state.Dot1x = types.Int64Value(int64(profile.Dot1x))
	state.PortIsolationEnable = types.BoolValue(profile.PortIsolationEnable)
	state.LLDPMedEnable = types.BoolValue(profile.LLDPMedEnable)
	state.TopoNotifyEnable = types.BoolValue(profile.TopoNotifyEnable)
	state.SpanningTreeEnable = types.BoolValue(profile.SpanningTreeEnable)
	state.LoopbackDetectEnable = types.BoolValue(profile.LoopbackDetectEnable)

	tagIDs, diags := types.ListValueFrom(ctx, types.StringType, profile.TagNetworkIDs)
	resp.Diagnostics.Append(diags...)
	state.TagNetworkIDs = tagIDs

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PortProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PortProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state PortProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()

	profile := &client.PortProfile{
		Name:                 plan.Name.ValueString(),
		NativeNetworkID:      plan.NativeNetworkID.ValueString(),
		POE:                  int(plan.POE.ValueInt64()),
		Dot1x:                int(plan.Dot1x.ValueInt64()),
		PortIsolationEnable:  plan.PortIsolationEnable.ValueBool(),
		LLDPMedEnable:        plan.LLDPMedEnable.ValueBool(),
		TopoNotifyEnable:     plan.TopoNotifyEnable.ValueBool(),
		SpanningTreeEnable:   plan.SpanningTreeEnable.ValueBool(),
		LoopbackDetectEnable: plan.LoopbackDetectEnable.ValueBool(),
		SpanningTreeSetting: &client.SpanningTreeSetting{
			Priority:    128,
			BpduForward: true,
		},
		DhcpL2RelaySettings: &client.DhcpL2RelaySettings{},
	}

	if !plan.TagNetworkIDs.IsNull() && !plan.TagNetworkIDs.IsUnknown() {
		var tagIDs []string
		resp.Diagnostics.Append(plan.TagNetworkIDs.ElementsAs(ctx, &tagIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		profile.TagNetworkIDs = tagIDs
	}

	_, err := r.client.UpdatePortProfile(ctx, siteID, state.ID.ValueString(), profile)
	if err != nil {
		resp.Diagnostics.AddError("Error updating port profile", err.Error())
		return
	}

	plan.ID = state.ID
	plan.SiteID = state.SiteID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PortProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PortProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePortProfile(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting port profile", err.Error())
		return
	}
}

func (r *PortProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	siteID, profileID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'siteID/profileID'.",
		)
		return
	}

	profile, err := r.client.GetPortProfile(ctx, siteID, profileID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing port profile", err.Error())
		return
	}

	tagIDs, diags := types.ListValueFrom(ctx, types.StringType, profile.TagNetworkIDs)
	resp.Diagnostics.Append(diags...)

	state := PortProfileResourceModel{
		ID:                   types.StringValue(profile.ID),
		SiteID:               types.StringValue(siteID),
		Name:                 types.StringValue(profile.Name),
		NativeNetworkID:      types.StringValue(profile.NativeNetworkID),
		TagNetworkIDs:        tagIDs,
		POE:                  types.Int64Value(int64(profile.POE)),
		Dot1x:                types.Int64Value(int64(profile.Dot1x)),
		PortIsolationEnable:  types.BoolValue(profile.PortIsolationEnable),
		LLDPMedEnable:        types.BoolValue(profile.LLDPMedEnable),
		TopoNotifyEnable:     types.BoolValue(profile.TopoNotifyEnable),
		SpanningTreeEnable:   types.BoolValue(profile.SpanningTreeEnable),
		LoopbackDetectEnable: types.BoolValue(profile.LoopbackDetectEnable),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
