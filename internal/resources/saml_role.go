package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-tplink-omada/internal/client"
)

var _ resource.Resource = &SAMLRoleResource{}
var _ resource.ResourceWithImportState = &SAMLRoleResource{}

type SAMLRoleResource struct {
	client *client.Client
}

type SAMLRoleResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	RoleID          types.String `tfsdk:"role_id"`
	RoleName        types.String `tfsdk:"role_name"`
	SiteIDs         types.List   `tfsdk:"site_ids"`
	TemporaryEnable types.Bool   `tfsdk:"temporary_enable"`
	StartTime       types.Int64  `tfsdk:"start_time"`
	EndTime         types.Int64  `tfsdk:"end_time"`
}

func NewSAMLRoleResource() resource.Resource {
	return &SAMLRoleResource{}
}

func (r *SAMLRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saml_role"
}

func (r *SAMLRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a SAML role (external user group) on the Omada Controller. Maps a SAML usergroup_name attribute to a controller role with site access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the SAML role.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the external user group. Must match the SAML usergroup_name attribute value exactly (case-sensitive).",
				Required:    true,
			},
			"role_id": schema.StringAttribute{
				Description: "The controller role to assign. Valid values: super_admin_id, admin_id, viewer_id.",
				Required:    true,
			},
			"role_name": schema.StringAttribute{
				Description: "The human-readable name of the assigned role (computed).",
				Computed:    true,
			},
			"site_ids": schema.ListAttribute{
				Description: "List of site IDs this role has access to.",
				Required:    true,
				ElementType: types.StringType,
			},
			"temporary_enable": schema.BoolAttribute{
				Description: "Whether this role has a temporary validity period. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"start_time": schema.Int64Attribute{
				Description: "Start time of the temporary validity period (Unix milliseconds). Defaults to 0.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"end_time": schema.Int64Attribute{
				Description: "End time of the temporary validity period (Unix milliseconds). Defaults to 0.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
		},
	}
}

func (r *SAMLRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SAMLRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SAMLRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var siteIDs []string
	resp.Diagnostics.Append(plan.SiteIDs.ElementsAs(ctx, &siteIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.SAMLRoleCreateRequest{
		UserGroupName:   plan.Name.ValueString(),
		RoleID:          plan.RoleID.ValueString(),
		TemporaryEnable: plan.TemporaryEnable.ValueBool(),
		StartTime:       plan.StartTime.ValueInt64(),
		EndTime:         plan.EndTime.ValueInt64(),
		SitePrivileges: []client.SAMLRoleSitePrivilegeCreate{{
			SiteType:    2,
			Sites:       siteIDs,
			ServiceType: 1,
		}},
	}

	role, err := r.client.CreateSAMLRole(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SAML role", err.Error())
		return
	}

	resp.Diagnostics.Append(mapSAMLRoleToState(ctx, role, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SAMLRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SAMLRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetSAMLRole(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading SAML role", err.Error())
		return
	}

	resp.Diagnostics.Append(mapSAMLRoleToState(ctx, role, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SAMLRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SAMLRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SAMLRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var siteIDs []string
	resp.Diagnostics.Append(plan.SiteIDs.ElementsAs(ctx, &siteIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.SAMLRoleCreateRequest{
		UserGroupName:   plan.Name.ValueString(),
		RoleID:          plan.RoleID.ValueString(),
		TemporaryEnable: plan.TemporaryEnable.ValueBool(),
		StartTime:       plan.StartTime.ValueInt64(),
		EndTime:         plan.EndTime.ValueInt64(),
		SitePrivileges: []client.SAMLRoleSitePrivilegeCreate{{
			SiteType:    2,
			Sites:       siteIDs,
			ServiceType: 1,
		}},
	}

	role, err := r.client.UpdateSAMLRole(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating SAML role", err.Error())
		return
	}

	resp.Diagnostics.Append(mapSAMLRoleToState(ctx, role, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SAMLRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SAMLRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSAMLRole(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting SAML role", err.Error())
		return
	}
}

func (r *SAMLRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	role, err := r.client.GetSAMLRole(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing SAML role", err.Error())
		return
	}

	state := SAMLRoleResourceModel{}
	resp.Diagnostics.Append(mapSAMLRoleToState(ctx, role, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func mapSAMLRoleToState(ctx context.Context, role *client.SAMLRole, state *SAMLRoleResourceModel) diag.Diagnostics {
	state.ID = types.StringValue(role.ID)
	state.Name = types.StringValue(role.UserGroupName)
	state.RoleID = types.StringValue(role.RoleID)
	state.RoleName = types.StringValue(role.RoleName)
	state.TemporaryEnable = types.BoolValue(role.TemporaryEnable)
	state.StartTime = types.Int64Value(role.StartTime)
	state.EndTime = types.Int64Value(role.EndTime)

	var siteIDs []string
	for _, sp := range role.SitePrivileges {
		for _, site := range sp.Sites {
			siteIDs = append(siteIDs, site.ID)
		}
	}
	siteList, diags := types.ListValueFrom(ctx, types.StringType, siteIDs)
	if diags.HasError() {
		return diags
	}
	state.SiteIDs = siteList
	return nil
}
