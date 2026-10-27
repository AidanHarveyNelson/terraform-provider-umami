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
	_ resource.Resource                = &PixelResource{}
	_ resource.ResourceWithImportState = &PixelResource{}
)

func NewPixelResource() resource.Resource {
	return &PixelResource{}
}

type PixelResource struct {
	client *Client
}

func (r *PixelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pixel"
}

func (r *PixelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A tracking pixel: a 1×1 image at `<umami url>/p/<slug>` that records a view each time it is loaded, for emails and pages where the tracking script cannot run. Deleting a team deletes its pixels. See [documentation](https://docs.umami.is/docs/pixels) for more details.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID (UUID) of the pixel.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the pixel.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 100)},
			},
			"slug":    sluggedSlugAttribute("pixel", "p"),
			"team_id": sluggedTeamIDAttribute("pixel"),
			"user_id": schema.StringAttribute{
				MarkdownDescription: "ID of the user that owns the pixel, when it is not owned by a team.",
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

func (r *PixelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResource(req, resp); client != nil {
		r.client = client
	}
}

func (r *PixelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PixelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreatePixel(ctx, PixelRequest{
		Name:   plan.Name.ValueString(),
		Slug:   plan.Slug.ValueString(),
		TeamID: plan.TeamID.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Pixel", fmt.Sprintf("Could not create pixel %q: %s", plan.Name.ValueString(), err))
		return
	}

	pixel, err := r.client.GetPixel(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Pixel", fmt.Sprintf("Could not read pixel %s: %s", created.ID, err))
		return
	}

	state := flattenPixel(pixel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PixelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PixelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pixel, err := r.client.GetPixel(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Pixel", fmt.Sprintf("Could not read pixel %s: %s", state.ID.ValueString(), err))
		return
	}

	state = flattenPixel(pixel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PixelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PixelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if _, err := r.client.UpdatePixel(ctx, id, PixelRequest{
		Name: plan.Name.ValueString(),
		Slug: plan.Slug.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error Updating Pixel", fmt.Sprintf("Could not update pixel %s: %s", id, err))
		return
	}

	pixel, err := r.client.GetPixel(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Pixel", fmt.Sprintf("Could not read pixel %s: %s", id, err))
		return
	}

	newState := flattenPixel(pixel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *PixelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PixelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.DeletePixel(ctx, id); err != nil {
		if _, getErr := r.client.GetPixel(ctx, id); isNotFoundError(getErr) {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Pixel", fmt.Sprintf("Could not delete pixel %s: %s", id, err))
	}
}

func (r *PixelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !uuidRegex.MatchString(req.ID) {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Expected a pixel ID (UUID), got %q.", req.ID))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
