// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type LinkModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	URL       types.String `tfsdk:"url"`
	Slug      types.String `tfsdk:"slug"`
	TeamID    types.String `tfsdk:"team_id"`
	UserID    types.String `tfsdk:"user_id"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func flattenLink(link *Link) LinkModel {
	return LinkModel{
		ID:        types.StringValue(link.ID),
		Name:      types.StringValue(link.Name),
		URL:       types.StringValue(link.URL),
		Slug:      types.StringValue(link.Slug),
		TeamID:    types.StringPointerValue(link.TeamID),
		UserID:    types.StringPointerValue(link.UserID),
		CreatedAt: types.StringPointerValue(link.CreatedAt),
	}
}
