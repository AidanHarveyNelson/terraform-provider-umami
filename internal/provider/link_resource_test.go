// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccLinkResource(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc")

	config := func(name, url, slug, teamID string) string {
		return fmt.Sprintf(`
resource "umami_team" "test" {
  name = %[1]q
}

resource "umami_link" "test" {
  name    = %[2]q
  url     = %[3]q
  slug    = %[4]q
  team_id = %[5]s
}
`, testID, name, url, slug, teamID)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLinkDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config:      config("Docs", "https://example.com/docs", "short", "null"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`string length must be between 8 and 100`),
			},
			{
				Config: config("Docs", "https://example.com/docs", testID, "null"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("umami_link.test", "id", uuidRegex),
					resource.TestCheckResourceAttr("umami_link.test", "name", "Docs"),
					resource.TestCheckResourceAttr("umami_link.test", "url", "https://example.com/docs"),
					resource.TestCheckResourceAttr("umami_link.test", "slug", testID),
					resource.TestCheckNoResourceAttr("umami_link.test", "team_id"),
					resource.TestMatchResourceAttr("umami_link.test", "user_id", uuidRegex),
					resource.TestCheckResourceAttrSet("umami_link.test", "created_at"),
					testAccCheckLinkRedirects(testID, "https://example.com/docs"),
				),
			},
			{
				ResourceName:      "umami_link.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: config("Pricing", "https://example.com/pricing", testID+"-2", "null"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("umami_link.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("umami_link.test", "name", "Pricing"),
					resource.TestCheckResourceAttr("umami_link.test", "slug", testID+"-2"),
					testAccCheckLinkRedirects(testID+"-2", "https://example.com/pricing"),
				),
			},
			{
				Config: config("Pricing", "https://example.com/pricing", testID+"-2", "umami_team.test.id"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("umami_link.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("umami_link.test", "team_id", "umami_team.test", "id"),
					resource.TestCheckNoResourceAttr("umami_link.test", "user_id"),
				),
			},
		},
	})
}

func TestAccLinkResourceDuplicateSlug(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_link" "first" {
  name = "First"
  url  = "https://example.com/1"
  slug = %[1]q
}

resource "umami_link" "second" {
  name       = "Second"
  url        = "https://example.com/2"
  slug       = %[1]q
  depends_on = [umami_link.first]
}
`, testID),
				ExpectError: regexp.MustCompile(`may already be in use`),
			},
		},
	})
}

func TestAccLinkResourceDisappears(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_link" "test" {
  name = "Docs"
  url  = "https://example.com/docs"
  slug = %q
}
`, testID),
				Check: func(s *terraform.State) error {
					return testAccClient(t).DeleteLink(context.Background(), s.RootModule().Resources["umami_link.test"].Primary.ID)
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// Links are served by the Umami app, outside the /api path.
func testAccCheckLinkRedirects(slug, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		res, err := client.Get(strings.TrimSuffix(os.Getenv(envEndpoint), "/api") + "/q/" + slug)
		if err != nil {
			return err
		}
		defer func() { _ = res.Body.Close() }()
		if got := res.Header.Get("Location"); got != want {
			return fmt.Errorf("/q/%s redirected to %q (HTTP %d), want %q", slug, got, res.StatusCode, want)
		}
		return nil
	}
}

func testAccCheckLinkDestroyed(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "umami_link" {
				continue
			}
			if _, err := client.GetLink(context.Background(), rs.Primary.ID); !isNotFoundError(err) {
				return fmt.Errorf("link %s still exists (err: %v)", rs.Primary.ID, err)
			}
		}
		return nil
	}
}
