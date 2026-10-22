// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeamDataSource(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")
	userID := os.Getenv(envTestMemberUserID)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckMember(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "umami_team" "missing" {
  name = "%s-missing"
}
`, testID),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Team Not Found`),
			},
			{
				Config: fmt.Sprintf(`
resource "umami_team" "test" {
  name = %[1]q
}

resource "umami_team_member" "test" {
  team_id = umami_team.test.id
  user_id = %[2]q
  role    = "team-view-only"
}

data "umami_team" "by_id" {
  id         = umami_team.test.id
  depends_on = [umami_team_member.test]
}

data "umami_team" "by_name" {
  name       = umami_team.test.name
  depends_on = [umami_team_member.test]
}
`, testID, userID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.umami_team.by_id", "name", "umami_team.test", "name"),
					resource.TestCheckResourceAttrPair("data.umami_team.by_id", "access_code", "umami_team.test", "access_code"),
					resource.TestCheckResourceAttrPair("data.umami_team.by_name", "id", "umami_team.test", "id"),
					resource.TestCheckResourceAttr("data.umami_team.by_id", "members.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("data.umami_team.by_id", "members.*", map[string]string{
						"user_id":  userID,
						"username": "tfacc-member",
						"role":     "team-view-only",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.umami_team.by_name", "members.*", map[string]string{
						"role": "team-owner",
					}),
				),
			},
		},
	})
}
