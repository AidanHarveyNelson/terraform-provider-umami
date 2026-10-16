// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WebsiteModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Domain    types.String `tfsdk:"domain"`
	TeamID    types.String `tfsdk:"team_id"`
	UserID    types.String `tfsdk:"user_id"`
	CreatedBy types.String `tfsdk:"created_by"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func expandCreateWebsiteRequest(model WebsiteModel) CreateWebsiteRequest {
	req := CreateWebsiteRequest{
		Name:   model.Name.ValueString(),
		Domain: model.Domain.ValueString(),
		TeamID: model.TeamID.ValueStringPointer(),
	}
	if !model.ID.IsNull() && !model.ID.IsUnknown() {
		req.ID = model.ID.ValueString()
	}
	return req
}

func flattenWebsite(website *Website) WebsiteModel {
	return WebsiteModel{
		ID:        types.StringValue(website.ID),
		Name:      types.StringValue(website.Name),
		Domain:    types.StringPointerValue(website.Domain),
		TeamID:    types.StringPointerValue(website.TeamID),
		UserID:    types.StringPointerValue(website.UserID),
		CreatedBy: types.StringPointerValue(website.CreatedBy),
		CreatedAt: types.StringPointerValue(website.CreatedAt),
	}
}
