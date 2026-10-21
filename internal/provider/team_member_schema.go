// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The team-owner role is held by the team's creator and cannot be assigned through the API.
var teamMemberRoles = []string{"team-manager", "team-member", "team-view-only"}

type TeamMemberModel struct {
	ID        types.String `tfsdk:"id"`
	TeamID    types.String `tfsdk:"team_id"`
	UserID    types.String `tfsdk:"user_id"`
	Role      types.String `tfsdk:"role"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func teamMemberID(teamID, userID string) string {
	return teamID + "/" + userID
}

func flattenTeamMember(member *TeamMember) TeamMemberModel {
	return TeamMemberModel{
		ID:        types.StringValue(teamMemberID(member.TeamID, member.UserID)),
		TeamID:    types.StringValue(member.TeamID),
		UserID:    types.StringValue(member.UserID),
		Role:      types.StringValue(member.Role),
		CreatedAt: types.StringPointerValue(member.CreatedAt),
	}
}
