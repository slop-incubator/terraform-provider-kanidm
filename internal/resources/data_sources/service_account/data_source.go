// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kanidm "github.com/slop-incubator/go-kanidm/kanidm"
)

var _ datasource.DataSource = &serviceAccountDataSource{}

type serviceAccountDataSource struct {
	client *kanidm.Client
}

type ServiceAccountDataSourceModel struct {
	SPN            types.String `tfsdk:"spn"`
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	DisplayName    types.String `tfsdk:"display_name"`
	EntryManagedBy types.String `tfsdk:"entry_managed_by"`
}

func NewDataSource() datasource.DataSource {
	return &serviceAccountDataSource{}
}

func (d *serviceAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (d *serviceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Kanidm service account by SPN.",
		Attributes: map[string]schema.Attribute{
			"spn": schema.StringAttribute{
				MarkdownDescription: "Full SPN of the service account, e.g. `myapp@example.com`.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the service account.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Short name of the service account.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the service account.",
				Computed:            true,
			},
			"entry_managed_by": schema.StringAttribute{
				MarkdownDescription: "SPN of the managing account or group.",
				Computed:            true,
			},
		},
	}
}

func (d *serviceAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serviceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ServiceAccountDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sa, err := d.client.Accounts.GetServiceAccount(ctx, config.SPN.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read service account", err.Error())
		return
	}

	config.ID = types.StringValue(sa.UUID)
	config.Name = types.StringValue(sa.Name)
	config.DisplayName = types.StringValue(sa.DisplayName)
	config.SPN = types.StringValue(sa.SPN)
	if sa.EntryManagedBy != "" {
		config.EntryManagedBy = types.StringValue(sa.EntryManagedBy)
	} else {
		config.EntryManagedBy = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
