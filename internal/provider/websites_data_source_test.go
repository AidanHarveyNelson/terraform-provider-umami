// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebsitesDataSource(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_team" "test" {
  name = %[1]q
}

resource "umami_website" "personal" {
  name   = "%[1]s-personal"
  domain = "personal.%[1]s.example.com"
}

resource "umami_website" "team" {
  count = 2

  name    = "%[1]s-${count.index}"
  domain  = "${count.index}.%[1]s.example.com"
  team_id = umami_team.test.id
}

data "umami_websites" "team" {
  team_id    = umami_team.test.id
  depends_on = [umami_website.team]
}

data "umami_websites" "search" {
  search     = %[1]q
  depends_on = [umami_website.personal, umami_website.team]
}

data "umami_websites" "personal" {
  search     = "personal.%[1]s"
  depends_on = [umami_website.personal]
}
`, testID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.umami_websites.team", "websites.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("data.umami_websites.team", "websites.*", map[string]string{
						"domain": "0." + testID + ".example.com",
					}),
					resource.TestCheckResourceAttr("data.umami_websites.search", "websites.#", "3"),
					resource.TestCheckResourceAttr("data.umami_websites.personal", "websites.#", "1"),
					resource.TestCheckResourceAttrPair("data.umami_websites.personal", "websites.0.id", "umami_website.personal", "id"),
				),
			},
		},
	})
}
