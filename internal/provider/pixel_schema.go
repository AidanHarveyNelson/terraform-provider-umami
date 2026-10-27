// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PixelModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Slug      types.String `tfsdk:"slug"`
	TeamID    types.String `tfsdk:"team_id"`
	UserID    types.String `tfsdk:"user_id"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func flattenPixel(pixel *Pixel) PixelModel {
	return PixelModel{
		ID:        types.StringValue(pixel.ID),
		Name:      types.StringValue(pixel.Name),
		Slug:      types.StringValue(pixel.Slug),
		TeamID:    types.StringPointerValue(pixel.TeamID),
		UserID:    types.StringPointerValue(pixel.UserID),
		CreatedAt: types.StringPointerValue(pixel.CreatedAt),
	}
}
