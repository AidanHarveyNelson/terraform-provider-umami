// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type WebsiteModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Domain       types.String `tfsdk:"domain"`
	TeamID       types.String `tfsdk:"team_id"`
	UserID       types.String `tfsdk:"user_id"`
	CreatedBy    types.String `tfsdk:"created_by"`
	CreatedAt    types.String `tfsdk:"created_at"`
	ReplayConfig types.Object `tfsdk:"replay_config"`
}

type ReplayConfigModel struct {
	ReplayEnabled     types.Bool    `tfsdk:"replay_enabled"`
	HeatmapEnabled    types.Bool    `tfsdk:"heatmap_enabled"`
	SampleRate        types.Float64 `tfsdk:"sample_rate"`
	HeatmapSampleRate types.Float64 `tfsdk:"heatmap_sample_rate"`
	MaskLevel         types.String  `tfsdk:"mask_level"`
	MaxDuration       types.Int64   `tfsdk:"max_duration"`
	BlockSelector     types.String  `tfsdk:"block_selector"`
}

var replayConfigAttrTypes = map[string]attr.Type{
	"replay_enabled":      types.BoolType,
	"heatmap_enabled":     types.BoolType,
	"sample_rate":         types.Float64Type,
	"heatmap_sample_rate": types.Float64Type,
	"mask_level":          types.StringType,
	"max_duration":        types.Int64Type,
	"block_selector":      types.StringType,
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

func expandReplayConfig(ctx context.Context, obj types.Object) (json.RawMessage, diag.Diagnostics) {
	if obj.IsNull() {
		return json.RawMessage("null"), nil
	}

	var m ReplayConfigModel
	diags := obj.As(ctx, &m, basetypes.ObjectAsOptions{})
	cfg := map[string]any{
		"replayEnabled":  m.ReplayEnabled.ValueBool(),
		"heatmapEnabled": m.HeatmapEnabled.ValueBool(),
	}
	if !m.SampleRate.IsNull() {
		cfg["sampleRate"] = m.SampleRate.ValueFloat64()
	}
	if !m.HeatmapSampleRate.IsNull() {
		cfg["heatmapSampleRate"] = m.HeatmapSampleRate.ValueFloat64()
	}
	if !m.MaskLevel.IsNull() {
		cfg["maskLevel"] = m.MaskLevel.ValueString()
	}
	if !m.MaxDuration.IsNull() {
		cfg["maxDuration"] = m.MaxDuration.ValueInt64()
	}
	if !m.BlockSelector.IsNull() {
		cfg["blockSelector"] = m.BlockSelector.ValueString()
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		diags.AddError("Error Encoding Replay Config", err.Error())
	}
	return data, diags
}

func flattenReplayConfig(cfg map[string]any) types.Object {
	enabled := func(key string) types.Bool {
		v, _ := cfg[key].(bool)
		return types.BoolValue(v)
	}
	float := func(key string) types.Float64 {
		if v, ok := cfg[key].(float64); ok {
			return types.Float64Value(v)
		}
		return types.Float64Null()
	}
	str := func(key string) types.String {
		if v, ok := cfg[key].(string); ok {
			return types.StringValue(v)
		}
		return types.StringNull()
	}
	maxDuration := types.Int64Null()
	if v, ok := cfg["maxDuration"].(float64); ok {
		maxDuration = types.Int64Value(int64(v))
	}

	return types.ObjectValueMust(replayConfigAttrTypes, map[string]attr.Value{
		"replay_enabled":      enabled("replayEnabled"),
		"heatmap_enabled":     enabled("heatmapEnabled"),
		"sample_rate":         float("sampleRate"),
		"heatmap_sample_rate": float("heatmapSampleRate"),
		"mask_level":          str("maskLevel"),
		"max_duration":        maxDuration,
		"block_selector":      str("blockSelector"),
	})
}

func flattenWebsite(website *Website) WebsiteModel {
	return WebsiteModel{
		ID:           types.StringValue(website.ID),
		Name:         types.StringValue(website.Name),
		Domain:       types.StringPointerValue(website.Domain),
		TeamID:       types.StringPointerValue(website.TeamID),
		UserID:       types.StringPointerValue(website.UserID),
		CreatedBy:    types.StringPointerValue(website.CreatedBy),
		CreatedAt:    types.StringPointerValue(website.CreatedAt),
		ReplayConfig: flattenReplayConfig(website.ReplayConfig),
	}
}
