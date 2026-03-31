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

var _ resource.Resource = &MDNSReflectorResource{}
var _ resource.ResourceWithImportState = &MDNSReflectorResource{}

// MDNSReflectorResource manages an mDNS reflector rule on the Omada Controller.
type MDNSReflectorResource struct {
	client *client.Client
}

// MDNSReflectorResourceModel maps the resource schema to Go types.
type MDNSReflectorResourceModel struct {
	ID              types.String `tfsdk:"id"`
	SiteID          types.String `tfsdk:"site_id"`
	Name            types.String `tfsdk:"name"`
	Type            types.Int64  `tfsdk:"type"`
	Status          types.Bool   `tfsdk:"status"`
	ProfileIDs      types.List   `tfsdk:"profile_ids"`
	ServiceNetworks types.List   `tfsdk:"service_networks"`
	ClientNetworks  types.List   `tfsdk:"client_networks"`
}

func NewMDNSReflectorResource() resource.Resource {
	return &MDNSReflectorResource{}
}

func (r *MDNSReflectorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mdns_reflector"
}

func (r *MDNSReflectorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an mDNS reflector rule on the Omada Controller. " +
			"mDNS reflector rules enable service discovery (AirPlay, Chromecast, etc.) across VLANs. " +
			"Gateway rules (type=1) reflect mDNS at the router level; AP rules (type=0) reflect at the access point level.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the mDNS reflector rule.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": siteIDResourceSchema(),
			"name": schema.StringAttribute{
				Description: "The name of the mDNS reflector rule.",
				Required:    true,
			},
			"type": schema.Int64Attribute{
				Description: "The rule type: 0=AP, 1=OSG (gateway). Gateway rules require a gateway device adopted into the site.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
			},
			"status": schema.BoolAttribute{
				Description: "Whether the rule is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"profile_ids": schema.ListAttribute{
				Description: "List of built-in service profile IDs. Known values: \"buildIn-1\" = AirPlay.",
				Required:    true,
				ElementType: types.StringType,
			},
			"service_networks": schema.ListAttribute{
				Description: "List of network IDs where services are provided (e.g., TVs, speakers, Chromecasts).",
				Required:    true,
				ElementType: types.StringType,
			},
			"client_networks": schema.ListAttribute{
				Description: "List of network IDs where clients discover services (e.g., phones, laptops).",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *MDNSReflectorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MDNSReflectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MDNSReflectorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := plan.SiteID.ValueString()

	var profileIDs []string
	resp.Diagnostics.Append(plan.ProfileIDs.ElementsAs(ctx, &profileIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var serviceNetworks []string
	resp.Diagnostics.Append(plan.ServiceNetworks.ElementsAs(ctx, &serviceNetworks, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var clientNetworks []string
	resp.Diagnostics.Append(plan.ClientNetworks.ElementsAs(ctx, &clientNetworks, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkSetting := &client.MDNSNetworkSetting{
		ProfileIDs:      profileIDs,
		ServiceNetworks: serviceNetworks,
		ClientNetworks:  clientNetworks,
	}

	rule := &client.MDNSRule{
		Name:   plan.Name.ValueString(),
		Status: plan.Status.ValueBool(),
		Type:   int(plan.Type.ValueInt64()),
	}

	// Set the appropriate nested key based on rule type
	if rule.Type == 1 {
		rule.OSG = networkSetting
	} else {
		rule.AP = networkSetting
	}

	created, err := r.client.CreateMDNSRule(ctx, siteID, rule)
	if err != nil {
		resp.Diagnostics.AddError("Error creating mDNS reflector rule", err.Error())
		return
	}

	r.setStateFromAPI(ctx, &plan, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MDNSReflectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MDNSReflectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()

	rule, err := r.client.GetMDNSRule(ctx, siteID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading mDNS reflector rule", err.Error())
		return
	}

	r.setStateFromAPI(ctx, &state, rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MDNSReflectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MDNSReflectorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state MDNSReflectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()

	var profileIDs []string
	resp.Diagnostics.Append(plan.ProfileIDs.ElementsAs(ctx, &profileIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var serviceNetworks []string
	resp.Diagnostics.Append(plan.ServiceNetworks.ElementsAs(ctx, &serviceNetworks, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var clientNetworks []string
	resp.Diagnostics.Append(plan.ClientNetworks.ElementsAs(ctx, &clientNetworks, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkSetting := &client.MDNSNetworkSetting{
		ProfileIDs:      profileIDs,
		ServiceNetworks: serviceNetworks,
		ClientNetworks:  clientNetworks,
	}

	rule := &client.MDNSRule{
		Name:   plan.Name.ValueString(),
		Status: plan.Status.ValueBool(),
		Type:   int(plan.Type.ValueInt64()),
	}

	if rule.Type == 1 {
		rule.OSG = networkSetting
	} else {
		rule.AP = networkSetting
	}

	updated, err := r.client.UpdateMDNSRule(ctx, siteID, state.ID.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError("Error updating mDNS reflector rule", err.Error())
		return
	}

	plan.ID = state.ID
	plan.SiteID = state.SiteID
	r.setStateFromAPI(ctx, &plan, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MDNSReflectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MDNSReflectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMDNSRule(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting mDNS reflector rule", err.Error())
		return
	}
}

func (r *MDNSReflectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: siteID/ruleID
	siteID, ruleID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'siteID/ruleID'.",
		)
		return
	}

	rule, err := r.client.GetMDNSRule(ctx, siteID, ruleID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing mDNS reflector rule", err.Error())
		return
	}

	state := MDNSReflectorResourceModel{
		SiteID: types.StringValue(siteID),
	}
	r.setStateFromAPI(ctx, &state, rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// setStateFromAPI populates the resource model from an API response.
func (r *MDNSReflectorResource) setStateFromAPI(ctx context.Context, model *MDNSReflectorResourceModel, rule *client.MDNSRule) {
	model.ID = types.StringValue(rule.ID)
	model.Name = types.StringValue(rule.Name)
	model.Type = types.Int64Value(int64(rule.Type))
	model.Status = types.BoolValue(rule.Status)

	// Extract network settings from the appropriate nested key
	var setting *client.MDNSNetworkSetting
	if rule.OSG != nil {
		setting = rule.OSG
	} else if rule.AP != nil {
		setting = rule.AP
	}

	if setting != nil {
		profileIDs, _ := types.ListValueFrom(ctx, types.StringType, setting.ProfileIDs)
		model.ProfileIDs = profileIDs

		serviceNetworks, _ := types.ListValueFrom(ctx, types.StringType, setting.ServiceNetworks)
		model.ServiceNetworks = serviceNetworks

		clientNetworks, _ := types.ListValueFrom(ctx, types.StringType, setting.ClientNetworks)
		model.ClientNetworks = clientNetworks
	} else {
		// No network setting — set empty lists
		model.ProfileIDs, _ = types.ListValueFrom(ctx, types.StringType, []string{})
		model.ServiceNetworks, _ = types.ListValueFrom(ctx, types.StringType, []string{})
		model.ClientNetworks, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}
}
