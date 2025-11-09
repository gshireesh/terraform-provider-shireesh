// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package components

import "github.com/gshireesh/terraform-provider-shireesh/internal/component"

func SimpleSpec() component.Spec {
	return component.Spec{
		Name:    "simple",
		Roles:   []string{"resource"},
		Gateway: true,
		Section: "core",
		Fields: []component.Field{
			{Name: "id", Type: "string", Description: "The ID of this resource.", Mode: component.IDAttributeMode},
			{Name: "value", Type: "string", Description: "Simple value", Optional: true},
			{Name: "flags", Type: "list", ElemType: "bool", Description: "Boolean flags", Optional: true},
			{Name: "counts", Type: "list", ElemType: "number", Description: "Numeric counts", Optional: true},
			{Name: "score", Type: "int64", Description: "Numeric counts", Optional: false},
		},
	}
}
