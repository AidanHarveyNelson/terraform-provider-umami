// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	userRoles = []string{"admin", "user", "view-only"}

	// Umami lowercases usernames, so an uppercase letter would show as drift.
	usernameRegex = regexp.MustCompile(`^\P{Lu}+$`)
)

type UserModel struct {
	ID        types.String `tfsdk:"id"`
	Username  types.String `tfsdk:"username"`
	Password  types.String `tfsdk:"password"`
	Role      types.String `tfsdk:"role"`
	CreatedAt types.String `tfsdk:"created_at"`
}

// Umami never returns the password, so it is carried over from the plan or state.
func flattenUser(user *User, password types.String) UserModel {
	return UserModel{
		ID:        types.StringValue(user.ID),
		Username:  types.StringValue(user.Username),
		Password:  password,
		Role:      types.StringValue(user.Role),
		CreatedAt: types.StringPointerValue(user.CreatedAt),
	}
}
