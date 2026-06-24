// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package person

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kanidm "github.com/slop-incubator/go-kanidm/kanidm"
)

var _ datasource.DataSource = &personDataSource{}

type personDataSource struct {
	client *kanidm.Client
}

type PersonDataSourceModel struct {
	SPN         types.String `tfsdk:"spn"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Mail        types.Set    `tfsdk:"mail"`
}

func NewDataSource() datasource.DataSource {
	return &personDataSource{}
}

func (d *personDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_person"
}

func (d *personDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Kanidm person account by SPN. Use this to reference persons managed outside Terraform.",
		Attributes: map[string]schema.Attribute{
			"spn": schema.StringAttribute{
				MarkdownDescription: "Full SPN of the person to look up, e.g. `alice@example.com`.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the person.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Short username of the person.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the person.",
				Computed:            true,
			},
			"mail": schema.SetAttribute{
				MarkdownDescription: "Email addresses of the person.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *personDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *personDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PersonDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := d.client.Accounts.GetUser(ctx, config.SPN.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read person", err.Error())
		return
	}

	config.ID = types.StringValue(user.UUID)
	config.Name = types.StringValue(user.Name)
	config.DisplayName = types.StringValue(user.DisplayName)
	config.SPN = types.StringValue(user.SPN)

	mailVals := make([]attr.Value, len(user.Mail))
	for i, m := range user.Mail {
		mailVals[i] = types.StringValue(m)
	}
	mailSet, diags := types.SetValue(types.StringType, mailVals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Mail = mailSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
