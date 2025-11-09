// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gshireesh/terraform-provider-shireesh/internal/provider/generated"
	"golang.org/x/oauth2/clientcredentials"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &ScaffoldingProvider{}
var _ provider.ProviderWithFunctions = &ScaffoldingProvider{}
var _ provider.ProviderWithEphemeralResources = &ScaffoldingProvider{}

// ScaffoldingProvider defines the provider implementation.
type ScaffoldingProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ScaffoldingProviderModel describes the provider data model.
type ScaffoldingProviderModel struct {
	APIBaseURL     types.String `tfsdk:"api_base_url"`
	OAuth2TokenURL types.String `tfsdk:"oauth2_token_url"`
	ClientID       types.String `tfsdk:"client_id"`
	ClientSecret   types.String `tfsdk:"client_secret"`
	Scopes         types.List   `tfsdk:"scopes"`
	Insecure       types.Bool   `tfsdk:"insecure"`
	Environment    types.String `tfsdk:"environment"`
}

func (p *ScaffoldingProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "scaffolding"
	resp.Version = p.version
}

func (p *ScaffoldingProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL for the proxy API (overridable for dev/stage).",
				Optional:            true,
			},
			"oauth2_token_url": schema.StringAttribute{
				MarkdownDescription: "OAuth2 token endpoint URL for client credentials flow.",
				Optional:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client ID.",
				Optional:            true,
				Sensitive:           true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client secret.",
				Optional:            true,
				Sensitive:           true,
			},
			"scopes": schema.ListAttribute{
				MarkdownDescription: "Optional OAuth2 scopes.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"insecure": schema.BoolAttribute{
				MarkdownDescription: "Allow insecure connections to proxy (dev only).",
				Optional:            true,
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "Environment name (dev, stage, prod).",
				Optional:            true,
			},
		},
	}
}

func (p *ScaffoldingProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ScaffoldingProviderModel

	// Some tests may invoke Configure with a zero-value req.Config which can cause a panic
	// inside framework internals. Recover and treat it as empty configuration.
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Treat as empty config; no diagnostics added.
			}
		}()
		resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	}()
	if resp.Diagnostics.HasError() {
		return
	}

	apiBase := firstNonEmpty(ts(data.APIBaseURL), os.Getenv("SCAFFOLDING_API_BASE_URL"))
	if apiBase == "" {
		apiBase = generated.DefaultConfig.APIBaseURL
	}
	tokenURL := firstNonEmpty(ts(data.OAuth2TokenURL), os.Getenv("SCAFFOLDING_OAUTH2_TOKEN_URL"))
	clientID := firstNonEmpty(ts(data.ClientID), os.Getenv("SCAFFOLDING_CLIENT_ID"))
	clientSecret := firstNonEmpty(ts(data.ClientSecret), os.Getenv("SCAFFOLDING_CLIENT_SECRET"))
	insecure := tb(data.Insecure, generated.DefaultConfig.Insecure)
	scopes := listStrings(ctx, data.Scopes)
	if len(scopes) == 0 {
		if env := os.Getenv("SCAFFOLDING_SCOPES"); env != "" {
			scopes = strings.Split(env, ",")
		}
	}

	var httpClient *http.Client
	var bearer string
	if tokenURL != "" && clientID != "" && clientSecret != "" {
		cc := clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     tokenURL,
			Scopes:       scopes,
		}
		tok, err := cc.Token(ctx)
		if err == nil {
			bearer = tok.AccessToken
		}
		httpClient = cc.Client(ctx)
	} else {
		// fallback: plain client (unauthenticated)
		httpClient = http.DefaultClient
	}

	// Store a simple API client in the context data for resources and data sources
	client := &generated.APIClient{BaseURL: apiBase, HTTP: httpClient, Insecure: insecure, Bearer: bearer}
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ScaffoldingProvider) Resources(ctx context.Context) []func() resource.Resource {
	// Generated components resources (from component specs)
	return append([]func() resource.Resource{}, generated.Components.Resources...)
}

func (p *ScaffoldingProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return append([]func() ephemeral.EphemeralResource{}, generated.Components.EphemeralResources...)
}

func (p *ScaffoldingProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return append([]func() datasource.DataSource{}, generated.Components.DataSources...)
}

func (p *ScaffoldingProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ScaffoldingProvider{
			version: version,
		}
	}
}

func ts(s types.String) string {
	if s.IsNull() || s.IsUnknown() {
		return ""
	}
	return s.ValueString()
}

func tb(b types.Bool, def bool) bool {
	if b.IsNull() || b.IsUnknown() {
		return def
	}
	return b.ValueBool()
}

func listStrings(ctx context.Context, l types.List) []string {
	if l.IsNull() || l.IsUnknown() {
		return nil
	}
	var vals []types.String
	if diags := l.ElementsAs(ctx, &vals, false); diags.HasError() {
		return nil
	}
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		out = append(out, v.ValueString())
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
