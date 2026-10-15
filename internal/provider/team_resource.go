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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &TeamResource{}
	_ resource.ResourceWithImportState = &TeamResource{}
)

func NewTeamResource() resource.Resource {
	return &TeamResource{}
}

type TeamResource struct {
	client *Client
}

func (r *TeamResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (r *TeamResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An Umami team. Websites assigned to a team are visible to its members. The authenticated user becomes the team owner. Deleting a team first moves its remaining websites to the authenticated user, because Umami would otherwise leave them unreachable. See [documentation](https://docs.umami.is/docs/api/teams) for more details.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID (UUID) of the team.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the team.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 50)},
			},
			"access_code": schema.StringAttribute{
				MarkdownDescription: "Code that users can enter to join the team.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *TeamResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResource(req, resp); client != nil {
		r.client = client
	}
}

func (r *TeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TeamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateTeam(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Team", fmt.Sprintf("Could not create team: %s", err))
		return
	}

	team, err := r.client.GetTeam(ctx, created.ID)
	if err != nil {
		if cleanupErr := r.client.DeleteTeam(ctx, created.ID); cleanupErr != nil {
			tflog.Warn(ctx, "Failed to clean up team after create failure", map[string]any{"id": created.ID, "error": cleanupErr.Error()})
			resp.Diagnostics.AddWarning(
				"Team Create Rollback Failed",
				fmt.Sprintf("Team %s was created but could not be read back or deleted: %s. Delete it manually or import it.", created.ID, cleanupErr),
			)
		}
		resp.Diagnostics.AddError("Error Reading Team", fmt.Sprintf("Could not read team ID %s: %s", created.ID, err))
		return
	}

	state := flattenTeam(team)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	team, err := r.client.GetTeam(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Team", fmt.Sprintf("Could not read team ID %s: %s", state.ID.ValueString(), err))
		return
	}

	state = flattenTeam(team)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if _, err := r.client.UpdateTeam(ctx, id, plan.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Updating Team", fmt.Sprintf("Could not update team ID %s: %s", id, err))
		return
	}

	team, err := r.client.GetTeam(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Team", fmt.Sprintf("Could not read team ID %s: %s", id, err))
		return
	}

	newState := flattenTeam(team)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *TeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	// Umami does not detach a deleted team's websites, which leaves them unreachable.
	websites, err := r.client.ListTeamWebsites(ctx, id)
	if err != nil && !isNotFoundError(r.client.resolveMissing(ctx, err)) {
		resp.Diagnostics.AddError("Error Deleting Team", fmt.Sprintf("Could not list the websites of team ID %s: %s", id, err))
		return
	}
	if len(websites) > 0 {
		me, err := r.client.GetMe(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Error Deleting Team", fmt.Sprintf("Could not look up the authenticated user: %s", err))
			return
		}
		names := make([]string, 0, len(websites))
		for _, w := range websites {
			if err := r.client.TransferWebsite(ctx, w.ID, "", me.User.ID); err != nil {
				resp.Diagnostics.AddError("Error Deleting Team", fmt.Sprintf("Could not move website %s out of team ID %s: %s", w.ID, id, err))
				return
			}
			names = append(names, w.Name)
		}
		resp.Diagnostics.AddWarning(
			"Websites Moved Out Of Deleted Team",
			fmt.Sprintf("Moved %d website(s) to user %s before deleting team %s: %s.", len(websites), me.User.Username, id, strings.Join(names, ", ")),
		)
	}

	if err := r.client.DeleteTeam(ctx, id); err != nil {
		if _, getErr := r.client.GetTeam(ctx, id); isNotFoundError(getErr) {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Team", fmt.Sprintf("Could not delete team ID %s: %s", id, err))
	}
}

func (r *TeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
