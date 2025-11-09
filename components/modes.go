// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package components

import "github.com/gshireesh/terraform-provider-shireesh/internal/component"

// ModesSpec exercises various integer sizes and attribute modes including nested objects and lists.
// It is intended purely for generation/testing purposes.
func ModesSpec() component.Spec {
	return component.Spec{
		Name:    "modes",
		Roles:   []string{"resource", "datasource", "ephemeral"},
		Gateway: true,
		Section: "experiments",
		Fields: []component.Field{
			// Top-level identity and key fields
			{Name: "project", Type: "string", Description: "Project identifier (key)", Mode: component.KeyAttributeMode},
			{Name: "id", Type: "string", Description: "Server assigned ID", Mode: component.IDAttributeMode},

			// Primitive integers and number/bool
			{Name: "count32", Type: "int32", Description: "32-bit counter", Optional: true, RequiresReplace: true},
			{Name: "count64", Type: "int64", Description: "64-bit counter", Optional: true},
			{Name: "ratio", Type: "number", Description: "Floating ratio value", Optional: true, RequiresReplace: true},
			{Name: "enabled", Type: "bool", Description: "Feature enabled flag", Optional: true},

			// Lists of integers
			{Name: "flags_int32", Type: "list", ElemType: "int32", Description: "List of int32 flags", Optional: true},
			{Name: "counters_int64", Type: "list", ElemType: "int64", Description: "List of int64 counters", Optional: true},

			// Dynamic content (write-only secret payload)
			{Name: "secret_payload", Type: "dynamic", Description: "Opaque write-only secret payload", Mode: component.WriteOnlyAttributeMode, Optional: true},

			// Nested configuration object demonstrating modes within nested structure
			{Name: "config", Type: "object", Description: "Configuration block with various modes", Optional: true, RequiresReplace: true, Fields: []component.Field{
				{Name: "version", Type: "string", Description: "Immutable version string", Mode: component.ImmutableAttributeMode, Optional: true},
				{Name: "secret", Type: "string", Description: "Write-only nested secret", Mode: component.WriteOnlyAttributeMode, Optional: true},
				{Name: "result", Type: "string", Description: "Read-only computed result", Mode: component.ReadOnlyAttributeMode},
				{Name: "initial_token", Type: "string", Description: "Token only returned at create", Mode: component.ReadOnlyOnceAttributeMode},
				{Name: "note", Type: "string", Description: "Editable note (default read/write)", Optional: true},
			}},

			// List of nested objects including int32/int64 fields
			{Name: "metrics", Type: "list", ElemType: "object", Description: "List of metrics objects", Optional: true, Fields: []component.Field{
				{Name: "name", Type: "string", Description: "Metric name (key)", Mode: component.KeyAttributeMode},
				{Name: "value_int32", Type: "int32", Description: "Sample int32 metric"},
				{Name: "value_int64", Type: "int64", Description: "Sample int64 metric"},
				{Name: "computed_hash", Type: "string", Description: "Computed hash of metric", Mode: component.ReadOnlyAttributeMode},
			}},

			// Another nested object to exercise read_only_once and write_only together
			{Name: "provisioning", Type: "object", Description: "Provisioning data", Optional: true, Fields: []component.Field{
				{Name: "request_id", Type: "string", Description: "Client supplied request ID (key)", Mode: component.KeyAttributeMode},
				{Name: "api_key", Type: "string", Description: "Write-only API key", Mode: component.WriteOnlyAttributeMode, Optional: true},
				{Name: "provisioned_id", Type: "string", Description: "ID after provisioning (read-only once)", Mode: component.ReadOnlyOnceAttributeMode},
			}},
		},
	}
}
