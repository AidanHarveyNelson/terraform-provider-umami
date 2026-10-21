// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccWebsiteShareWebsite(testID string) string {
	return fmt.Sprintf(`
resource "umami_website" "test" {
  name   = %[1]q
  domain = "%[1]s.example.com"
}
`, testID)
}

func TestAccWebsiteShareResource(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWebsiteShareDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config: testAccWebsiteShareWebsite(testID) + `
resource "umami_website_share" "test" {
  website_id = umami_website.test.id
  name       = "Public dashboard"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("umami_website_share.test", "website_id", "umami_website.test", "id"),
					resource.TestCheckResourceAttr("umami_website_share.test", "name", "Public dashboard"),
					resource.TestMatchResourceAttr("umami_website_share.test", "slug", regexp.MustCompile(`^\w{16}$`)),
					resource.TestCheckResourceAttr("umami_website_share.test", "views.#", "2"),
					resource.TestCheckTypeSetElemAttr("umami_website_share.test", "views.*", "overview"),
					resource.TestCheckTypeSetElemAttr("umami_website_share.test", "views.*", "events"),
					resource.TestCheckResourceAttr("umami_website_share.test", "allow_filter", "true"),
					resource.TestCheckNoResourceAttr("umami_website_share.test", "theme"),
				),
			},
			{
				ResourceName:      "umami_website_share.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccWebsiteShareWebsite(testID) + fmt.Sprintf(`
resource "umami_website_share" "test" {
  website_id   = umami_website.test.id
  name         = "Client dashboard"
  slug         = %q
  views        = ["overview", "realtime", "goals"]
  allow_filter = false
  theme        = "dark"
}
`, testID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("umami_website_share.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("umami_website_share.test", "name", "Client dashboard"),
					resource.TestCheckResourceAttr("umami_website_share.test", "slug", testID),
					resource.TestCheckResourceAttr("umami_website_share.test", "views.#", "3"),
					resource.TestCheckTypeSetElemAttr("umami_website_share.test", "views.*", "goals"),
					resource.TestCheckResourceAttr("umami_website_share.test", "allow_filter", "false"),
					resource.TestCheckResourceAttr("umami_website_share.test", "theme", "dark"),
				),
			},
			{
				Config: testAccWebsiteShareWebsite(testID) + fmt.Sprintf(`
resource "umami_website_share" "test" {
  website_id   = umami_website.test.id
  name         = "Client dashboard"
  slug         = %[1]q
  views        = ["overview", "realtime", "goals"]
  allow_filter = false
  theme        = "dark"
}

resource "umami_website_share" "duplicate" {
  website_id = umami_website.test.id
  name       = "Duplicate"
  slug       = %[1]q
}
`, testID),
				ExpectError: regexp.MustCompile(`may already be in use`),
			},
		},
	})
}

func TestAccWebsiteShareResourceDisappears(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWebsiteShareWebsite(testID) + `
resource "umami_website_share" "test" {
  website_id = umami_website.test.id
  name       = "Public dashboard"
}
`,
				Check: func(s *terraform.State) error {
					_, shareID := splitWebsiteShareID(s.RootModule().Resources["umami_website_share.test"].Primary.ID)
					return testAccClient(t).DeleteShare(context.Background(), shareID)
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckWebsiteShareDestroyed(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "umami_website_share" {
				continue
			}
			websiteID, shareID := splitWebsiteShareID(rs.Primary.ID)
			if _, err := client.GetWebsiteShare(context.Background(), websiteID, shareID); !isNotFoundError(err) {
				return fmt.Errorf("share %s still exists (err: %v)", rs.Primary.ID, err)
			}
		}
		return nil
	}
}
