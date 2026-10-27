// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUserResource(t *testing.T) {
	username := acctest.RandomWithPrefix("tfacc-")

	config := func(username, password, role string) string {
		return fmt.Sprintf(`
resource "umami_user" "test" {
  username = %q
  password = %q
  role     = %q
}
`, username, password, role)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAdmin(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config:      config(strings.ToUpper(username), "password-1", "user"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must not contain uppercase letters`),
			},
			{
				Config: config(username, "password-1", "view-only"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("umami_user.test", "id", uuidRegex),
					resource.TestCheckResourceAttr("umami_user.test", "username", username),
					resource.TestCheckResourceAttr("umami_user.test", "role", "view-only"),
					resource.TestCheckResourceAttrSet("umami_user.test", "created_at"),
					testAccCheckUserLogin("umami_user.test", "password-1"),
				),
			},
			{
				ResourceName:            "umami_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			{
				Config: config(username+"-renamed", "password-2", "user"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("umami_user.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("umami_user.test", "username", username+"-renamed"),
					resource.TestCheckResourceAttr("umami_user.test", "role", "user"),
					testAccCheckUserLogin("umami_user.test", "password-2"),
				),
			},
		},
	})
}

func TestAccUserResourceDisappears(t *testing.T) {
	username := acctest.RandomWithPrefix("tfacc-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAdmin(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_user" "test" {
  username = %q
  password = "password-1"
}
`, username),
				Check: func(s *terraform.State) error {
					return testAccClient(t).DeleteUser(context.Background(), s.RootModule().Resources["umami_user.test"].Primary.ID)
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// The default test credentials are a non-admin, and with api_key auth an API key, which Umami refuses for users.
func TestAccUserResourceNeedsAdmin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_user" "test" {
  username = %q
  password = "password-1"
}
`, acctest.RandomWithPrefix("tfacc-")),
				ExpectError: regexp.MustCompile(`managing users needs an Umami admin`),
			},
		},
	})
}

func testAccCheckUserLogin(address, password string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := NewClient(context.Background(), ClientConfig{
			Endpoint: os.Getenv(envEndpoint),
			Username: s.RootModule().Resources[address].Primary.Attributes["username"],
			Password: password,
		})
		return err
	}
}

func testAccCheckUserDestroyed(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "umami_user" {
				continue
			}
			if _, err := client.GetUser(context.Background(), rs.Primary.ID); !isNotFoundError(err) {
				return fmt.Errorf("user %s still exists (err: %v)", rs.Primary.ID, err)
			}
		}
		return nil
	}
}
