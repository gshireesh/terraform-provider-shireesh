// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package component

// Spec represents a component definition used to generate provider code and proto.
// It is intentionally kept small and additive-only to preserve backward compatibility.
type Spec struct {
	Name    string   `yaml:"name"`
	Roles   []string `yaml:"roles"` // values: resource, datasource, ephemeral
	Fields  []Field  `yaml:"fields"`
	Gateway bool     `yaml:"gateway"` // expose via grpc-gateway HTTP annotations
	Section string   `yaml:"section"` // logical grouping for docs/navigation
}

// AttributeMode defines semantic behavior of a field across operations.
// KeyAttributeMode: attributes identifying resource instances (required input, not computed).
// IDAttributeMode: "id" style attributes returned after create/read/update (computed).
// ReadWriteAttributeMode: default read/write behavior (respect Optional/Computed flags if set).
// ImmutableAttributeMode: write-once at create; subsequent changes should force replacement.
// ReadOnlyAttributeMode: returned by every operation; always Computed.
// ReadOnlyOnceAttributeMode: returned only by create; modeled as Computed.
// WriteOnlyAttributeMode: sent on create/update; never returned; modeled as Optional/Required without Computed.
// Empty value means default (ReadWriteAttributeMode behavior).
type AttributeMode string

const (
	KeyAttributeMode          AttributeMode = "key"
	IDAttributeMode           AttributeMode = "id"
	ReadWriteAttributeMode    AttributeMode = "read_write"
	ImmutableAttributeMode    AttributeMode = "immutable"
	ReadOnlyAttributeMode     AttributeMode = "read_only"
	ReadOnlyOnceAttributeMode AttributeMode = "read_only_once"
	WriteOnlyAttributeMode    AttributeMode = "write_only"
)

type Field struct {
	Name            string        `yaml:"name"`
	Type            string        `yaml:"type"` // string, number, bool, object, list, map, dynamic
	Description     string        `yaml:"description"`
	Optional        bool          `yaml:"optional"`
	Computed        bool          `yaml:"computed"`
	Default         string        `yaml:"default"`
	Fields          []Field       `yaml:"fields"`           // for object
	ElemType        string        `yaml:"elem_type"`        // for list/map
	Mode            AttributeMode `yaml:"mode"`             // semantic mode (optional)
	RequiresReplace bool          `yaml:"requires_replace"` // when true, changes force resource replacement
}

// Helpers.
func (s Spec) HasRole(role string) bool {
	for _, r := range s.Roles {
		if r == role {
			return true
		}
	}
	return false
}
