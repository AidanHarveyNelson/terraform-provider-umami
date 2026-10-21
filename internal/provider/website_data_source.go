// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ datasource.DataSource = (*WebsiteDataSource)(nil)

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
		MarkdownDescription: "Look up an existing Umami website by ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID (UUID) of the website.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex, "must be a UUID"),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the website.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Domain of the website.",
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
