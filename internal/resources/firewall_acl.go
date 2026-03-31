package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-tplink-omada/internal/client"
)

var _ resource.Resource = &ACLRuleResource{}
var _ resource.ResourceWithImportState = &ACLRuleResource{}

// ACLRuleResource manages a firewall ACL rule on the Omada Controller.
type ACLRuleResource struct {
	client *client.Client
}

// ACLRuleResourceModel maps the resource schema to Go types.
type ACLRuleResourceModel struct {
	ID              types.String `tfsdk:"id"`
	SiteID          types.String `tfsdk:"site_id"`
	Name            types.String `tfsdk:"name"`
	Type            types.Int64  `tfsdk:"type"`
	Status          types.Bool   `tfsdk:"status"`
	Policy          types.Int64  `tfsdk:"policy"`
	Protocols       types.List   `tfsdk:"protocols"`
	SourceType      types.Int64  `tfsdk:"source_type"`
	SourceIDs       types.List   `tfsdk:"source_ids"`
	DestinationType types.Int64  `tfsdk:"destination_type"`
	DestinationIDs  types.List   `tfsdk:"destination_ids"`
	LanToWan        types.Bool   `tfsdk:"lan_to_wan"`
	LanToLan        types.Bool   `tfsdk:"lan_to_lan"`
	BiDirectional   types.Bool   `tfsdk:"bi_directional"`
	Index           types.Int64  `tfsdk:"index"`
}

func NewACLRuleResource() resource.Resource {
	return &ACLRuleResource{}
}

func (r *ACLRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_acl"
}

func (r *ACLRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a firewall ACL rule on the Omada Controller. " +
			"ACL rules control traffic between networks, IP groups, and other entities. " +
			"Gateway ACL rules (type=0) require a gateway device adopted into the site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the ACL rule.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": siteIDResourceSchema(),
			"name": schema.StringAttribute{
				Description: "The name of the ACL rule.",
				Required:    true,
			},
			"type": schema.Int64Attribute{
				Description: "The ACL type: 0=gateway, 1=switch, 2=eap.",
				Required:    true,
			},
			"status": schema.BoolAttribute{
				Description: "Whether the rule is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"policy": schema.Int64Attribute{
				Description: "The policy action: 0=deny, 1=permit.",
				Required:    true,
			},
			"protocols": schema.ListAttribute{
				Description: "List of IP protocol numbers: 6=TCP, 17=UDP, 1=ICMP.",
				Required:    true,
				ElementType: types.Int64Type,
			},
			"source_type": schema.Int64Attribute{
				Description: "Source type: 0=network, 2=ip_group.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"source_ids": schema.ListAttribute{
				Description: "List of source entity IDs (network IDs or IP group IDs depending on source_type).",
				Required:    true,
				ElementType: types.StringType,
			},
			"destination_type": schema.Int64Attribute{
				Description: "Destination type: 0=network, 2=ip_group.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"destination_ids": schema.ListAttribute{
				Description: "List of destination entity IDs (network IDs or IP group IDs depending on destination_type).",
				Required:    true,
				ElementType: types.StringType,
			},
			"lan_to_wan": schema.BoolAttribute{
				Description: "Whether this rule applies to LAN-to-WAN traffic.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"lan_to_lan": schema.BoolAttribute{
				Description: "Whether this rule applies to LAN-to-LAN traffic.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"bi_directional": schema.BoolAttribute{
				Description: "Whether this rule applies in both directions.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"index": schema.Int64Attribute{
				Description: "Rule ordering index (first-match-wins). Computed by the controller.",
				Computed:    true,
			},
		},
	}
}

func (r *ACLRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ACLRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ACLRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := plan.SiteID.ValueString()

	var protocols []int
	resp.Diagnostics.Append(plan.Protocols.ElementsAs(ctx, &protocols, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sourceIDs []string
	resp.Diagnostics.Append(plan.SourceIDs.ElementsAs(ctx, &sourceIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var destIDs []string
	resp.Diagnostics.Append(plan.DestinationIDs.ElementsAs(ctx, &destIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule := &client.ACLRule{
		Name:            plan.Name.ValueString(),
		Type:            int(plan.Type.ValueInt64()),
		Status:          plan.Status.ValueBool(),
		Policy:          int(plan.Policy.ValueInt64()),
		Protocols:       protocols,
		SourceType:      int(plan.SourceType.ValueInt64()),
		SourceIDs:       sourceIDs,
		DestinationType: int(plan.DestinationType.ValueInt64()),
		DestinationIDs:  destIDs,
		BiDirectional:   plan.BiDirectional.ValueBool(),
		Direction: client.ACLDirection{
			LanToWan: plan.LanToWan.ValueBool(),
			LanToLan: plan.LanToLan.ValueBool(),
		},
	}

	created, err := r.client.CreateACLRule(ctx, siteID, rule)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ACL rule", err.Error())
		return
	}

	r.setStateFromAPI(ctx, &plan, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ACLRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ACLRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()
	aclType := int(state.Type.ValueInt64())

	rule, err := r.client.GetACLRule(ctx, siteID, state.ID.ValueString(), aclType)
	if err != nil {
		resp.Diagnostics.AddError("Error reading ACL rule", err.Error())
		return
	}

	r.setStateFromAPI(ctx, &state, rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ACLRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ACLRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ACLRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()

	var protocols []int
	resp.Diagnostics.Append(plan.Protocols.ElementsAs(ctx, &protocols, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sourceIDs []string
	resp.Diagnostics.Append(plan.SourceIDs.ElementsAs(ctx, &sourceIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var destIDs []string
	resp.Diagnostics.Append(plan.DestinationIDs.ElementsAs(ctx, &destIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule := &client.ACLRule{
		Name:            plan.Name.ValueString(),
		Type:            int(plan.Type.ValueInt64()),
		Status:          plan.Status.ValueBool(),
		Policy:          int(plan.Policy.ValueInt64()),
		Protocols:       protocols,
		SourceType:      int(plan.SourceType.ValueInt64()),
		SourceIDs:       sourceIDs,
		DestinationType: int(plan.DestinationType.ValueInt64()),
		DestinationIDs:  destIDs,
		BiDirectional:   plan.BiDirectional.ValueBool(),
		Direction: client.ACLDirection{
			LanToWan: plan.LanToWan.ValueBool(),
			LanToLan: plan.LanToLan.ValueBool(),
		},
	}

	updated, err := r.client.UpdateACLRule(ctx, siteID, state.ID.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ACL rule", err.Error())
		return
	}

	plan.ID = state.ID
	plan.SiteID = state.SiteID
	r.setStateFromAPI(ctx, &plan, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ACLRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ACLRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteACLRule(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting ACL rule", err.Error())
		return
	}
}

func (r *ACLRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: siteID/aclType/ruleID
	siteID, rest, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'siteID/aclType/ruleID' (e.g., 'siteId/0/ruleId').",
		)
		return
	}

	// rest should be "aclType/ruleID"
	aclTypeStr, ruleID, ok := parseImportID(rest)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'siteID/aclType/ruleID' (e.g., 'siteId/0/ruleId').",
		)
		return
	}

	aclType, err := strconv.Atoi(aclTypeStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ACL type in import ID",
			fmt.Sprintf("ACL type must be an integer (0=gateway, 1=switch, 2=eap), got: %s", aclTypeStr),
		)
		return
	}

	rule, err := r.client.GetACLRule(ctx, siteID, ruleID, aclType)
	if err != nil {
		resp.Diagnostics.AddError("Error importing ACL rule", err.Error())
		return
	}

	state := ACLRuleResourceModel{
		SiteID: types.StringValue(siteID),
	}
	r.setStateFromAPI(ctx, &state, rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// setStateFromAPI populates the resource model from an API response.
func (r *ACLRuleResource) setStateFromAPI(ctx context.Context, model *ACLRuleResourceModel, rule *client.ACLRule) {
	model.ID = types.StringValue(rule.ID)
	model.Name = types.StringValue(rule.Name)
	model.Type = types.Int64Value(int64(rule.Type))
	model.Status = types.BoolValue(rule.Status)
	model.Policy = types.Int64Value(int64(rule.Policy))
	model.SourceType = types.Int64Value(int64(rule.SourceType))
	model.DestinationType = types.Int64Value(int64(rule.DestinationType))
	model.LanToWan = types.BoolValue(rule.Direction.LanToWan)
	model.LanToLan = types.BoolValue(rule.Direction.LanToLan)
	model.BiDirectional = types.BoolValue(rule.BiDirectional)
	model.Index = types.Int64Value(int64(rule.Index))

	protocols, _ := types.ListValueFrom(ctx, types.Int64Type, rule.Protocols)
	model.Protocols = protocols

	sourceIDs, _ := types.ListValueFrom(ctx, types.StringType, rule.SourceIDs)
	model.SourceIDs = sourceIDs

	destIDs, _ := types.ListValueFrom(ctx, types.StringType, rule.DestinationIDs)
	model.DestinationIDs = destIDs
}
