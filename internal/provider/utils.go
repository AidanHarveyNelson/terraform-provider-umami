// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
)

var (
	// Umami validates ids with zod's z.uuid(), which requires RFC 4122 version and variant bits.
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

	// Umami's DOMAIN_REGEX without the lookahead RE2 does not support.
	domainRegex = regexp.MustCompile(`^(localhost(:[1-9]\d{0,4})?|(([a-z0-9-_]{1,63}\.))+([a-z0-9-_]{2,63}))$`)
)
