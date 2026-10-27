// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConfigDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "umami_config" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.umami_config.test", "cloud_mode", "false"),
					resource.TestCheckResourceAttr("data.umami_config.test", "private_mode", "false"),
					resource.TestCheckResourceAttrSet("data.umami_config.test", "session_deletion_enabled"),
					resource.TestCheckResourceAttr("data.umami_config.test", "tracker_script_name", "script.js"),
					resource.TestCheckResourceAttr("data.umami_config.test", "tracker_script_names.#", "0"),
					resource.TestCheckNoResourceAttr("data.umami_config.test", "links_url"),
				),
			},
		},
	})
}

func TestSplitTrackerScriptNames(t *testing.T) {
	ptr := func(s string) *string { return &s }

	for _, tc := range []struct {
		in   *string
		want []string
	}{
		{nil, []string{}},
		{ptr(""), []string{}},
		{ptr("stats.js"), []string{"stats.js"}},
		{ptr(" stats.js , , app.js "), []string{"stats.js", "app.js"}},
	} {
		if got := splitTrackerScriptNames(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("splitTrackerScriptNames(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
