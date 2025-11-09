// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package components

import "github.com/hashicorp/terraform-provider-scaffolding-framework/internal/component"

func ComplexSpec() component.Spec {
	return component.Spec{
		Name:    "complex",
		Roles:   []string{"resource", "datasource", "ephemeral"},
		Gateway: true,
		Section: "core",
		Fields: []component.Field{
			{Name: "name", Type: "string", Description: "Component name", Optional: true},
			{Name: "id", Type: "string", Description: "Unique identifier", Mode: component.IDAttributeMode},
			{Name: "settings", Type: "object", Description: "Settings object with mixed types", Optional: true, Fields: []component.Field{
				{Name: "retries", Type: "number", Description: "Number of retries", Optional: true},
				{Name: "enabled", Type: "bool", Description: "Whether enabled", Optional: true},
				{Name: "labels", Type: "list", ElemType: "string", Description: "List of labels", Optional: true},
			}},
			{Name: "groups", Type: "list", Description: "List of group objects (list of objects each having list members)", Optional: true, ElemType: "object", Fields: []component.Field{
				{Name: "group_name", Type: "string", Description: "Group name"},
				{Name: "members", Type: "list", ElemType: "string", Description: "Members in the group", Optional: true},
			}},
			{Name: "tags", Type: "list", ElemType: "string", Description: "Tags applied", Optional: true, Computed: true},
		},
	}
}
