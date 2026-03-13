package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-tplink-omada/internal/client"
)

var _ resource.Resource = &WlanGroupResource{}
var _ resource.ResourceWithImportState = &WlanGroupResource{}

// WlanGroupResource manages an Omada WLAN group.
type WlanGroupResource struct {
	client *client.Client
}

// WlanGroupResourceModel maps the resource schema to Go types.
type WlanGroupResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Primary types.Bool   `tfsdk:"primary"`
}

func NewWlanGroupResource() resource.Resource {
	return &WlanGroupResource{}
}

func (r *WlanGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wlan_group"
}

func (r *WlanGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WLAN group on the Omada Controller. WLAN groups organize wireless networks (SSIDs) together.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the WLAN group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the WLAN group.",
				Required:    true,
			},
			"primary": schema.BoolAttribute{
				Description: "Whether this is the primary (default) WLAN group. Read-only, computed from the API.",
				Computed:    true,
			},
		},
	}
}

func (r *WlanGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WlanGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WlanGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()

	// The Omada controller auto-creates a "Default" WLAN group when a site is
	// created.  Check whether a group with the requested name already exists
	// and adopt it instead of trying to create a duplicate (which would fail
	// with error -33200).
	groups, err := r.client.ListWlanGroups(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing WLAN groups", err.Error())
		return
	}

	var existing *client.WlanGroup
	for _, g := range groups {
		if g.Name == name {
			existing = &g
			break
		}
	}

	if existing != nil {
		// Adopt the existing group.
		plan.ID = types.StringValue(existing.ID)
		plan.Name = types.StringValue(existing.Name)
		plan.Primary = types.BoolValue(existing.Primary)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	// No existing group — create a new one.
	wlanID, err := r.client.CreateWlanGroup(ctx, name, false)
	if err != nil {
		resp.Diagnostics.AddError("Error creating WLAN group", err.Error())
		return
	}

	// Read back to get full state
	group, err := r.client.GetWlanGroup(ctx, wlanID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created WLAN group", err.Error())
		return
	}

	plan.ID = types.StringValue(group.ID)
	plan.Name = types.StringValue(group.Name)
	plan.Primary = types.BoolValue(group.Primary)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WlanGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WlanGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.client.GetWlanGroup(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading WLAN group", err.Error())
		return
	}

	state.Name = types.StringValue(group.Name)
	state.Primary = types.BoolValue(group.Primary)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WlanGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WlanGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WlanGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateWlanGroup(ctx, state.ID.ValueString(), plan.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating WLAN group", err.Error())
		return
	}

	// Read back to confirm
	group, err := r.client.GetWlanGroup(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated WLAN group", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Name = types.StringValue(group.Name)
	plan.Primary = types.BoolValue(group.Primary)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WlanGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WlanGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteWlanGroup(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting WLAN group", err.Error())
		return
	}
}

func (r *WlanGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	group, err := r.client.GetWlanGroup(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing WLAN group", err.Error())
		return
	}

	state := WlanGroupResourceModel{
		ID:      types.StringValue(group.ID),
		Name:    types.StringValue(group.Name),
		Primary: types.BoolValue(group.Primary),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
