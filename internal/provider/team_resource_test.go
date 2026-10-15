// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccTeamResource(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_team" "test" {
  name = %q
}
`, testID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("umami_team.test", "name", testID),
					resource.TestMatchResourceAttr("umami_team.test", "id", uuidRegex),
					resource.TestCheckResourceAttrSet("umami_team.test", "access_code"),
				),
			},
			{
				ResourceName:      "umami_team.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
resource "umami_team" "test" {
  name = "%s-renamed"
}
`, testID),
				Check: resource.TestCheckResourceAttr("umami_team.test", "name", testID+"-renamed"),
			},
		},
	})
}

func TestAccTeamResourceDeleteKeepsWebsites(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")
	var websiteID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_team" "test" {
  name = %q
}
`, testID),
				Check: func(s *terraform.State) error {
					teamID := s.RootModule().Resources["umami_team.test"].Primary.ID
					w, err := testAccClient(t).CreateWebsite(context.Background(), CreateWebsiteRequest{Name: testID, Domain: testID + ".example.com", TeamID: &teamID})
					if err != nil {
						return err
					}
					websiteID = w.ID
					return nil
				},
			},
			{
				Config: `# team removed`,
				Check: func(s *terraform.State) error {
					c := testAccClient(t)
					w, err := c.GetWebsite(context.Background(), websiteID)
					if err != nil {
						return fmt.Errorf("website lost after team deletion: %w", err)
					}
					if w.TeamID != nil {
						return fmt.Errorf("website still has team %s", *w.TeamID)
					}
					return c.DeleteWebsite(context.Background(), websiteID)
				},
			},
		},
	})
}

func testAccCheckTeamDestroyed(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "umami_team" {
				continue
			}
			if _, err := client.GetTeam(context.Background(), rs.Primary.ID); !isNotFoundError(err) {
				return fmt.Errorf("team %s still exists (err: %v)", rs.Primary.ID, err)
			}
		}
		return nil
	}
}
