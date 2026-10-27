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

const (
	envTestAdminUsername = "UMAMI_TEST_ADMIN_USERNAME"
	envTestAdminPassword = "UMAMI_TEST_ADMIN_PASSWORD"
)

// testAccPreCheckAdmin switches the provider to admin login, which managing users needs.
func testAccPreCheckAdmin(t *testing.T) {
	t.Helper()

	testAccPreCheck(t)
	username, password := os.Getenv(envTestAdminUsername), os.Getenv(envTestAdminPassword)
	if username == "" || password == "" {
		t.Fatalf("%s and %s must be set to an admin login for user tests", envTestAdminUsername, envTestAdminPassword)
	}
	t.Setenv(envAPIKey, "")
	t.Setenv(envUsername, username)
	t.Setenv(envPassword, password)
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
