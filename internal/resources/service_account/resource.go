// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package service_account

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kanidm "github.com/slop-incubator/go-kanidm/kanidm"
)

var _ resource.Resource = &serviceAccountResource{}
var _ resource.ResourceWithImportState = &serviceAccountResource{}

type serviceAccountResource struct {
	client *kanidm.Client
}

// ServiceAccountResourceModel maps the Terraform schema to Go types.
type ServiceAccountResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	DisplayName    types.String `tfsdk:"display_name"`
	EntryManagedBy types.String `tfsdk:"entry_managed_by"`
	SPN            types.String `tfsdk:"spn"`
}

func NewResource() resource.Resource {
	return &serviceAccountResource{}
}

func (r *serviceAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (r *serviceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a Kanidm service account.

Service accounts are non-person identities used by automation, applications, and services.
They can have API tokens generated for them.

## Import

` + "```" + `shell
tofu import kanidm_service_account.example myapp@example.com
` + "```" + ``,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the service account, assigned by Kanidm on creation.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The short name (SPN prefix) of the service account, e.g. `myapp`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable display name of the service account.",
				Required:            true,
			},
			"entry_managed_by": schema.StringAttribute{
				MarkdownDescription: "SPN of the group or account that manages this entry.",
				Optional:            true,
			},
			"spn": schema.StringAttribute{
				MarkdownDescription: "Full SPN of the service account, e.g. `myapp@example.com`. Computed by Kanidm.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *serviceAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serviceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServiceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := kanidm.CreateServiceAccountRequest{
		Name:        plan.Name.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
	}
	if !plan.EntryManagedBy.IsNull() && !plan.EntryManagedBy.IsUnknown() {
		createReq.EntryManagedBy = plan.EntryManagedBy.ValueString()
	}

	sa, err := r.client.Accounts.CreateServiceAccount(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create service account", err.Error())
		return
	}

	plan.ID = types.StringValue(sa.UUID)
	plan.SPN = types.StringValue(sa.SPN)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServiceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sa, err := r.client.Accounts.GetServiceAccount(ctx, state.SPN.ValueString())
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
		resp.Diagnostics.AddError("Failed to read service account", err.Error())
		return
	}

	state.ID = types.StringValue(sa.UUID)
	state.Name = types.StringValue(sa.Name)
	state.DisplayName = types.StringValue(sa.DisplayName)
	state.SPN = types.StringValue(sa.SPN)
	if sa.EntryManagedBy != "" {
		state.EntryManagedBy = types.StringValue(sa.EntryManagedBy)
	} else {
		state.EntryManagedBy = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServiceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := kanidm.UpdateServiceAccountRequest{
		DisplayName:    plan.DisplayName.ValueString(),
		EntryManagedBy: plan.EntryManagedBy.ValueString(),
	}

	sa, err := r.client.Accounts.UpdateServiceAccount(ctx, plan.SPN.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update service account", err.Error())
		return
	}

	plan.ID = types.StringValue(sa.UUID)
	plan.SPN = types.StringValue(sa.SPN)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Accounts.DeleteServiceAccount(ctx, state.SPN.ValueString())
	if errors.Is(err, kanidm.ErrNotFound) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete service account", err.Error())
	}
}

func (r *serviceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	sa, err := r.client.Accounts.GetServiceAccount(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import service account", err.Error())
		return
	}

	state := ServiceAccountResourceModel{
		ID:          types.StringValue(sa.UUID),
		Name:        types.StringValue(sa.Name),
		DisplayName: types.StringValue(sa.DisplayName),
		SPN:         types.StringValue(sa.SPN),
	}
	if sa.EntryManagedBy != "" {
		state.EntryManagedBy = types.StringValue(sa.EntryManagedBy)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
