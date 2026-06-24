// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package system_config

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kanidm "github.com/slop-incubator/go-kanidm/kanidm"
)

var _ datasource.DataSource = &systemConfigDataSource{}

type systemConfigDataSource struct {
	client *kanidm.Client
}

type SystemConfigDataSourceModel struct {
	// id is required by the framework even for singleton data sources.
	ID             types.String `tfsdk:"id"`
	Domain         types.String `tfsdk:"domain"`
	DisplayName    types.String `tfsdk:"display_name"`
	LDAPEnabled    types.Bool   `tfsdk:"ldap_enabled"`
	BadlistEnabled types.Bool   `tfsdk:"badlist_enabled"`
}

func NewDataSource() datasource.DataSource {
	return &systemConfigDataSource{}
}

func (d *systemConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_config"
}

func (d *systemConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the global Kanidm system configuration. This is a singleton read-only data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Placeholder identifier (always `\"system\"`).",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The Kanidm domain name.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name of the Kanidm instance.",
				Computed:            true,
			},
			"ldap_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the LDAP interface is enabled.",
				Computed:            true,
			},
			"badlist_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the password bad-list is enabled.",
				Computed:            true,
			},
		},
	}
}

func (d *systemConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *systemConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SystemConfigDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := d.client.System.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read system config", err.Error())
		return
	}

	config.ID = types.StringValue("system")
	config.Domain = types.StringValue(cfg.Domain)
	config.DisplayName = types.StringValue(cfg.DisplayName)
	config.LDAPEnabled = types.BoolValue(cfg.LDAPEnabled)
	config.BadlistEnabled = types.BoolValue(cfg.BadlistEnabled)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
