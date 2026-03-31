package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-tplink-omada/internal/client"
)

var _ resource.Resource = &IPGroupResource{}
var _ resource.ResourceWithImportState = &IPGroupResource{}

// IPGroupResource manages an IP/Port group on the Omada Controller.
type IPGroupResource struct {
	client *client.Client
}

// IPGroupResourceModel maps the resource schema to Go types.
type IPGroupResourceModel struct {
	ID     types.String        `tfsdk:"id"`
	SiteID types.String        `tfsdk:"site_id"`
	Name   types.String        `tfsdk:"name"`
	Type   types.Int64         `tfsdk:"type"`
	IPList []IPGroupEntryModel `tfsdk:"ip_list"`
}

// IPGroupEntryModel represents a single IP + port combination.
type IPGroupEntryModel struct {
	IP       types.String `tfsdk:"ip"`
	PortList types.List   `tfsdk:"port_list"`
}

func NewIPGroupResource() resource.Resource {
	return &IPGroupResource{}
}

func (r *IPGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_group"
}

func (r *IPGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an IP/Port group on the Omada Controller. " +
			"IP groups are used as source or destination in firewall ACL rules for port-based filtering. " +
			"Requires a gateway device adopted into the site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the IP group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": siteIDResourceSchema(),
			"name": schema.StringAttribute{
				Description: "The name of the IP group.",
				Required:    true,
			},
			"type": schema.Int64Attribute{
				Description: "The group type. Use 1 for IP/Port group.",
				Computed:    true,
			},
			"ip_list": schema.ListNestedAttribute{
				Description: "List of IP address and port combinations.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							Description: "IP address or CIDR subnet (e.g., '192.168.1.100' or '192.168.1.0/24').",
							Required:    true,
						},
						"port_list": schema.ListAttribute{
							Description: "List of port numbers or ranges as strings (e.g., '80', '7000-7100').",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *IPGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IPGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IPGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := plan.SiteID.ValueString()

	ipList := r.modelToIPList(ctx, plan.IPList, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	group := &client.IPGroup{
		Name:   plan.Name.ValueString(),
		Type:   1, // IP/Port group
		IPList: ipList,
	}

	created, err := r.client.CreateIPGroup(ctx, siteID, group)
	if err != nil {
		resp.Diagnostics.AddError("Error creating IP group", err.Error())
		return
	}

	r.setStateFromAPI(ctx, &plan, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IPGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IPGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()

	group, err := r.client.GetIPGroup(ctx, siteID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading IP group", err.Error())
		return
	}

	r.setStateFromAPI(ctx, &state, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IPGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan IPGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IPGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID := state.SiteID.ValueString()

	ipList := r.modelToIPList(ctx, plan.IPList, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	group := &client.IPGroup{
		Name:   plan.Name.ValueString(),
		Type:   1,
		IPList: ipList,
	}

	updated, err := r.client.UpdateIPGroup(ctx, siteID, state.ID.ValueString(), group)
	if err != nil {
		resp.Diagnostics.AddError("Error updating IP group", err.Error())
		return
	}

	plan.ID = state.ID
	plan.SiteID = state.SiteID
	r.setStateFromAPI(ctx, &plan, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IPGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state IPGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteIPGroup(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting IP group", err.Error())
		return
	}
}

func (r *IPGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	siteID, groupID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'siteID/groupID'.",
		)
		return
	}

	group, err := r.client.GetIPGroup(ctx, siteID, groupID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing IP group", err.Error())
		return
	}

	state := IPGroupResourceModel{
		SiteID: types.StringValue(siteID),
	}
	r.setStateFromAPI(ctx, &state, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// modelToIPList converts the Terraform model to the client IPGroupEntry slice.
func (r *IPGroupResource) modelToIPList(ctx context.Context, entries []IPGroupEntryModel, diags *diag.Diagnostics) []client.IPGroupEntry {
	var ipList []client.IPGroupEntry
	for _, e := range entries {
		entry := client.IPGroupEntry{
			IP: e.IP.ValueString(),
		}
		if !e.PortList.IsNull() && !e.PortList.IsUnknown() {
			var ports []string
			diags.Append(e.PortList.ElementsAs(ctx, &ports, false)...)
			if diags.HasError() {
				return nil
			}
			entry.PortList = ports
		}
		ipList = append(ipList, entry)
	}
	return ipList
}

// setStateFromAPI populates the resource model from an API response.
func (r *IPGroupResource) setStateFromAPI(ctx context.Context, model *IPGroupResourceModel, group *client.IPGroup) {
	model.ID = types.StringValue(group.ID)
	model.Name = types.StringValue(group.Name)
	model.Type = types.Int64Value(int64(group.Type))

	model.IPList = make([]IPGroupEntryModel, len(group.IPList))
	for i, entry := range group.IPList {
		model.IPList[i] = IPGroupEntryModel{
			IP: types.StringValue(entry.IP),
		}
		if len(entry.PortList) > 0 {
			portList, _ := types.ListValueFrom(ctx, types.StringType, entry.PortList)
			model.IPList[i].PortList = portList
		} else {
			model.IPList[i].PortList = types.ListNull(types.StringType)
		}
	}
}
