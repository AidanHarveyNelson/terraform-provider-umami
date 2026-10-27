// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserDataSource(t *testing.T) {
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
  role     = "view-only"
}

# A similar username, so the search returns more than one user.
resource "umami_user" "other" {
  username = "%[1]s-other"
  password = "password-1"
}

data "umami_user" "by_username" {
  username   = %q
  depends_on = [umami_user.test, umami_user.other]
}

data "umami_user" "by_id" {
  id = umami_user.test.id
}
`, username, strings.ToUpper(username)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.umami_user.by_username", "id", "umami_user.test", "id"),
					resource.TestCheckResourceAttr("data.umami_user.by_username", "role", "view-only"),
					resource.TestCheckResourceAttrSet("data.umami_user.by_username", "created_at"),
					resource.TestCheckResourceAttr("data.umami_user.by_id", "username", username),
					resource.TestCheckResourceAttr("data.umami_user.by_id", "role", "view-only"),
				),
			},
		},
	})
}

func TestAccUserDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAdmin(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(`data "umami_user" "test" { username = %q }`, acctest.RandomWithPrefix("tfacc-missing-")),
				ExpectError: regexp.MustCompile(`No user has username`),
			},
			{
				Config:      `data "umami_user" "test" { id = "0f3c9a3e-7f4d-5b8a-9c1e-2d3f4a5b6c7d" }`,
				ExpectError: regexp.MustCompile(`does not exist`),
			},
		},
	})
}
