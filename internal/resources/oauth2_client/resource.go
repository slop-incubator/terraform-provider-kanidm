// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package oauth2_client

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kanidm "github.com/slop-incubator/go-kanidm/kanidm"
)

var _ resource.Resource = &oauth2ClientResource{}
var _ resource.ResourceWithImportState = &oauth2ClientResource{}

type oauth2ClientResource struct {
	client *kanidm.Client
}

// OAuth2ClientResourceModel maps the Terraform schema to Go types.
type OAuth2ClientResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	DisplayName  types.String `tfsdk:"display_name"`
	Origin       types.String `tfsdk:"origin"`
	RedirectURIs types.Set    `tfsdk:"redirect_uris"`
	Type         types.String `tfsdk:"type"`
	// Sensitive — never appears in plan output, stored encrypted in state.
	ClientSecret types.String `tfsdk:"client_secret"`
	PKCEMethod   types.String `tfsdk:"pkce_method"`
}

func NewResource() resource.Resource {
	return &oauth2ClientResource{}
}

func (r *oauth2ClientResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth2_client"
}

func (r *oauth2ClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a Kanidm OAuth2 client (also called a relying party).

## Client Types

- **basic** — Confidential client with a client secret. The secret is written to Terraform state.
  Use an encrypted remote backend (S3 + SSE-KMS, Vault, Terraform Cloud) and **never commit plain state files**.
- **public** — Public client without a secret. Use with PKCE (set ` + "`pkce_method = \"S256\"`" + `).

## Secrets Management

The ` + "`client_secret`" + ` is marked sensitive and will not appear in plan output, but it is stored in
Terraform state. Protect your state file using a secrets-capable backend.

## Import

` + "```" + `shell
tofu import kanidm_oauth2_client.example my-app
` + "```" + ``,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the OAuth2 client, assigned by Kanidm.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Short name of the OAuth2 client.",
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
				MarkdownDescription: "Origin URL of the client application.",
				Required:            true,
			},
			"redirect_uris": schema.SetAttribute{
				MarkdownDescription: "Set of allowed redirect URIs for the OAuth2 authorization flow.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Client type: `\"basic\"` (confidential) or `\"public\"`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("basic", "public"),
				},
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Client secret for `basic` clients. Computed by Kanidm. " +
					"**Sensitive** — stored in state; use an encrypted backend.",
				Computed:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pkce_method": schema.StringAttribute{
				MarkdownDescription: "PKCE method to require. Set to `\"S256\"` for public clients. Leave unset to not require PKCE.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("S256"),
				},
			},
		},
	}
}

func (r *oauth2ClientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *oauth2ClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OAuth2ClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var redirectURIs []string
	resp.Diagnostics.Append(plan.RedirectURIs.ElementsAs(ctx, &redirectURIs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := kanidm.CreateOAuth2ClientRequest{
		Name:         plan.Name.ValueString(),
		DisplayName:  plan.DisplayName.ValueString(),
		Origin:       plan.Origin.ValueString(),
		RedirectURIs: redirectURIs,
		Type:         plan.Type.ValueString(),
		PKCEMethod:   plan.PKCEMethod.ValueString(),
	}

	c, err := r.client.OAuth2.CreateClient(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create OAuth2 client", err.Error())
		return
	}

	plan.ID = types.StringValue(c.UUID)
	// client_secret is only present for basic clients; store it sensitively.
	if c.ClientSecret != "" {
		plan.ClientSecret = types.StringValue(c.ClientSecret)
	} else {
		plan.ClientSecret = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *oauth2ClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OAuth2ClientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := r.client.OAuth2.GetClient(ctx, state.Name.ValueString())
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
		resp.Diagnostics.AddError("Failed to read OAuth2 client", err.Error())
		return
	}

	state.ID = types.StringValue(c.UUID)
	state.Name = types.StringValue(c.Name)
	state.DisplayName = types.StringValue(c.DisplayName)
	state.Origin = types.StringValue(c.Origin)
	state.Type = types.StringValue(c.Type)

	if c.PKCEMethod != "" {
		state.PKCEMethod = types.StringValue(c.PKCEMethod)
	}

	redirectVals := make([]types.String, len(c.RedirectURIs))
	for i, u := range c.RedirectURIs {
		redirectVals[i] = types.StringValue(u)
	}
	// Keep client_secret from state (API does not return it on GET).

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *oauth2ClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OAuth2ClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var redirectURIs []string
	resp.Diagnostics.Append(plan.RedirectURIs.ElementsAs(ctx, &redirectURIs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := kanidm.UpdateOAuth2ClientRequest{
		DisplayName:  plan.DisplayName.ValueString(),
		Origin:       plan.Origin.ValueString(),
		RedirectURIs: redirectURIs,
		PKCEMethod:   plan.PKCEMethod.ValueString(),
	}

	c, err := r.client.OAuth2.UpdateClient(ctx, plan.Name.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update OAuth2 client", err.Error())
		return
	}

	plan.ID = types.StringValue(c.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *oauth2ClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OAuth2ClientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.OAuth2.DeleteClient(ctx, state.Name.ValueString())
	if errors.Is(err, kanidm.ErrNotFound) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete OAuth2 client", err.Error())
	}
}

func (r *oauth2ClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	c, err := r.client.OAuth2.GetClient(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import OAuth2 client", err.Error())
		return
	}

	state := OAuth2ClientResourceModel{
		ID:          types.StringValue(c.UUID),
		Name:        types.StringValue(c.Name),
		DisplayName: types.StringValue(c.DisplayName),
		Origin:      types.StringValue(c.Origin),
		Type:        types.StringValue(c.Type),
		// client_secret cannot be recovered on import — the user must manage it out of band.
		ClientSecret: types.StringNull(),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
