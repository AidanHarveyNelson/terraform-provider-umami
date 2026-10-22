// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	envEndpoint = "UMAMI_API_CLIENT_ENDPOINT"
	envAPIKey   = "UMAMI_API_KEY"
	envUsername = "UMAMI_USERNAME"
	envPassword = "UMAMI_PASSWORD"
)

// Ensure UmamiProvider satisfies various provider interfaces.
var _ provider.Provider = &UmamiProvider{}

// UmamiProvider defines the provider implementation.
type UmamiProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// UmamiProviderModel describes the provider data model.
type UmamiProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIKey   types.String `tfsdk:"api_key"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (p *UmamiProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "umami"
	resp.Version = p.version
}

func (p *UmamiProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage [Umami](https://umami.is/) analytics websites, teams, team members and share links. Authenticate with an API key (Umami v3.4.0+ or Umami Cloud), or with a username and password (self-hosted only). If both are set, the API key is tried first and login is used if Umami rejects it, as versions before v3.4.0 do.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Umami API base URL, including the API path: `https://<your-umami>/api` when self-hosted, or `https://api.umami.is/v1` for Umami Cloud. Also set by `UMAMI_API_CLIENT_ENDPOINT`.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Umami API key (`umami_...`), created under Settings → API keys. Also set by `UMAMI_API_KEY`.",
				Optional:            true,
				Sensitive:           true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username to log in with, for self-hosted instances. The user must not have two-factor authentication enabled. Also set by `UMAMI_USERNAME`.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password to log in with. Also set by `UMAMI_PASSWORD`.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *UmamiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data UmamiProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for attr, value := range map[string]types.String{
		"endpoint": data.Endpoint,
		"api_key":  data.APIKey,
		"username": data.Username,
		"password": data.Password,
	} {
		if value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root(attr),
				"Unknown Umami provider configuration",
				"The provider cannot be configured with a value that is unknown until apply. Set it statically or through its environment variable.",
			)
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := ClientConfig{
		Endpoint:  configOrEnv(data.Endpoint, envEndpoint),
		APIKey:    configOrEnv(data.APIKey, envAPIKey),
		Username:  configOrEnv(data.Username, envUsername),
		Password:  configOrEnv(data.Password, envPassword),
		UserAgent: "terraform-provider-umami/" + p.version,
	}

	if cfg.Endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing Umami API endpoint",
			"Set the endpoint attribute or the "+envEndpoint+" environment variable.",
		)
	}
	if cfg.APIKey == "" && (cfg.Username == "" || cfg.Password == "") {
		resp.Diagnostics.AddError(
			"Missing Umami credentials",
			"Set api_key (or "+envAPIKey+"), or both username and password (or "+envUsername+" and "+envPassword+").",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := NewClient(ctx, cfg)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Umami API client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func configOrEnv(value types.String, envVar string) string {
	if !value.IsNull() && !value.IsUnknown() {
		if v := strings.TrimSpace(value.ValueString()); v != "" {
			return v
		}
	}
	return strings.TrimSpace(os.Getenv(envVar))
}

func (p *UmamiProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTeamMemberResource,
		NewTeamResource,
		NewWebsiteResource,
		NewWebsiteShareResource,
	}
}

func (p *UmamiProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewMeDataSource,
		NewTeamDataSource,
		NewWebsiteDataSource,
		NewWebsitesDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &UmamiProvider{
			version: version,
		}
	}
}
