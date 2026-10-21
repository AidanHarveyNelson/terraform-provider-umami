// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccTeamMemberResource(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")
	userID := os.Getenv(envTestMemberUserID)

	config := func(role string) string {
		return fmt.Sprintf(`
resource "umami_team" "test" {
  name = %q
}

resource "umami_team_member" "test" {
  team_id = umami_team.test.id
  user_id = %q
  role    = %q
}
`, testID, userID, role)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckMember(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamMemberDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config:      config("team-owner"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
			{
				Config: config("team-view-only"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("umami_team_member.test", "user_id", userID),
					resource.TestCheckResourceAttr("umami_team_member.test", "role", "team-view-only"),
					resource.TestCheckResourceAttrPair("umami_team_member.test", "team_id", "umami_team.test", "id"),
					resource.TestMatchResourceAttr("umami_team_member.test", "id", regexp.MustCompile(`^[0-9a-f-]{36}/`+userID+`$`)),
					resource.TestCheckResourceAttrSet("umami_team_member.test", "created_at"),
				),
			},
			{
				ResourceName:      "umami_team_member.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: config("team-manager"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("umami_team_member.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("umami_team_member.test", "role", "team-manager"),
			},
		},
	})
}

func TestAccTeamMemberResourceDisappears(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")
	userID := os.Getenv(envTestMemberUserID)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckMember(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "umami_team" "test" {
  name = %q
}

resource "umami_team_member" "test" {
  team_id = umami_team.test.id
  user_id = %q
  role    = "team-member"
}
`, testID, userID),
				Check: func(s *terraform.State) error {
					rs := s.RootModule().Resources["umami_team_member.test"].Primary
					return testAccClient(t).RemoveTeamMember(context.Background(), rs.Attributes["team_id"], rs.Attributes["user_id"])
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckTeamMemberDestroyed(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "umami_team_member" {
				continue
			}
			teamID, userID := rs.Primary.Attributes["team_id"], rs.Primary.Attributes["user_id"]
			if _, err := client.GetTeamMember(context.Background(), teamID, userID); !isNotFoundError(err) {
				return fmt.Errorf("user %s is still in team %s (err: %v)", userID, teamID, err)
			}
		}
		return nil
	}
}
