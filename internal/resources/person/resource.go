// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package person

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kanidm "github.com/slop-incubator/go-kanidm/kanidm"
)

var _ resource.Resource = &personResource{}
var _ resource.ResourceWithImportState = &personResource{}

type personResource struct {
	client *kanidm.Client
}

// PersonResourceModel maps the Terraform schema to Go types.
type PersonResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Mail        types.Set    `tfsdk:"mail"`
	SPN         types.String `tfsdk:"spn"`
}

func NewResource() resource.Resource {
	return &personResource{}
}

func (r *personResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_person"
}

func (r *personResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a Kanidm person account.

## Import

` + "```" + `shell
tofu import kanidm_person.example alice@example.com
` + "```" + ``,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the person, assigned by Kanidm on creation.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The short username (SPN prefix), e.g. `alice`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable display name, e.g. `Alice Example`.",
				Required:            true,
			},
			"mail": schema.SetAttribute{
				MarkdownDescription: "Set of email addresses for the person. Multiple addresses are supported.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"spn": schema.StringAttribute{
				MarkdownDescription: "Full SPN of the person, e.g. `alice@example.com`. Computed by Kanidm.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *personResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *personResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PersonResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mail := stringSetToSlice(ctx, plan.Mail)

	createReq := kanidm.CreatePersonRequest{
		Name:        plan.Name.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		Mail:        mail,
	}

	user, err := r.client.Accounts.CreatePerson(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create person", sanitiseError(err))
		return
	}

	mapUserToState(user, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *personResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PersonResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.Accounts.GetUser(ctx, state.SPN.ValueString())
	if errors.Is(err, kanidm.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if errors.Is(err, kanidm.ErrUnauthorized) {
		resp.Diagnostics.AddError(
			"Authentication error",
			"The provider token is expired or invalid. Regenerate the token and update KANIDM_TOKEN.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read person", sanitiseError(err))
		return
	}

	mapUserToState(user, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *personResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PersonResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mail := stringSetToSlice(ctx, plan.Mail)

	updateReq := kanidm.UpdatePersonRequest{
		DisplayName: plan.DisplayName.ValueString(),
		Mail:        mail,
	}

	user, err := r.client.Accounts.UpdatePerson(ctx, plan.SPN.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update person", sanitiseError(err))
		return
	}

	mapUserToState(user, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *personResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PersonResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Accounts.DeletePerson(ctx, state.SPN.ValueString())
	if errors.Is(err, kanidm.ErrNotFound) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete person", sanitiseError(err))
	}
}

func (r *personResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by SPN (e.g. alice@example.com).
	user, err := r.client.Accounts.GetUser(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import person", sanitiseError(err))
		return
	}

	var state PersonResourceModel
	mapUserToState(user, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- helpers ---

func mapUserToState(user *kanidm.User, model *PersonResourceModel) {
	model.ID = types.StringValue(user.UUID)
	model.Name = types.StringValue(user.Name)
	model.DisplayName = types.StringValue(user.DisplayName)
	model.SPN = types.StringValue(user.SPN)

	mailVals := make([]attr.Value, len(user.Mail))
	for i, m := range user.Mail {
		mailVals[i] = types.StringValue(m)
	}
	mailSet, _ := types.SetValue(types.StringType, mailVals)
	model.Mail = mailSet
}

func stringSetToSlice(ctx context.Context, set types.Set) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var elems []types.String
	_ = set.ElementsAs(ctx, &elems, false)
	result := make([]string, len(elems))
	for i, e := range elems {
		result[i] = e.ValueString()
	}
	return result
}

func sanitiseError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
