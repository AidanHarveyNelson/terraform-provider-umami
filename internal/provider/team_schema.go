// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type TeamModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	AccessCode types.String `tfsdk:"access_code"`
	CreatedAt  types.String `tfsdk:"created_at"`
}

func flattenTeam(team *Team) TeamModel {
	return TeamModel{
		ID:         types.StringValue(team.ID),
		Name:       types.StringValue(team.Name),
		AccessCode: types.StringPointerValue(team.AccessCode),
		CreatedAt:  types.StringPointerValue(team.CreatedAt),
	}
}
