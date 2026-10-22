// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource                     = (*TeamDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*TeamDataSource)(nil)
)

func NewTeamDataSource() datasource.DataSource {
	return &TeamDataSource{}
}

type TeamDataSource struct {
	client *Client
}

type TeamDataSourceModel struct {
	ID         types.String             `tfsdk:"id"`
	Name       types.String             `tfsdk:"name"`
	AccessCode types.String             `tfsdk:"access_code"`
	CreatedAt  types.String             `tfsdk:"created_at"`
	Members    []TeamMemberSummaryModel `tfsdk:"members"`
}

type TeamMemberSummaryModel struct {
	UserID   types.String `tfsdk:"user_id"`
	Username types.String `tfsdk:"username"`
	Role     types.String `tfsdk:"role"`
}

func (d *TeamDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (d *TeamDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an Umami team the authenticated user belongs to, by ID or by name, with its members.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID (UUID) of the team. One of `id` or `name` is required.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.RegexMatches(uuidRegex, "must be a UUID")},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the team, matched exactly. One of `id` or `name` is required.",
				Optional:            true,
				Computed:            true,
			},
			"access_code": schema.StringAttribute{
				MarkdownDescription: "Code that users can enter to join the team.",
				Computed:            true,
				Sensitive:           true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp (RFC 3339).",
				Computed:            true,
			},
			"members": schema.ListNestedAttribute{
				MarkdownDescription: "Members of the team, including its owner.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id":  schema.StringAttribute{MarkdownDescription: "ID of the user.", Computed: true},
						"username": schema.StringAttribute{MarkdownDescription: "Username of the user.", Computed: true},
						"role":     schema.StringAttribute{MarkdownDescription: "Role of the user in the team.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *TeamDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *TeamDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSource(req, resp); client != nil {
		d.client = client
	}
}

func (d *TeamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TeamDataSourceModel
	if resp.Diagnostics.Append(req.Config.Get(ctx, &config)...); resp.Diagnostics.HasError() {
		return
	}

	team, err := d.findTeam(ctx, config)
	if err != nil {
		if isNotFoundError(err) {
			resp.Diagnostics.AddError("Team Not Found", fmt.Sprintf("No matching team is accessible with the configured credentials: %s", describeTeamLookup(config)))
			return
		}
		resp.Diagnostics.AddError("Failed to get team", err.Error())
		return
	}

	members, err := d.client.ListTeamMembers(ctx, team.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list team members", err.Error())
		return
	}

	state := TeamDataSourceModel{
		ID:         types.StringValue(team.ID),
		Name:       types.StringValue(team.Name),
		AccessCode: types.StringPointerValue(team.AccessCode),
		CreatedAt:  types.StringPointerValue(team.CreatedAt),
		Members:    make([]TeamMemberSummaryModel, 0, len(members)),
	}
	for _, m := range members {
		username := types.StringNull()
		if m.User != nil {
			username = types.StringValue(m.User.Username)
		}
		state.Members = append(state.Members, TeamMemberSummaryModel{
			UserID:   types.StringValue(m.UserID),
			Username: username,
			Role:     types.StringValue(m.Role),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *TeamDataSource) findTeam(ctx context.Context, config TeamDataSourceModel) (*Team, error) {
	if !config.ID.IsNull() {
		return d.client.GetTeam(ctx, config.ID.ValueString())
	}

	teams, err := d.client.ListTeams(ctx)
	if err != nil {
		return nil, err
	}
	var match *Team
	for i := range teams {
		if teams[i].Name != config.Name.ValueString() {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("more than one team is named %q (%s, %s); look the team up by id instead", teams[i].Name, match.ID, teams[i].ID)
		}
		match = &teams[i]
	}
	if match == nil {
		return nil, ErrNotFound
	}
	return match, nil
}

func describeTeamLookup(config TeamDataSourceModel) string {
	if !config.ID.IsNull() {
		return "id " + config.ID.ValueString()
	}
	return fmt.Sprintf("name %q", config.Name.ValueString())
}
