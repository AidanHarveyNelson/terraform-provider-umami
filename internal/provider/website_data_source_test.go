// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebsiteDataSource(t *testing.T) {
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

data "umami_website" "test" {
  id = umami_website.test.id
}
`, testID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.umami_website.test", "name", "umami_website.test", "name"),
					resource.TestCheckResourceAttrPair("data.umami_website.test", "domain", "umami_website.test", "domain"),
					resource.TestCheckResourceAttrPair("data.umami_website.test", "user_id", "umami_website.test", "user_id"),
				),
			},
		},
	})
}
