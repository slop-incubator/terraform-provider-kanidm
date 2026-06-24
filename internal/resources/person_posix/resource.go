// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package person_posix

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

var _ resource.Resource = &personPosixResource{}

type personPosixResource struct {
	client *kanidm.Client
}

// PersonPosixResourceModel maps the Terraform schema to Go types.
type PersonPosixResourceModel struct {
	// person_id references kanidm_person.id (the UUID).
	PersonID   types.String `tfsdk:"person_id"`
	UIDNumber  types.Int64  `tfsdk:"uid_number"`
	GIDNumber  types.Int64  `tfsdk:"gid_number"`
	LoginShell types.String `tfsdk:"login_shell"`
	HomeDir    types.String `tfsdk:"home_dir"`
}

func NewResource() resource.Resource {
	return &personPosixResource{}
}

func (r *personPosixResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_person_posix"
}

func (r *personPosixResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Enables and manages POSIX attributes on a Kanidm person account.

This is a separate resource from ` + "`kanidm_person`" + ` because the POSIX extension is optional and
toggled independently by the API. Destroying this resource removes POSIX attributes from
the person but does not delete the person itself.

` + "```" + `hcl
resource "kanidm_person" "alice" {
  name         = "alice"
  display_name = "Alice Example"
}

resource "kanidm_person_posix" "alice" {
  person_id   = kanidm_person.alice.id
  login_shell = "/bin/bash"
  home_dir    = "/home/alice"
}
` + "```" + ``,

		Attributes: map[string]schema.Attribute{
			"person_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the `kanidm_person` resource to extend with POSIX attributes.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"uid_number": schema.Int64Attribute{
				MarkdownDescription: "POSIX UID number. Auto-allocated by Kanidm if not specified.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
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
			"login_shell": schema.StringAttribute{
				MarkdownDescription: "Path to the login shell, e.g. `/bin/bash`.",
				Optional:            true,
			},
			"home_dir": schema.StringAttribute{
				MarkdownDescription: "Path to the home directory, e.g. `/home/alice`.",
				Optional:            true,
			},
		},
	}
}

func (r *personPosixResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *personPosixResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PersonPosixResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	posixReq := kanidm.EnablePersonPosixRequest{
		UUID:       plan.PersonID.ValueString(),
		LoginShell: plan.LoginShell.ValueString(),
		HomeDir:    plan.HomeDir.ValueString(),
	}
	if !plan.UIDNumber.IsNull() && !plan.UIDNumber.IsUnknown() {
		uid := plan.UIDNumber.ValueInt64()
		posixReq.UIDNumber = &uid
	}
	if !plan.GIDNumber.IsNull() && !plan.GIDNumber.IsUnknown() {
		gid := plan.GIDNumber.ValueInt64()
		posixReq.GIDNumber = &gid
	}

	result, err := r.client.Accounts.EnablePersonPosix(ctx, posixReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to enable POSIX attributes", err.Error())
		return
	}

	plan.UIDNumber = types.Int64Value(result.UIDNumber)
	plan.GIDNumber = types.Int64Value(result.GIDNumber)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *personPosixResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PersonPosixResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	posix, err := r.client.Accounts.GetPersonPosix(ctx, state.PersonID.ValueString())
	if errors.Is(err, kanidm.ErrNotFound) {
		// POSIX extension was removed — remove this resource from state.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read POSIX attributes", err.Error())
		return
	}

	state.UIDNumber = types.Int64Value(posix.UIDNumber)
	state.GIDNumber = types.Int64Value(posix.GIDNumber)

	if posix.LoginShell != "" {
		state.LoginShell = types.StringValue(posix.LoginShell)
	}
	if posix.HomeDir != "" {
		state.HomeDir = types.StringValue(posix.HomeDir)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *personPosixResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PersonPosixResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := kanidm.UpdatePersonPosixRequest{
		UUID:       plan.PersonID.ValueString(),
		LoginShell: plan.LoginShell.ValueString(),
		HomeDir:    plan.HomeDir.ValueString(),
	}

	result, err := r.client.Accounts.UpdatePersonPosix(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update POSIX attributes", err.Error())
		return
	}

	plan.UIDNumber = types.Int64Value(result.UIDNumber)
	plan.GIDNumber = types.Int64Value(result.GIDNumber)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *personPosixResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PersonPosixResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Accounts.DisablePersonPosix(ctx, state.PersonID.ValueString())
	if errors.Is(err, kanidm.ErrNotFound) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to remove POSIX attributes", err.Error())
	}
}
