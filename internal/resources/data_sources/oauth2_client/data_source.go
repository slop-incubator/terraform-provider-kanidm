// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package oauth2_client

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kanidm "github.com/slop-incubator/go-kanidm/kanidm"
)

var _ datasource.DataSource = &oauth2ClientDataSource{}

type oauth2ClientDataSource struct {
	client *kanidm.Client
}

type OAuth2ClientDataSourceModel struct {
	Name         types.String `tfsdk:"name"`
	ID           types.String `tfsdk:"id"`
	DisplayName  types.String `tfsdk:"display_name"`
	Origin       types.String `tfsdk:"origin"`
	Type         types.String `tfsdk:"type"`
	RedirectURIs types.Set    `tfsdk:"redirect_uris"`
	PKCEMethod   types.String `tfsdk:"pkce_method"`
}

func NewDataSource() datasource.DataSource {
	return &oauth2ClientDataSource{}
}

func (d *oauth2ClientDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth2_client"
}

func (d *oauth2ClientDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Kanidm OAuth2 client registration by name. " +
			"Note: `client_secret` is not returned by the Kanidm API on read and is therefore not available here.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Short name of the OAuth2 client to look up.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the client.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the client.",
				Computed:            true,
			},
			"origin": schema.StringAttribute{
				MarkdownDescription: "Origin URL of the client.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Client type: `basic` or `public`.",
				Computed:            true,
			},
			"redirect_uris": schema.SetAttribute{
				MarkdownDescription: "Configured redirect URIs.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"pkce_method": schema.StringAttribute{
				MarkdownDescription: "Configured PKCE method, if any.",
				Computed:            true,
			},
		},
	}
}

func (d *oauth2ClientDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*kanidm.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *kanidm.Client.")
		return
	}
	d.client = client
}

func (d *oauth2ClientDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config OAuth2ClientDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := d.client.OAuth2.GetClient(ctx, config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read OAuth2 client", err.Error())
		return
	}

	config.ID = types.StringValue(c.UUID)
	config.Name = types.StringValue(c.Name)
	config.DisplayName = types.StringValue(c.DisplayName)
	config.Origin = types.StringValue(c.Origin)
	config.Type = types.StringValue(c.Type)

	if c.PKCEMethod != "" {
		config.PKCEMethod = types.StringValue(c.PKCEMethod)
	} else {
		config.PKCEMethod = types.StringNull()
	}

	uriVals := make([]types.String, len(c.RedirectURIs))
	for i, u := range c.RedirectURIs {
		uriVals[i] = types.StringValue(u)
	}
	uriSet, diags := types.SetValueFrom(ctx, types.StringType, c.RedirectURIs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.RedirectURIs = uriSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
