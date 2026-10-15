// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	// Umami validates ids with zod's z.uuid(), which requires RFC 4122 version and variant bits.
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

	// Umami's DOMAIN_REGEX without the lookahead RE2 does not support.
	domainRegex = regexp.MustCompile(`^(localhost(:[1-9]\d{0,4})?|(([a-z0-9-_]{1,63}\.))+([a-z0-9-_]{2,63}))$`)
)

func isNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func clientFromProviderData(providerData any, diags *diag.Diagnostics, kind string) *Client {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(*Client)
	if !ok {
		diags.AddError(
			fmt.Sprintf("Unexpected %s Configure Type", kind),
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}
	return client
}

func configureResource(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *Client {
	return clientFromProviderData(req.ProviderData, &resp.Diagnostics, "Resource")
}
