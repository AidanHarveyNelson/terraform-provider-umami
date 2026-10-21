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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ resource.Resource                = &TeamMemberResource{}
	_ resource.ResourceWithImportState = &TeamMemberResource{}
)

func NewTeamMemberResource() resource.Resource {
	return &TeamMemberResource{}
}

type TeamMemberResource struct {
	client *Client
}

func (r *TeamMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_member"
}

func (r *TeamMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A user's membership of an Umami team. The authenticated user needs the team-owner or team-manager role in the team, and must outrank the member's role to change or remove it. The team owner's own membership cannot be managed. Users themselves are created by an Umami admin. See [documentation](https://docs.umami.is/docs/api/teams) for more details.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the membership, in the form `team_id/user_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				MarkdownDescription: "ID of the team.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.RegexMatches(uuidRegex, "must be a UUID")},
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "ID of the user to add to the team.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.RegexMatches(uuidRegex, "must be a UUID")},
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "Role of the user in the team: `team-manager` (manage the team, its members and websites), `team-member` (manage websites) or `team-view-only` (view analytics only).",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf(teamMemberRoles...)},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp the user joined the team (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *TeamMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResource(req, resp); client != nil {
		r.client = client
	}
}

func (r *TeamMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TeamMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID, userID := plan.TeamID.ValueString(), plan.UserID.ValueString()
	if _, err := r.client.AddTeamMember(ctx, teamID, userID, plan.Role.ValueString()); err != nil {
		detail := fmt.Sprintf("Could not add user %s to team %s: %s", userID, teamID, err)
		if strings.Contains(err.Error(), "already a member") {
			detail += fmt.Sprintf(". Import the existing membership with: terraform import <address> %s", teamMemberID(teamID, userID))
		}
		resp.Diagnostics.AddError("Error Adding Team Member", detail)
		return
	}

	member, err := r.client.GetTeamMember(ctx, teamID, userID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Team Member", fmt.Sprintf("Could not read user %s in team %s: %s", userID, teamID, err))
		return
	}

	state := flattenTeamMember(member)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID, userID := state.TeamID.ValueString(), state.UserID.ValueString()
	member, err := r.client.GetTeamMember(ctx, teamID, userID)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Team Member", fmt.Sprintf("Could not read user %s in team %s: %s", userID, teamID, err))
		return
	}

	state = flattenTeamMember(member)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TeamMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID, userID := plan.TeamID.ValueString(), plan.UserID.ValueString()
	if _, err := r.client.UpdateTeamMember(ctx, teamID, userID, plan.Role.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Updating Team Member", fmt.Sprintf("Could not change the role of user %s in team %s: %s", userID, teamID, err))
		return
	}

	member, err := r.client.GetTeamMember(ctx, teamID, userID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Team Member", fmt.Sprintf("Could not read user %s in team %s: %s", userID, teamID, err))
		return
	}

	state := flattenTeamMember(member)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID, userID := state.TeamID.ValueString(), state.UserID.ValueString()
	if err := r.client.RemoveTeamMember(ctx, teamID, userID); err != nil {
		if _, getErr := r.client.GetTeamMember(ctx, teamID, userID); isNotFoundError(getErr) {
			return
		}
		resp.Diagnostics.AddError("Error Removing Team Member", fmt.Sprintf("Could not remove user %s from team %s: %s", userID, teamID, err))
	}
}

func (r *TeamMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, userID, ok := strings.Cut(req.ID, "/")
	if !ok || !uuidRegex.MatchString(teamID) || !uuidRegex.MatchString(userID) {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Expected team_id/user_id, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), teamID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
}
