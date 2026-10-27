// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPixelResource(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc")

	config := func(name, slug string) string {
		return fmt.Sprintf(`
resource "umami_team" "test" {
  name = %[1]q
}

resource "umami_pixel" "test" {
  name    = %[2]q
  slug    = %[3]q
  team_id = umami_team.test.id
}
`, testID, name, slug)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPixelDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config: config("Newsletter", testID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("umami_pixel.test", "id", uuidRegex),
					resource.TestCheckResourceAttr("umami_pixel.test", "name", "Newsletter"),
					resource.TestCheckResourceAttr("umami_pixel.test", "slug", testID),
					resource.TestCheckResourceAttrPair("umami_pixel.test", "team_id", "umami_team.test", "id"),
					resource.TestCheckNoResourceAttr("umami_pixel.test", "user_id"),
					resource.TestCheckResourceAttrSet("umami_pixel.test", "created_at"),
				),
			},
			{
				ResourceName:      "umami_pixel.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: config("Newsletter v2", testID+"-2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("umami_pixel.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("umami_pixel.test", "name", "Newsletter v2"),
					resource.TestCheckResourceAttr("umami_pixel.test", "slug", testID+"-2"),
				),
			},
		},
	})
}

func TestAccPixelResourceDisappears(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_pixel" "test" {
  name = "Newsletter"
  slug = %q
}
`, testID),
				Check: func(s *terraform.State) error {
					return testAccClient(t).DeletePixel(context.Background(), s.RootModule().Resources["umami_pixel.test"].Primary.ID)
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckPixelDestroyed(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "umami_pixel" {
				continue
			}
			if _, err := client.GetPixel(context.Background(), rs.Primary.ID); !isNotFoundError(err) {
				return fmt.Errorf("pixel %s still exists (err: %v)", rs.Primary.ID, err)
			}
		}
		return nil
	}
}
