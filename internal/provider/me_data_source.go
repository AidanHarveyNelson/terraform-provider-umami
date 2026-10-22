// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*MeDataSource)(nil)

func NewMeDataSource() datasource.DataSource {
	return &MeDataSource{}
}

type MeDataSource struct {
	client *Client
}

type MeModel struct {
	ID       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Role     types.String `tfsdk:"role"`
	IsAdmin  types.Bool   `tfsdk:"is_admin"`
}

func (d *MeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_me"
}

func (d *MeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The user the provider is authenticated as.",

		Attributes: map[string]schema.Attribute{
			"id":       schema.StringAttribute{MarkdownDescription: "ID of the user.", Computed: true},
			"username": schema.StringAttribute{MarkdownDescription: "Username of the user.", Computed: true},
			"role":     schema.StringAttribute{MarkdownDescription: "Global role of the user: `admin`, `user` or `view-only`.", Computed: true},
			"is_admin": schema.BoolAttribute{MarkdownDescription: "Whether the user is an Umami admin.", Computed: true},
		},
	}
}

func (d *MeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSource(req, resp); client != nil {
		d.client = client
	}
}

func (d *MeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	me, err := d.client.GetMe(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get the authenticated user", err.Error())
		return
	}

	state := MeModel{
		ID:       types.StringValue(me.User.ID),
		Username: types.StringValue(me.User.Username),
		Role:     types.StringValue(me.User.Role),
		IsAdmin:  types.BoolValue(me.User.IsAdmin),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
