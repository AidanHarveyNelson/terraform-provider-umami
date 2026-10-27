// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const defaultTrackerScriptName = "script.js"

var _ datasource.DataSource = (*ConfigDataSource)(nil)

func NewConfigDataSource() datasource.DataSource {
	return &ConfigDataSource{}
}

type ConfigDataSource struct {
	client *Client
}

type ConfigModel struct {
	CloudMode              types.Bool   `tfsdk:"cloud_mode"`
	PrivateMode            types.Bool   `tfsdk:"private_mode"`
	SessionDeletionEnabled types.Bool   `tfsdk:"session_deletion_enabled"`
	TelemetryDisabled      types.Bool   `tfsdk:"telemetry_disabled"`
	UpdatesDisabled        types.Bool   `tfsdk:"updates_disabled"`
	TrackerScriptName      types.String `tfsdk:"tracker_script_name"`
	TrackerScriptNames     types.List   `tfsdk:"tracker_script_names"`
	FaviconURL             types.String `tfsdk:"favicon_url"`
	LinksURL               types.String `tfsdk:"links_url"`
	PixelsURL              types.String `tfsdk:"pixels_url"`
}

func (d *ConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

func (d *ConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Server settings of the Umami instance, from its environment variables. Useful for building the tracking script URL when `TRACKER_SCRIPT_NAME` renames it.",

		Attributes: map[string]schema.Attribute{
			"cloud_mode":               schema.BoolAttribute{MarkdownDescription: "Whether the instance runs in cloud mode (`CLOUD_MODE`).", Computed: true},
			"private_mode":             schema.BoolAttribute{MarkdownDescription: "Whether private mode is on (`PRIVATE_MODE`).", Computed: true},
			"session_deletion_enabled": schema.BoolAttribute{MarkdownDescription: "Whether individual sessions can be deleted, which needs a relational database without ClickHouse.", Computed: true},
			"telemetry_disabled":       schema.BoolAttribute{MarkdownDescription: "Whether telemetry is turned off (`DISABLE_TELEMETRY`).", Computed: true},
			"updates_disabled":         schema.BoolAttribute{MarkdownDescription: "Whether update checks are turned off (`DISABLE_UPDATES`).", Computed: true},
			"tracker_script_name": schema.StringAttribute{
				MarkdownDescription: "Name of the tracking script, as used in the tracking code the Umami UI shows: the first of `tracker_script_names`, or `" + defaultTrackerScriptName + "`.",
				Computed:            true,
			},
			"tracker_script_names": schema.ListAttribute{
				MarkdownDescription: "Custom names the tracking script is served under (`TRACKER_SCRIPT_NAME`). Empty when unset.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"favicon_url": schema.StringAttribute{MarkdownDescription: "Favicon service URL (`FAVICON_URL`), if set.", Computed: true},
			"links_url":   schema.StringAttribute{MarkdownDescription: "Base URL of tracked links (`LINKS_URL`), if set. Otherwise links are served at `<umami url>/q/<slug>`.", Computed: true},
			"pixels_url":  schema.StringAttribute{MarkdownDescription: "Base URL of tracking pixels (`PIXELS_URL`), if set. Otherwise pixels are served at `<umami url>/p/<slug>`.", Computed: true},
		},
	}
}

func (d *ConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSource(req, resp); client != nil {
		d.client = client
	}
}

func (d *ConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	cfg, err := d.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get the Umami config", err.Error())
		return
	}

	names := splitTrackerScriptNames(cfg.TrackerScriptName)
	scriptName := defaultTrackerScriptName
	if len(names) > 0 {
		scriptName = names[0]
	}
	namesList, diags := types.ListValueFrom(ctx, types.StringType, names)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	state := ConfigModel{
		CloudMode:              types.BoolValue(cfg.CloudMode),
		PrivateMode:            types.BoolValue(cfg.PrivateMode),
		SessionDeletionEnabled: types.BoolValue(cfg.SessionDeletionEnabled),
		TelemetryDisabled:      types.BoolValue(cfg.TelemetryDisabled),
		UpdatesDisabled:        types.BoolValue(cfg.UpdatesDisabled),
		TrackerScriptName:      types.StringValue(scriptName),
		TrackerScriptNames:     namesList,
		FaviconURL:             types.StringPointerValue(cfg.FaviconURL),
		LinksURL:               types.StringPointerValue(cfg.LinksURL),
		PixelsURL:              types.StringPointerValue(cfg.PixelsURL),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Umami serves the script under each comma-separated name in TRACKER_SCRIPT_NAME.
func splitTrackerScriptNames(value *string) []string {
	names := []string{}
	if value == nil {
		return names
	}
	for _, name := range strings.Split(*value, ",") {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	return names
}
