// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ datasource.DataSource                     = (*WebsiteDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*WebsiteDataSource)(nil)
)

func NewWebsiteDataSource() datasource.DataSource {
	return &WebsiteDataSource{}
}

type WebsiteDataSource struct {
	client *Client
}

func (d *WebsiteDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_website"
}

func (d *WebsiteDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an existing Umami website by ID or by domain. Lookups by domain only see websites the authenticated user owns, or that belong to a team where the user is owner or manager.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID (UUID) of the website. One of `id` or `domain` is required.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex, "must be a UUID"),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the website.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Domain of the website, matched exactly. One of `id` or `domain` is required.",
				Optional:            true,
				Computed:            true,
			},
			"team_id": schema.StringAttribute{
				MarkdownDescription: "ID of the team that owns the website, if any.",
				Computed:            true,
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "ID of the user that owns the website, if any.",
				Computed:            true,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "ID of the user that created the website.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp (RFC 3339).",
				Computed:            true,
			},
			"replay_config": schema.SingleNestedAttribute{
				MarkdownDescription: "Session replay and heatmap settings.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"replay_enabled":      schema.BoolAttribute{MarkdownDescription: "Whether session replays are recorded.", Computed: true},
					"heatmap_enabled":     schema.BoolAttribute{MarkdownDescription: "Whether heatmaps are recorded.", Computed: true},
					"sample_rate":         schema.Float64Attribute{MarkdownDescription: "Share of sessions replays are recorded for.", Computed: true},
					"heatmap_sample_rate": schema.Float64Attribute{MarkdownDescription: "Share of sessions heatmaps are recorded for.", Computed: true},
					"mask_level":          schema.StringAttribute{MarkdownDescription: "How much page content replays hide.", Computed: true},
					"max_duration":        schema.Int64Attribute{MarkdownDescription: "Longest replay recorded, in milliseconds.", Computed: true},
					"block_selector":      schema.StringAttribute{MarkdownDescription: "CSS selector for elements left out of replays.", Computed: true},
				},
			},
		},
	}
}

func (d *WebsiteDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("domain")),
	}
}

func (d *WebsiteDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSource(req, resp); client != nil {
		d.client = client
	}
}

func (d *WebsiteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WebsiteModel
	if resp.Diagnostics.Append(req.Config.Get(ctx, &config)...); resp.Diagnostics.HasError() {
		return
	}

	if !config.Domain.IsNull() {
		d.readByDomain(ctx, config.Domain.ValueString(), resp)
		return
	}

	website, err := d.client.GetWebsite(ctx, config.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.Diagnostics.AddError("Website Not Found", fmt.Sprintf("Website %s does not exist or is not accessible with the configured credentials.", config.ID.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Failed to get website", err.Error())
		return
	}

	state := flattenWebsite(website)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *WebsiteDataSource) readByDomain(ctx context.Context, domain string, resp *datasource.ReadResponse) {
	websites, err := d.client.ListWebsites(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list websites", err.Error())
		return
	}

	var matches []Website
	for _, w := range websites {
		if w.Domain != nil && *w.Domain == domain {
			matches = append(matches, w)
		}
	}
	switch len(matches) {
	case 0:
		resp.Diagnostics.AddError("Website Not Found", fmt.Sprintf("No website with domain %q is accessible with the configured credentials.", domain))
		return
	case 1:
	default:
		ids := make([]string, len(matches))
		for i, w := range matches {
			ids[i] = w.ID
		}
		resp.Diagnostics.AddError("Multiple Websites Found", fmt.Sprintf("%d websites have domain %q: %s. Look the website up by id instead.", len(matches), domain, strings.Join(ids, ", ")))
		return
	}

	state := flattenWebsite(&matches[0])
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
