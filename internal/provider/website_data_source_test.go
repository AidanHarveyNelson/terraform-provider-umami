// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebsiteDataSource(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "umami_website" "missing" {
  domain = "missing.%s.example.com"
}
`, testID),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`No website with domain`),
			},
			{
				Config: `
data "umami_website" "both" {
  id     = "8c3e2f1a-5b6d-4e7f-9a0b-1c2d3e4f5a6b"
  domain = "example.com"
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
			{
				Config: fmt.Sprintf(`
resource "umami_website" "test" {
  name   = %[1]q
  domain = "%[1]s.example.com"
}

data "umami_website" "test" {
  id = umami_website.test.id
}

data "umami_website" "by_domain" {
  domain = umami_website.test.domain
}
`, testID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.umami_website.test", "name", "umami_website.test", "name"),
					resource.TestCheckResourceAttrPair("data.umami_website.test", "domain", "umami_website.test", "domain"),
					resource.TestCheckResourceAttrPair("data.umami_website.test", "user_id", "umami_website.test", "user_id"),
					resource.TestCheckResourceAttr("data.umami_website.test", "replay_config.replay_enabled", "false"),
					resource.TestCheckResourceAttrPair("data.umami_website.by_domain", "id", "umami_website.test", "id"),
					resource.TestCheckResourceAttrPair("data.umami_website.by_domain", "name", "umami_website.test", "name"),
				),
			},
		},
	})
}
