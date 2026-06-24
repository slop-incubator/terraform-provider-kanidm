// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package group_posix

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kanidm "github.com/slop-incubator/go-kanidm/kanidm"
)

var _ resource.Resource = &groupPosixResource{}

type groupPosixResource struct {
	client *kanidm.Client
}

// GroupPosixResourceModel maps the Terraform schema to Go types.
type GroupPosixResourceModel struct {
	// group_id references kanidm_group.id (the UUID).
	GroupID   types.String `tfsdk:"group_id"`
	GIDNumber types.Int64  `tfsdk:"gid_number"`
}

func NewResource() resource.Resource {
	return &groupPosixResource{}
}

func (r *groupPosixResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_posix"
}

func (r *groupPosixResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Enables and manages POSIX attributes on a Kanidm group.

This is a separate resource from ` + "`kanidm_group`" + ` because the POSIX extension is optional and
toggled independently by the API.

` + "```" + `hcl
resource "kanidm_group" "developers" {
  name = "developers"
}

resource "kanidm_group_posix" "developers" {
  group_id = kanidm_group.developers.id
}
` + "```" + ``,

		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the `kanidm_group` resource to extend with POSIX attributes.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"gid_number": schema.Int64Attribute{
				MarkdownDescription: "POSIX GID number. Auto-allocated by Kanidm if not specified.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *groupPosixResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *groupPosixResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupPosixResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	posixReq := kanidm.EnableGroupPosixRequest{
		UUID: plan.GroupID.ValueString(),
	}
	if !plan.GIDNumber.IsNull() && !plan.GIDNumber.IsUnknown() {
		gid := plan.GIDNumber.ValueInt64()
		posixReq.GIDNumber = &gid
	}

	result, err := r.client.Groups.EnableGroupPosix(ctx, posixReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to enable POSIX attributes on group", err.Error())
		return
	}

	plan.GIDNumber = types.Int64Value(result.GIDNumber)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupPosixResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupPosixResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	posix, err := r.client.Groups.GetGroupPosix(ctx, state.GroupID.ValueString())
	if errors.Is(err, kanidm.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read group POSIX attributes", err.Error())
		return
	}

	state.GIDNumber = types.Int64Value(posix.GIDNumber)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupPosixResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupPosixResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := kanidm.UpdateGroupPosixRequest{UUID: plan.GroupID.ValueString()}
	if !plan.GIDNumber.IsNull() {
		gid := plan.GIDNumber.ValueInt64()
		updateReq.GIDNumber = &gid
	}

	result, err := r.client.Groups.UpdateGroupPosix(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update group POSIX attributes", err.Error())
		return
	}

	plan.GIDNumber = types.Int64Value(result.GIDNumber)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupPosixResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupPosixResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Groups.DisableGroupPosix(ctx, state.GroupID.ValueString())
	if errors.Is(err, kanidm.ErrNotFound) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to remove group POSIX attributes", err.Error())
	}
}
