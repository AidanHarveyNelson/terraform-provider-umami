// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"umami": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv(envEndpoint) == "" {
		t.Fatalf("%s must be set for acceptance tests", envEndpoint)
	}
	if os.Getenv(envAPIKey) == "" && (os.Getenv(envUsername) == "" || os.Getenv(envPassword) == "") {
		t.Fatalf("%s, or %s and %s, must be set for acceptance tests", envAPIKey, envUsername, envPassword)
	}
}

const envTestMemberUserID = "UMAMI_TEST_MEMBER_USER_ID"

func testAccPreCheckMember(t *testing.T) {
	t.Helper()

	testAccPreCheck(t)
	if os.Getenv(envTestMemberUserID) == "" {
		t.Fatalf("%s must be set to the ID of a second user for team membership tests", envTestMemberUserID)
	}
}

func testAccClient(t *testing.T) *Client {
	t.Helper()

	c, err := NewClient(context.Background(), ClientConfig{
		Endpoint: os.Getenv(envEndpoint),
		APIKey:   os.Getenv(envAPIKey),
		Username: os.Getenv(envUsername),
		Password: os.Getenv(envPassword),
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
