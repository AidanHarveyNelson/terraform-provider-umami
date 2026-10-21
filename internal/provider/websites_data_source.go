// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*WebsitesDataSource)(nil)

func NewWebsitesDataSource() datasource.DataSource {
	return &WebsitesDataSource{}
}

type WebsitesDataSource struct {
	client *Client
}

type WebsitesModel struct {
	TeamID   types.String          `tfsdk:"team_id"`
	Search   types.String          `tfsdk:"search"`
	Websites []WebsiteSummaryModel `tfsdk:"websites"`
}

type WebsiteSummaryModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Domain    types.String `tfsdk:"domain"`
	TeamID    types.String `tfsdk:"team_id"`
	UserID    types.String `tfsdk:"user_id"`
	CreatedBy types.String `tfsdk:"created_by"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *WebsitesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_websites"
}

func (d *WebsitesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List Umami websites. Without `team_id`, lists the websites the authenticated user owns and those of teams where the user is owner or manager.",

		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				MarkdownDescription: "Only list the websites of this team.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.RegexMatches(uuidRegex, "must be a UUID")},
			},
			"search": schema.StringAttribute{
				MarkdownDescription: "Only list websites whose name or domain contains this text.",
				Optional:            true,
			},
			"websites": schema.ListNestedAttribute{
				MarkdownDescription: "The matching websites.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{MarkdownDescription: "ID (UUID) of the website.", Computed: true},
						"name":       schema.StringAttribute{MarkdownDescription: "Display name of the website.", Computed: true},
						"domain":     schema.StringAttribute{MarkdownDescription: "Domain of the website.", Computed: true},
						"team_id":    schema.StringAttribute{MarkdownDescription: "ID of the team that owns the website, if any.", Computed: true},
						"user_id":    schema.StringAttribute{MarkdownDescription: "ID of the user that owns the website, if any.", Computed: true},
						"created_by": schema.StringAttribute{MarkdownDescription: "ID of the user that created the website.", Computed: true},
						"created_at": schema.StringAttribute{MarkdownDescription: "Creation timestamp (RFC 3339).", Computed: true},
					},
				},
			},
		},
	}
}

func (d *WebsitesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSource(req, resp); client != nil {
		d.client = client
	}
}

func (d *WebsitesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WebsitesModel
	if resp.Diagnostics.Append(req.Config.Get(ctx, &config)...); resp.Diagnostics.HasError() {
		return
	}

	var websites []Website
	var err error
	if !config.TeamID.IsNull() {
		websites, err = d.client.ListTeamWebsites(ctx, config.TeamID.ValueString())
	} else {
		websites, err = d.client.ListWebsites(ctx, config.Search.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to list websites", err.Error())
		return
	}

	search := config.Search.ValueString()
	config.Websites = make([]WebsiteSummaryModel, 0, len(websites))
	for _, w := range websites {
		if search != "" && !containsFold(w.Name, search) && (w.Domain == nil || !containsFold(*w.Domain, search)) {
			continue
		}
		config.Websites = append(config.Websites, WebsiteSummaryModel{
			ID:        types.StringValue(w.ID),
			Name:      types.StringValue(w.Name),
			Domain:    types.StringPointerValue(w.Domain),
			TeamID:    types.StringPointerValue(w.TeamID),
			UserID:    types.StringPointerValue(w.UserID),
			CreatedBy: types.StringPointerValue(w.CreatedBy),
			CreatedAt: types.StringPointerValue(w.CreatedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
