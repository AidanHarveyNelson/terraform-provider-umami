// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	client *Client
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An Umami user, for self-hosted instances. The provider must be authenticated as an admin with `username` and `password`: Umami refuses API keys for managing users, and only admins can create them.\n\n" +
			"~> **Warning:** Deleting a user also deletes the websites, links and pixels it owns, with all of their analytics data. Teams it owns are deleted with their links and pixels, and their websites are left without a team, reachable only by admins. Move anything worth keeping to another user or team first.\n\n" +
			"See [documentation](https://docs.umami.is/docs/api/users) for more details.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID (UUID) of the user.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username, used to log in. Must be lowercase, because Umami lowercases usernames.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(usernameRegex, "must not contain uppercase letters, because Umami lowercases usernames"),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password of the user, at least 8 characters. Umami does not return passwords, so a password changed outside Terraform is not detected, and an imported user has none in state until the next apply sets it.",
				Required:            true,
				Sensitive:           true,
				Validators:          []validator.String{stringvalidator.LengthBetween(8, 255)},
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "Global role of the user: `admin` (manage the whole instance), `user` (create websites and teams) or `view-only` (view what is shared with them). Defaults to `user`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("user"),
				Validators:          []validator.String{stringvalidator.OneOf(userRoles...)},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResource(req, resp); client != nil {
		r.client = client
	}
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := plan.Username.ValueString()
	created, err := r.client.CreateUser(ctx, UserRequest{
		Username: username,
		Password: plan.Password.ValueString(),
		Role:     plan.Role.ValueString(),
	})
	if err != nil {
		detail := fmt.Sprintf("Could not create user %q: %s", username, err)
		if strings.Contains(err.Error(), "already exists") {
			detail += ". Import the existing user with: terraform import <address> <user id>"
		}
		resp.Diagnostics.AddError("Error Creating User", detail)
		return
	}

	user, err := r.client.GetUser(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading User", fmt.Sprintf("Could not read user %s: %s", created.ID, err))
		return
	}

	state := flattenUser(user, plan.Password)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetUser(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading User", fmt.Sprintf("Could not read user %s: %s", state.ID.ValueString(), err))
		return
	}

	state = flattenUser(user, state.Password)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state UserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := UserRequest{
		Username: plan.Username.ValueString(),
		Role:     plan.Role.ValueString(),
	}
	if !plan.Password.Equal(state.Password) {
		body.Password = plan.Password.ValueString()
	}

	id := state.ID.ValueString()
	if _, err := r.client.UpdateUser(ctx, id, body); err != nil {
		resp.Diagnostics.AddError("Error Updating User", fmt.Sprintf("Could not update user %s: %s", id, err))
		return
	}

	user, err := r.client.GetUser(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading User", fmt.Sprintf("Could not read user %s: %s", id, err))
		return
	}

	newState := flattenUser(user, plan.Password)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.DeleteUser(ctx, id); err != nil {
		// Umami answers 500 when deleting a user that no longer exists.
		if _, getErr := r.client.GetUser(ctx, id); isNotFoundError(getErr) {
			return
		}
		resp.Diagnostics.AddError("Error Deleting User", fmt.Sprintf("Could not delete user %s: %s", id, err))
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !uuidRegex.MatchString(req.ID) {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Expected a user ID (UUID), got %q.", req.ID))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
