// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	kanidm "github.com/slop-incubator/go-kanidm/kanidm"
)

// Ensure KanidmProvider satisfies the provider.Provider interface.
var _ provider.Provider = &KanidmProvider{}
var _ provider.ProviderWithFunctions = &KanidmProvider{}

// KanidmProvider defines the provider implementation.
type KanidmProvider struct {
	version string
}

// KanidmProviderModel maps the provider schema to Go types.
type KanidmProviderModel struct {
	URL           types.String `tfsdk:"url"`
	Token         types.String `tfsdk:"token"`
	TLSSkipVerify types.Bool   `tfsdk:"tls_skip_verify"`
	Timeout       types.String `tfsdk:"timeout"`
}

// New returns a provider.Provider factory function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KanidmProvider{version: version}
	}
}

func (p *KanidmProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kanidm"
	resp.Version = p.version
}

func (p *KanidmProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The **Kanidm** provider manages resources in a [Kanidm](https://kanidm.github.io/kanidm/) identity management server.

## Authentication

The provider requires a bearer token. Create a dedicated service account scoped to only the permissions
your automation needs (e.g. ` + "`idm_group_admins`" + ` for group management) rather than using ` + "`idm_admin`" + `.

` + "```" + `shell
# Generate a token for a service account named 'tofu-automation':
kanidm service-account api-token generate tofu-automation tofu-provider --expiry 365d -o json
` + "```" + `

Supply the token via the ` + "`KANIDM_TOKEN`" + ` environment variable or the ` + "`token`" + ` provider attribute.

## State Security

The provider writes OAuth2 ` + "`client_secret`" + ` values into Terraform state. Use an encrypted remote
backend (e.g. S3 + SSE-KMS, Terraform Cloud, or Vault) and do not commit plain state files.`,

		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the Kanidm instance (e.g. `https://idm.example.com`). " +
					"Can also be set via the `KANIDM_URL` environment variable.",
				Optional: true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "Bearer token for authenticating with Kanidm. " +
					"Can also be set via the `KANIDM_TOKEN` environment variable. " +
					"**Never hardcode this value** — use a variable or environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"tls_skip_verify": schema.BoolAttribute{
				MarkdownDescription: "Disable TLS certificate verification. " +
					"**For development only.** Setting this to `true` against a non-localhost URL will produce a warning. " +
					"Set `KANIDM_TLS_STRICT=1` in CI to treat this as an error.",
				Optional: true,
			},
			"timeout": schema.StringAttribute{
				MarkdownDescription: "HTTP client timeout as a Go duration string (e.g. `\"30s\"`, `\"2m\"`). Defaults to `\"30s\"`.",
				Optional:            true,
			},
		},
	}
}

// Configure builds the Kanidm client and stores it in the provider data.
func (p *KanidmProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config KanidmProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve URL: attribute → env var.
	rawURL := config.URL.ValueString()
	if rawURL == "" {
		rawURL = os.Getenv("KANIDM_URL")
	}
	if rawURL == "" {
		resp.Diagnostics.AddError(
			"Missing Kanidm URL",
			"Set the `url` provider attribute or the `KANIDM_URL` environment variable.",
		)
		return
	}

	// Resolve token: attribute → env var. Never log the token value.
	token := config.Token.ValueString()
	if token == "" {
		token = os.Getenv("KANIDM_TOKEN")
	}
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Kanidm token",
			"Set the `token` provider attribute or the `KANIDM_TOKEN` environment variable.",
		)
		return
	}

	// Resolve timeout.
	timeoutStr := config.Timeout.ValueString()
	if timeoutStr == "" {
		timeoutStr = "30s"
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid timeout",
			"The `timeout` value must be a valid Go duration string (e.g. \"30s\", \"2m\"): "+err.Error(),
		)
		return
	}

	// Handle tls_skip_verify with security guardrails.
	skipVerify := !config.TLSSkipVerify.IsNull() && config.TLSSkipVerify.ValueBool()
	if skipVerify {
		parsed, parseErr := url.Parse(rawURL)
		isLocalhost := parseErr == nil && (parsed.Hostname() == "localhost" ||
			strings.HasPrefix(parsed.Hostname(), "127.") ||
			parsed.Hostname() == "::1")

		if !isLocalhost {
			// If KANIDM_TLS_STRICT is set, escalate to an error.
			if os.Getenv("KANIDM_TLS_STRICT") == "1" {
				resp.Diagnostics.AddError(
					"TLS verification disabled against non-localhost URL",
					"KANIDM_TLS_STRICT=1 is set. Remove `tls_skip_verify = true` or target a local instance.",
				)
				return
			}
			resp.Diagnostics.AddWarning(
				"TLS verification is disabled",
				"tls_skip_verify = true against a non-localhost URL is insecure. "+
					"This setting must not be used in production. "+
					"Set KANIDM_TLS_STRICT=1 in CI to treat this as an error.",
			)
		}
	}

	tflog.Debug(ctx, "Configuring Kanidm provider", map[string]any{
		"url":             rawURL,
		"timeout":         timeoutStr,
		"tls_skip_verify": skipVerify,
		// token deliberately omitted
	})

	client, err := kanidm.New(kanidm.Options{
		BaseURL:       rawURL,
		Token:         token,
		TLSSkipVerify: skipVerify,
		Timeout:       timeout,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create Kanidm client",
			"Could not initialise the Kanidm client: "+err.Error(),
		)
		return
	}

	// Pass the client to resources and data sources.
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *KanidmProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *KanidmProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *KanidmProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}
