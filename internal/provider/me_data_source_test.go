// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMeDataSource(t *testing.T) {
	testID := acctest.RandomWithPrefix("tfacc-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "umami_me" "test" {}

resource "umami_team" "test" {
  name = %q
}

data "umami_team" "test" {
  id = umami_team.test.id
}
`, testID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("data.umami_me.test", "id", uuidRegex),
					resource.TestCheckResourceAttrSet("data.umami_me.test", "username"),
					resource.TestCheckResourceAttrSet("data.umami_me.test", "role"),
					resource.TestCheckResourceAttrSet("data.umami_me.test", "is_admin"),
					resource.TestCheckTypeSetElemAttrPair("data.umami_team.test", "members.0.user_id", "data.umami_me.test", "id"),
				),
			},
		},
	})
}
