// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package oauth2_rs

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kanidm "github.com/slop-incubator/go-kanidm/kanidm"
)

var _ resource.Resource = &oauth2RSResource{}
var _ resource.ResourceWithImportState = &oauth2RSResource{}

type oauth2RSResource struct {
	client *kanidm.Client
}

// OAuth2RSResourceModel maps the Terraform schema to Go types.
type OAuth2RSResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	DisplayName  types.String `tfsdk:"display_name"`
	Origin       types.String `tfsdk:"origin"`
	ScopeMaps    types.Map    `tfsdk:"scope_maps"`
	SupScopeMaps types.Map    `tfsdk:"sup_scope_maps"`
	ClaimMaps    types.List   `tfsdk:"claim_maps"`
}

// ClaimMapModel represents a single claim mapping entry.
type ClaimMapModel struct {
	Name   types.String `tfsdk:"name"`
	Values types.Set    `tfsdk:"values"`
	Group  types.String `tfsdk:"group"`
}

func NewResource() resource.Resource {
	return &oauth2RSResource{}
}

func (r *oauth2RSResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth2_resource_server"
}

func (r *oauth2RSResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	claimMapAttrTypes := map[string]attr.Type{
		"name":   types.StringType,
		"values": types.SetType{ElemType: types.StringType},
		"group":  types.StringType,
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a Kanidm OAuth2 resource server (RS) registration.

A resource server is the protected API that clients obtain tokens for. Scope maps control
which scopes members of a group can request.

## Import

` + "```" + `shell
tofu import kanidm_oauth2_resource_server.example my-api
` + "```" + ``,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the resource server, assigned by Kanidm.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Short name of the resource server.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable display name shown in consent screens.",
				Required:            true,
			},
			"origin": schema.StringAttribute{
				MarkdownDescription: "Origin URL of the resource server (e.g. `https://api.example.com`).",
				Required:            true,
			},
			"scope_maps": schema.MapAttribute{
				MarkdownDescription: "Map of group SPN → set of scopes. Members of each group can request the mapped scopes.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.SetType{ElemType: types.StringType},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"sup_scope_maps": schema.MapAttribute{
				MarkdownDescription: "Supplemental scope maps — same structure as `scope_maps`, for additional scopes.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.SetType{ElemType: types.StringType},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"claim_maps": schema.ListNestedAttribute{
				MarkdownDescription: "Custom claim mappings that inject additional claims into tokens.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the claim to inject.",
							Required:            true,
						},
						"values": schema.SetAttribute{
							MarkdownDescription: "Claim values.",
							Required:            true,
							ElementType:         types.StringType,
						},
						"group": schema.StringAttribute{
							MarkdownDescription: "Group SPN whose members receive this claim.",
							Required:            true,
						},
					},
				},
			},
		},
	}

	_ = claimMapAttrTypes // used in mappers
}

func (r *oauth2RSResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*kanidm.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *kanidm.Client.")
		return
	}
	r.client = client
}

func (r *oauth2RSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OAuth2RSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := kanidm.CreateOAuth2RSRequest{
		Name:        plan.Name.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		Origin:      plan.Origin.ValueString(),
	}

	rs, err := r.client.OAuth2.CreateResourceServer(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create OAuth2 resource server", err.Error())
		return
	}

	plan.ID = types.StringValue(rs.UUID)

	// TODO: apply scope_maps, sup_scope_maps, claim_maps via subsequent API calls.

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *oauth2RSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OAuth2RSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rs, err := r.client.OAuth2.GetResourceServer(ctx, state.Name.ValueString())
	if errors.Is(err, kanidm.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if errors.Is(err, kanidm.ErrUnauthorized) {
		resp.Diagnostics.AddError("Authentication error",
			"The provider token is expired or invalid. Regenerate the token and update KANIDM_TOKEN.")
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read OAuth2 resource server", err.Error())
		return
	}

	state.ID = types.StringValue(rs.UUID)
	state.Name = types.StringValue(rs.Name)
	state.DisplayName = types.StringValue(rs.DisplayName)
	state.Origin = types.StringValue(rs.Origin)

	// TODO: map scope_maps, sup_scope_maps, claim_maps from rs response.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *oauth2RSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OAuth2RSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := kanidm.UpdateOAuth2RSRequest{
		DisplayName: plan.DisplayName.ValueString(),
		Origin:      plan.Origin.ValueString(),
	}

	rs, err := r.client.OAuth2.UpdateResourceServer(ctx, plan.Name.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update OAuth2 resource server", err.Error())
		return
	}

	plan.ID = types.StringValue(rs.UUID)

	// TODO: reconcile scope_maps, sup_scope_maps, claim_maps.

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *oauth2RSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OAuth2RSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.OAuth2.DeleteResourceServer(ctx, state.Name.ValueString())
	if errors.Is(err, kanidm.ErrNotFound) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete OAuth2 resource server", err.Error())
	}
}

func (r *oauth2RSResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rs, err := r.client.OAuth2.GetResourceServer(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import OAuth2 resource server", err.Error())
		return
	}

	state := OAuth2RSResourceModel{
		ID:          types.StringValue(rs.UUID),
		Name:        types.StringValue(rs.Name),
		DisplayName: types.StringValue(rs.DisplayName),
		Origin:      types.StringValue(rs.Origin),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
