// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ resource.Resource                = &LinkResource{}
	_ resource.ResourceWithImportState = &LinkResource{}
)

func NewLinkResource() resource.Resource {
	return &LinkResource{}
}

type LinkResource struct {
	client *Client
}

func (r *LinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_link"
}

// Slugs of links and pixels share these rules; Umami requires at least 8 characters.
func sluggedSlugAttribute(kind, path string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: fmt.Sprintf("Path segment of the %s URL, `<umami url>/%s/<slug>` (or under `links_url`/`pixels_url` of `umami_config` when set). 8 to 100 letters, digits, hyphens and underscores, unique among the %ss of the Umami instance.", kind, path, kind),
		Required:            true,
		Validators: []validator.String{
			stringvalidator.LengthBetween(8, 100),
			stringvalidator.RegexMatches(shareSlugRegex, "must contain only letters, digits, hyphens and underscores"),
		},
	}
}

func sluggedTeamIDAttribute(kind string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: fmt.Sprintf("ID of the team that owns the %[1]s. Owned by the authenticated user when omitted. Umami cannot move a %[1]s to another owner, so changing this replaces it.", kind),
		Optional:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		Validators:          []validator.String{stringvalidator.RegexMatches(uuidRegex, "must be a UUID")},
	}
}

func (r *LinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A tracked short link. Visiting `<umami url>/q/<slug>` records a visit and redirects to `url`. Deleting a team deletes its links. See [documentation](https://docs.umami.is/docs/links) for more details.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID (UUID) of the link.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the link.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 100)},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Destination URL the link redirects to.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 500)},
			},
			"slug":    sluggedSlugAttribute("link", "q"),
			"team_id": sluggedTeamIDAttribute("link"),
			"user_id": schema.StringAttribute{
				MarkdownDescription: "ID of the user that owns the link, when it is not owned by a team.",
				Computed:            true,
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

func (r *LinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResource(req, resp); client != nil {
		r.client = client
	}
}

func (r *LinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LinkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateLink(ctx, LinkRequest{
		Name:   plan.Name.ValueString(),
		URL:    plan.URL.ValueString(),
		Slug:   plan.Slug.ValueString(),
		TeamID: plan.TeamID.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Link", fmt.Sprintf("Could not create link %q: %s", plan.Name.ValueString(), err))
		return
	}

	link, err := r.client.GetLink(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Link", fmt.Sprintf("Could not read link %s: %s", created.ID, err))
		return
	}

	state := flattenLink(link)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	link, err := r.client.GetLink(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Link", fmt.Sprintf("Could not read link %s: %s", state.ID.ValueString(), err))
		return
	}

	state = flattenLink(link)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state LinkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if _, err := r.client.UpdateLink(ctx, id, LinkRequest{
		Name: plan.Name.ValueString(),
		URL:  plan.URL.ValueString(),
		Slug: plan.Slug.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error Updating Link", fmt.Sprintf("Could not update link %s: %s", id, err))
		return
	}

	link, err := r.client.GetLink(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Link", fmt.Sprintf("Could not read link %s: %s", id, err))
		return
	}

	newState := flattenLink(link)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *LinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.DeleteLink(ctx, id); err != nil {
		if _, getErr := r.client.GetLink(ctx, id); isNotFoundError(getErr) {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Link", fmt.Sprintf("Could not delete link %s: %s", id, err))
	}
}

func (r *LinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !uuidRegex.MatchString(req.ID) {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Expected a link ID (UUID), got %q.", req.ID))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
