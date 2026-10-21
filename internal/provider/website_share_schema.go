// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The pages a share can expose, from SHARE_NAV_ITEMS in Umami's website settings.
var shareViews = []string{
	"overview", "events", "sessions", "realtime", "performance", "compare", "breakdown",
	"goals", "funnels", "journeys", "retention", "utm", "revenue", "attribution",
}

type WebsiteShareModel struct {
	ID          types.String `tfsdk:"id"`
	WebsiteID   types.String `tfsdk:"website_id"`
	Name        types.String `tfsdk:"name"`
	Slug        types.String `tfsdk:"slug"`
	Views       types.Set    `tfsdk:"views"`
	AllowFilter types.Bool   `tfsdk:"allow_filter"`
	Theme       types.String `tfsdk:"theme"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func websiteShareID(websiteID, shareID string) string {
	return websiteID + "/" + shareID
}

func expandShareRequest(ctx context.Context, model WebsiteShareModel) (ShareRequest, diag.Diagnostics) {
	var views []string
	diags := model.Views.ElementsAs(ctx, &views, false)

	selected := make(map[string]bool, len(views))
	for _, v := range views {
		selected[v] = true
	}
	params := make(map[string]any, len(shareViews)+2)
	for _, v := range shareViews {
		params[v] = selected[v]
	}
	params["allowFilter"] = model.AllowFilter.ValueBool()
	if !model.Theme.IsNull() {
		params["theme"] = model.Theme.ValueString()
	}

	req := ShareRequest{Name: model.Name.ValueString(), Parameters: params}
	if !model.Slug.IsNull() && !model.Slug.IsUnknown() {
		req.Slug = model.Slug.ValueString()
	}
	return req, diags
}

func flattenWebsiteShare(share *Share) (WebsiteShareModel, diag.Diagnostics) {
	views := make([]string, 0, len(shareViews))
	for _, v := range shareViews {
		if enabled, _ := share.Parameters[v].(bool); enabled {
			views = append(views, v)
		}
	}
	viewSet, diags := types.SetValueFrom(context.Background(), types.StringType, views)

	allowFilter := true
	if v, ok := share.Parameters["allowFilter"].(bool); ok {
		allowFilter = v
	}
	theme := types.StringNull()
	if v, ok := share.Parameters["theme"].(string); ok && v != "" {
		theme = types.StringValue(v)
	}

	return WebsiteShareModel{
		ID:          types.StringValue(websiteShareID(share.EntityID, share.ID)),
		WebsiteID:   types.StringValue(share.EntityID),
		Name:        types.StringValue(share.Name),
		Slug:        types.StringValue(share.Slug),
		Views:       viewSet,
		AllowFilter: types.BoolValue(allowFilter),
		Theme:       theme,
		CreatedAt:   types.StringPointerValue(share.CreatedAt),
	}, diags
}
