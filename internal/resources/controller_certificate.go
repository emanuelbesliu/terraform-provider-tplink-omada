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

var _ resource.Resource = &ControllerCertificateResource{}
var _ resource.ResourceWithImportState = &ControllerCertificateResource{}

type ControllerCertificateResource struct {
	client *client.Client
}

type ControllerCertificateResourceModel struct {
	ID             types.String `tfsdk:"id"`
	CertificateKey types.String `tfsdk:"certificate_pem"`
	PrivateKeyKey  types.String `tfsdk:"private_key_pem"`
	CertID         types.String `tfsdk:"cert_id"`
	KeyID          types.String `tfsdk:"key_id"`
}

func NewControllerCertificateResource() resource.Resource {
	return &ControllerCertificateResource{}
}

func (r *ControllerCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_controller_certificate"
}

func (r *ControllerCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the TLS certificate for the Omada SDN Controller.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the controller certificate resource. Always set to 'controller-cert'.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"certificate_pem": schema.StringAttribute{
				Description: "The PEM-encoded certificate (public key) for the Omada Controller. Can be a single leaf certificate or a full chain (leaf + intermediates + root).",
				Required:    true,
				Sensitive:   true,
			},
			"private_key_pem": schema.StringAttribute{
				Description: "The PEM-encoded private key for the Omada Controller certificate.",
				Required:    true,
				Sensitive:   true,
			},
			"cert_id": schema.StringAttribute{
				Description: "The certificate ID returned by the controller after upload (computed).",
				Computed:    true,
			},
			"key_id": schema.StringAttribute{
				Description: "The private key ID returned by the controller after upload (computed).",
				Computed:    true,
			},
		},
	}
}

func (r *ControllerCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ControllerCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ControllerCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract PEM strings
	certPEM := []byte(plan.CertificateKey.ValueString())
	keyPEM := []byte(plan.PrivateKeyKey.ValueString())

	// Upload certificate
	certID, err := r.client.UploadCertificate(ctx, certPEM, "controller.crt")
	if err != nil {
		resp.Diagnostics.AddError("Error uploading certificate", err.Error())
		return
	}

	// Upload key
	keyID, err := r.client.UploadKey(ctx, keyPEM, "controller.key")
	if err != nil {
		resp.Diagnostics.AddError("Error uploading private key", err.Error())
		return
	}

	// Activate certificate
	if err := r.client.ActivateCertificate(ctx, certID, keyID); err != nil {
		resp.Diagnostics.AddError("Error activating certificate", err.Error())
		return
	}

	// Set the state
	plan.ID = types.StringValue("controller-cert")
	plan.CertID = types.StringValue(certID)
	plan.KeyID = types.StringValue(keyID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ControllerCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ControllerCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch current certificate setting from controller
	setting, err := r.client.GetControllerCertificateSetting(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading controller certificate setting", err.Error())
		return
	}

	// Update computed fields
	state.CertID = types.StringValue(setting.CertID)
	state.KeyID = types.StringValue(setting.KeyID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ControllerCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ControllerCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract PEM strings
	certPEM := []byte(plan.CertificateKey.ValueString())
	keyPEM := []byte(plan.PrivateKeyKey.ValueString())

	// Upload new certificate
	certID, err := r.client.UploadCertificate(ctx, certPEM, "controller.crt")
	if err != nil {
		resp.Diagnostics.AddError("Error uploading certificate", err.Error())
		return
	}

	// Upload new key
	keyID, err := r.client.UploadKey(ctx, keyPEM, "controller.key")
	if err != nil {
		resp.Diagnostics.AddError("Error uploading private key", err.Error())
		return
	}

	// Activate the new certificate
	if err := r.client.ActivateCertificate(ctx, certID, keyID); err != nil {
		resp.Diagnostics.AddError("Error activating certificate", err.Error())
		return
	}

	// Update state
	plan.ID = types.StringValue("controller-cert")
	plan.CertID = types.StringValue(certID)
	plan.KeyID = types.StringValue(keyID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ControllerCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The Omada API does not support certificate deletion.
	// When the resource is destroyed, we simply remove it from Terraform state.
	// The certificate remains on the controller and continues to be used.
	//
	// This is intentional to prevent accidental deletion of the controller's TLS certificate,
	// which would render the controller's web interface inaccessible.
	//
	// To replace the certificate, use Terraform apply with a new certificate_pem value.
	// This is a no-op delete.
}

func (r *ControllerCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by reading the current controller certificate setting
	// The user must provide certificate and key PEM in the config since we can't read them back from the API

	setting, err := r.client.GetControllerCertificateSetting(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error importing controller certificate", err.Error())
		return
	}

	state := ControllerCertificateResourceModel{
		ID:     types.StringValue("controller-cert"),
		CertID: types.StringValue(setting.CertID),
		KeyID:  types.StringValue(setting.KeyID),
		// certificate_pem and private_key_pem cannot be imported (API doesn't expose them)
		// User must provide these in their Terraform config
		CertificateKey: types.StringNull(),
		PrivateKeyKey:  types.StringNull(),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
