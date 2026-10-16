// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/sha1"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccWebsiteResource(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")
	id := testUUIDv5DNS(testID + ".example.com")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWebsiteDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_website" "test" {
  id     = uuidv5("dns", "%[2]s.example.com")
  name   = %[2]q
  domain = "%[2]s.example.com"
}
`, id, testID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("umami_website.test", "id", id),
					resource.TestCheckResourceAttr("umami_website.test", "name", testID),
					resource.TestCheckResourceAttr("umami_website.test", "domain", testID+".example.com"),
					resource.TestCheckResourceAttrSet("umami_website.test", "user_id"),
					resource.TestCheckNoResourceAttr("umami_website.test", "team_id"),
					resource.TestCheckResourceAttrSet("umami_website.test", "created_at"),
				),
			},
			{
				ResourceName:      "umami_website.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
resource "umami_team" "test" {
  name = %[2]q
}

resource "umami_team" "other" {
  name = "%[2]s-other"
}

resource "umami_website" "test" {
  id      = %[1]q
  name    = "%[2]s-renamed"
  domain  = "renamed.%[2]s.example.com"
  team_id = umami_team.test.id
}
`, id, testID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("umami_website.test", "id", id),
					resource.TestCheckResourceAttr("umami_website.test", "name", testID+"-renamed"),
					resource.TestCheckResourceAttr("umami_website.test", "domain", "renamed."+testID+".example.com"),
					resource.TestCheckResourceAttrPair("umami_website.test", "team_id", "umami_team.test", "id"),
					resource.TestCheckNoResourceAttr("umami_website.test", "user_id"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "umami_team" "test" {
  name = %[2]q
}

resource "umami_team" "other" {
  name = "%[2]s-other"
}

resource "umami_website" "test" {
  id      = %[1]q
  name    = %[2]q
  domain  = "%[2]s.example.com"
  team_id = umami_team.other.id
}
`, id, testID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("umami_website.test", "team_id", "umami_team.other", "id"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "umami_website" "test" {
  id     = %[1]q
  name   = %[2]q
  domain = "%[2]s.example.com"
}
`, id, testID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("umami_website.test", "team_id"),
					resource.TestCheckResourceAttrSet("umami_website.test", "user_id"),
				),
			},
		},
	})
}

func TestAccWebsiteResourceGeneratedID(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWebsiteDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_website" "test" {
  name   = %[1]q
  domain = "%[1]s.example.com"
}
`, testID),
				Check: resource.TestMatchResourceAttr("umami_website.test", "id", uuidRegex),
			},
		},
	})
}

func TestAccWebsiteResourceDisappears(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_website" "test" {
  name   = %[1]q
  domain = "%[1]s.example.com"
}
`, testID),
				Check: func(s *terraform.State) error {
					return testAccClient(t).DeleteWebsite(context.Background(), s.RootModule().Resources["umami_website.test"].Primary.ID)
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckWebsiteDestroyed(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "umami_website" {
				continue
			}
			if _, err := client.GetWebsite(context.Background(), rs.Primary.ID); !isNotFoundError(err) {
				return fmt.Errorf("website %s still exists (err: %v)", rs.Primary.ID, err)
			}
		}
		return nil
	}
}

func testUUIDv5DNS(name string) string {
	ns := []byte{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
	h := sha1.Sum(append(ns, name...))
	h[6] = (h[6] & 0x0f) | 0x50
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}
