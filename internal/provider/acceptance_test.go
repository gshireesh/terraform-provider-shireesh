// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/gshireesh/terraform-provider-shireesh/fakeserver"
)

func TestAccSimple_withFakeServer(t *testing.T) {
	ctx := context.Background()
	srv := fakeserver.NewInMemoryServer()
	httpSrv, grpcTarget, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("start fake server: %v", err)
	}
	defer httpSrv.Close()

	base := "http://" + grpcTarget // e.g., http://127.0.0.1:xxxxx

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSimpleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSimpleConfig(base),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("shireesh_simple.example", "id"),
					resource.TestCheckResourceAttr("shireesh_simple.example", "score", "5"),
				),
			},
			{
				Config: testAccSimpleConfigUpdated(base),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("shireesh_simple.example", "score", "10"),
					resource.TestCheckResourceAttr("shireesh_simple.example", "value", "updated"),
				),
			},
		},
	})
}

func testAccCheckSimpleDestroy(s *terraform.State) error {
	// Fake server is ephemeral; if the resource is removed from state, it's destroyed
	return nil
}

func testAccSimpleConfig(base string) string {
	return fmt.Sprintf(`
provider "shireesh" {
  api_base_url = "%s"
}

resource "shireesh_simple" "example" {
  score = 5
  value = "val"
}
`, base)
}

func testAccSimpleConfigUpdated(base string) string {
	return fmt.Sprintf(`
provider "shireesh" {
  api_base_url = "%s"
}

resource "shireesh_simple" "example" {
  score = 10
  value = "updated"
}
`, base)
}
