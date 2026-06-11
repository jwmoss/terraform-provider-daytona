// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strings"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func terraformTimeString(value time.Time) types.String {
	if value.IsZero() {
		return types.StringNull()
	}
	return types.StringValue(value.Format(time.RFC3339))
}

func organizationRoleIDs(roles []apiclient.OrganizationRole) []string {
	ids := make([]string, 0, len(roles))
	for _, role := range roles {
		ids = append(ids, role.Id)
	}
	return ids
}

func parseCompositeImportID(importID, firstName, secondName string) (string, string, error) {
	parts := strings.Split(importID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected import ID in the form %s/%s", firstName, secondName)
	}
	return parts[0], parts[1], nil
}
