package resources

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-tplink-omada/internal/client"
)

var _ resource.Resource = &SAMLIdPResource{}
var _ resource.ResourceWithImportState = &SAMLIdPResource{}

type SAMLIdPResource struct {
	client *client.Client
}

type SAMLIdPResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	EntityID    types.String `tfsdk:"entity_id"`
	LoginURL    types.String `tfsdk:"login_url"`
	X509Cert    types.String `tfsdk:"x509_certificate"`
	EntityURL   types.String `tfsdk:"entity_url"`
	SignOnURL   types.String `tfsdk:"sign_on_url"`
}

func NewSAMLIdPResource() resource.Resource {
	return &SAMLIdPResource{}
}

func (r *SAMLIdPResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saml_idp"
}

func (r *SAMLIdPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a SAML identity provider connection on the Omada Controller.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the SAML IdP connection.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The display name of the SAML IdP connection.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the SAML IdP connection.",
				Optional:    true,
			},
			"entity_id": schema.StringAttribute{
				Description: "The SAML Entity ID (Issuer) of the identity provider.",
				Required:    true,
			},
			"login_url": schema.StringAttribute{
				Description: "The SAML Single Sign-On URL of the identity provider.",
				Required:    true,
			},
			"x509_certificate": schema.StringAttribute{
				Description: "The X.509 signing certificate in PEM format. Automatically base64-encoded before sending to the API.",
				Required:    true,
				Sensitive:   true,
			},
			"entity_url": schema.StringAttribute{
				Description: "The SP Entity ID (computed by the controller).",
				Computed:    true,
			},
			"sign_on_url": schema.StringAttribute{
				Description: "The SP Assertion Consumer Service URL (computed by the controller).",
				Computed:    true,
			},
		},
	}
}

func (r *SAMLIdPResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SAMLIdPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SAMLIdPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.SAMLIdPCreateRequest{
		IdpName:     plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		EntityID:    plan.EntityID.ValueString(),
		LoginURL:    plan.LoginURL.ValueString(),
		X509Cert:    base64.StdEncoding.EncodeToString([]byte(plan.X509Cert.ValueString())),
	}

	idp, err := r.client.CreateSAMLIdP(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SAML IdP", err.Error())
		return
	}

	mapSAMLIdPToState(idp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SAMLIdPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SAMLIdPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idp, err := r.client.GetSAMLIdP(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading SAML IdP", err.Error())
		return
	}

	mapSAMLIdPToState(idp, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SAMLIdPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SAMLIdPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SAMLIdPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.SAMLIdPCreateRequest{
		IdpName:     plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		EntityID:    plan.EntityID.ValueString(),
		LoginURL:    plan.LoginURL.ValueString(),
		X509Cert:    base64.StdEncoding.EncodeToString([]byte(plan.X509Cert.ValueString())),
	}

	idp, err := r.client.UpdateSAMLIdP(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating SAML IdP", err.Error())
		return
	}

	mapSAMLIdPToState(idp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SAMLIdPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SAMLIdPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSAMLIdP(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting SAML IdP", err.Error())
		return
	}
}

func (r *SAMLIdPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idp, err := r.client.GetSAMLIdP(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing SAML IdP", err.Error())
		return
	}

	state := SAMLIdPResourceModel{}
	mapSAMLIdPToState(idp, &state)
	// x509_certificate is returned base64-encoded from API; we can't recover the original PEM,
	// so we store whatever the API returns. User must set the correct PEM in their config.
	state.X509Cert = types.StringValue(idp.X509Cert)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func mapSAMLIdPToState(idp *client.SAMLIdP, state *SAMLIdPResourceModel) {
	state.ID = types.StringValue(idp.IdpID)
	state.Name = types.StringValue(idp.IdpName)
	if idp.Description != "" {
		state.Description = types.StringValue(idp.Description)
	}
	state.EntityID = types.StringValue(idp.EntityID)
	state.LoginURL = types.StringValue(idp.LoginURL)
	state.EntityURL = types.StringValue(idp.EntityURL)
	state.SignOnURL = types.StringValue(idp.SignOnURL)
	// x509_certificate: preserve plan value (PEM) — the API returns it base64-encoded,
	// so we don't overwrite with the encoded version to avoid perpetual diff.
}
