// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kanidm "github.com/slop-incubator/go-kanidm/kanidm"
)

var _ datasource.DataSource = &groupDataSource{}

type groupDataSource struct {
	client *kanidm.Client
}

type GroupDataSourceModel struct {
	Name           types.String `tfsdk:"name"`
	ID             types.String `tfsdk:"id"`
	SPN            types.String `tfsdk:"spn"`
	EntryManagedBy types.String `tfsdk:"entry_managed_by"`
	Members        types.Set    `tfsdk:"members"`
}

func NewDataSource() datasource.DataSource {
	return &groupDataSource{}
}

func (d *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Kanidm group by name. Use this to reference groups managed outside Terraform.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Short name of the group to look up.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the group.",
				Computed:            true,
			},
			"spn": schema.StringAttribute{
				MarkdownDescription: "Full SPN of the group.",
				Computed:            true,
			},
			"entry_managed_by": schema.StringAttribute{
				MarkdownDescription: "SPN of the managing account or group.",
				Computed:            true,
			},
			"members": schema.SetAttribute{
				MarkdownDescription: "Set of member SPNs.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *groupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config GroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	grp, err := d.client.Groups.GetGroup(ctx, config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read group", err.Error())
		return
	}

	config.ID = types.StringValue(grp.UUID)
	config.Name = types.StringValue(grp.Name)
	config.SPN = types.StringValue(grp.SPN)

	if grp.EntryManagedBy != "" {
		config.EntryManagedBy = types.StringValue(grp.EntryManagedBy)
	} else {
		config.EntryManagedBy = types.StringNull()
	}

	memberVals := make([]attr.Value, len(grp.Members))
	for i, m := range grp.Members {
		memberVals[i] = types.StringValue(m)
	}
	membersSet, diags := types.SetValue(types.StringType, memberVals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Members = membersSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
