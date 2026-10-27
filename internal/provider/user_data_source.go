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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource                     = (*UserDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*UserDataSource)(nil)
)

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

type UserDataSource struct {
	client *Client
}

type UserDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Username  types.String `tfsdk:"username"`
	Role      types.String `tfsdk:"role"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an Umami user by ID or username, for example to add an existing user to a team with `umami_team_member`. Needs an admin authenticated with `username` and `password`, because Umami refuses API keys for users and only admins can look up other users.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID (UUID) of the user. One of `id` or `username` is required.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.RegexMatches(uuidRegex, "must be a UUID")},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username of the user, matched without regard to case. One of `id` or `username` is required.",
				Optional:            true,
				Computed:            true,
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "Global role of the user: `admin`, `user` or `view-only`.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp (RFC 3339).",
				Computed:            true,
			},
		},
	}
}

func (d *UserDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("username")),
	}
}

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSource(req, resp); client != nil {
		d.client = client
	}
}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config UserDataSourceModel
	if resp.Diagnostics.Append(req.Config.Get(ctx, &config)...); resp.Diagnostics.HasError() {
		return
	}

	var user *User
	if !config.Username.IsNull() {
		username := strings.ToLower(config.Username.ValueString())
		users, err := d.client.ListUsers(ctx, username)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list users", err.Error())
			return
		}
		for i := range users {
			if users[i].Username == username {
				user = &users[i]
				break
			}
		}
		if user == nil {
			resp.Diagnostics.AddError("User Not Found", fmt.Sprintf("No user has username %q.", username))
			return
		}
	} else {
		var err error
		user, err = d.client.GetUser(ctx, config.ID.ValueString())
		if err != nil {
			if isNotFoundError(err) {
				resp.Diagnostics.AddError("User Not Found", fmt.Sprintf("User %s does not exist.", config.ID.ValueString()))
				return
			}
			resp.Diagnostics.AddError("Failed to get user", err.Error())
			return
		}
	}

	state := UserDataSourceModel{
		ID:        types.StringValue(user.ID),
		Username:  types.StringValue(user.Username),
		Role:      types.StringValue(user.Role),
		CreatedAt: types.StringPointerValue(user.CreatedAt),
	}
	if !config.Username.IsNull() {
		state.Username = config.Username
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
