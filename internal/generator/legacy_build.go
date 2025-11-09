// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gshireesh/terraform-provider-shireesh/internal/component"
)

type ProtoMessage struct {
	Name   string
	Fields []LegacyField
}
type LegacyField struct {
	Name, Type string
	Tag        int
	Comment    string
	Repeated   bool
}

func buildProtoMessagesLegacy(spec component.Spec, tags *TagStore) []ProtoMessage {
	isWriteOnly := func(f component.Field) bool { return f.Mode == component.WriteOnlyAttributeMode }
	isReadOnly := func(f component.Field) bool { return f.Mode == component.ReadOnlyAttributeMode }
	isReadOnlyOnce := func(f component.Field) bool { return f.Mode == component.ReadOnlyOnceAttributeMode }
	isID := func(f component.Field) bool { return f.Mode == component.IDAttributeMode }
	isImmutable := func(f component.Field) bool { return f.Mode == component.ImmutableAttributeMode }
	includeInCreateInput := func(f component.Field) bool { return !isID(f) && !isReadOnly(f) && !isReadOnlyOnce(f) }
	includeInCreateOutput := func(f component.Field) bool { return !isWriteOnly(f) }
	includeInReadOutput := func(f component.Field) bool { return !isWriteOnly(f) && !isReadOnlyOnce(f) }
	includeInUpdateInput := func(f component.Field) bool {
		return !isID(f) && !isReadOnly(f) && !isReadOnlyOnce(f) && !isImmutable(f)
	}
	includeInUpdateOutput := func(f component.Field) bool { return !isWriteOnly(f) && !isReadOnlyOnce(f) }
	includeInOpenInput := includeInCreateInput
	includeInOpenOutput := includeInCreateOutput
	filterNested := func(fields []component.Field, pred func(component.Field) bool) []component.Field {
		var out []component.Field
		for _, f := range fields {
			if pred(f) {
				out = append(out, f)
			}
		}
		return out
	}
	variantNestedName := func(componentName, fieldName, variant string) string {
		return pascal(componentName) + pascal(fieldName) + variant
	}
	makeNestedVariant := func(componentName string, f component.Field, variant string, pred func(component.Field) bool) []ProtoMessage {
		filtered := filterNested(f.Fields, pred)
		var pf []LegacyField
		baseKey := componentName + "." + f.Name
		for _, nf := range filtered {
			pf = append(pf, fieldToLegacy(componentName, nf, tags.Tag(baseKey, nf.Name)))
		}
		return []ProtoMessage{{Name: variantNestedName(componentName, f.Name, variant), Fields: pf}}
	}
	componentName := spec.Name
	pascalComp := pascal(componentName)
	messages := []ProtoMessage{}
	collect := func(pred func(component.Field) bool) []LegacyField {
		var out []LegacyField
		for _, f := range spec.Fields {
			if pred(f) {
				out = append(out, fieldToLegacy(componentName, f, tags.Tag(componentName, f.Name)))
			}
		}
		return out
	}
	addVariants := func(v string, pred func(component.Field) bool) {
		for _, f := range spec.Fields {
			if f.Type == "object" && len(filterNested(f.Fields, pred)) > 0 {
				messages = append(messages, makeNestedVariant(componentName, f, v, pred)...)
			}
			if f.Type == "list" && f.ElemType == "object" && len(filterNested(f.Fields, pred)) > 0 {
				messages = append(messages, makeNestedVariant(componentName, f, v, pred)...)
			}
		}
	}
	createIn := collect(includeInCreateInput)
	createOut := collect(includeInCreateOutput)
	readOut := collect(includeInReadOutput)
	updateIn := collect(includeInUpdateInput)
	updateOut := collect(includeInUpdateOutput)
	openIn := collect(includeInOpenInput)
	openOut := collect(includeInOpenOutput)
	addVariants("CreateInput", includeInCreateInput)
	addVariants("CreateOutput", includeInCreateOutput)
	addVariants("ReadOutput", includeInReadOutput)
	addVariants("UpdateInput", includeInUpdateInput)
	addVariants("UpdateOutput", includeInUpdateOutput)
	addVariants("OpenInput", includeInOpenInput)
	addVariants("OpenOutput", includeInOpenOutput)
	adjust := func(fields []LegacyField, variant string) []LegacyField {
		byName := map[string]component.Field{}
		for _, of := range spec.Fields {
			byName[of.Name] = of
		}
		for i := range fields {
			orig := byName[fields[i].Name]
			if orig.Type == "object" || (orig.Type == "list" && orig.ElemType == "object") {
				fields[i].Type = variantNestedName(componentName, orig.Name, variant)
			}
		}
		return fields
	}
	createIn = adjust(createIn, "CreateInput")
	createOut = adjust(createOut, "CreateOutput")
	readOut = adjust(readOut, "ReadOutput")
	updateIn = adjust(updateIn, "UpdateInput")
	updateOut = adjust(updateOut, "UpdateOutput")
	openIn = adjust(openIn, "OpenInput")
	openOut = adjust(openOut, "OpenOutput")
	order := func(fs []LegacyField) []LegacyField {
		sort.Slice(fs, func(i, j int) bool { return fs[i].Tag < fs[j].Tag })
		return fs
	}
	createIn = order(createIn)
	createOut = order(createOut)
	readOut = order(readOut)
	updateIn = order(updateIn)
	updateOut = order(updateOut)
	openIn = order(openIn)
	openOut = order(openOut)
	messages = append(messages,
		ProtoMessage{Name: pascalComp + "CreateInput", Fields: createIn}, ProtoMessage{Name: pascalComp + "CreateOutput", Fields: createOut}, ProtoMessage{Name: pascalComp + "ReadOutput", Fields: readOut}, ProtoMessage{Name: pascalComp + "UpdateInput", Fields: updateIn}, ProtoMessage{Name: pascalComp + "UpdateOutput", Fields: updateOut}, ProtoMessage{Name: pascalComp + "OpenInput", Fields: openIn}, ProtoMessage{Name: pascalComp + "OpenOutput", Fields: openOut},
		ProtoMessage{Name: pascalComp + "CreateRequest", Fields: []LegacyField{{Name: "item", Type: pascalComp + "CreateInput", Tag: 1}}}, ProtoMessage{Name: pascalComp + "CreateResponse", Fields: []LegacyField{{Name: "item", Type: pascalComp + "CreateOutput", Tag: 1}}}, ProtoMessage{Name: pascalComp + "ReadRequest", Fields: []LegacyField{{Name: "id", Type: "string", Tag: 1}}}, ProtoMessage{Name: pascalComp + "ReadResponse", Fields: []LegacyField{{Name: "item", Type: pascalComp + "ReadOutput", Tag: 1}}}, ProtoMessage{Name: pascalComp + "UpdateRequest", Fields: []LegacyField{{Name: "id", Type: "string", Tag: 1}, {Name: "item", Type: pascalComp + "UpdateInput", Tag: 2}}}, ProtoMessage{Name: pascalComp + "UpdateResponse", Fields: []LegacyField{{Name: "item", Type: pascalComp + "UpdateOutput", Tag: 1}}}, ProtoMessage{Name: pascalComp + "DeleteRequest", Fields: []LegacyField{{Name: "id", Type: "string", Tag: 1}}}, ProtoMessage{Name: pascalComp + "DeleteResponse", Fields: []LegacyField{{Name: "success", Type: "bool", Tag: 1}}}, ProtoMessage{Name: pascalComp + "OpenRequest", Fields: []LegacyField{{Name: "item", Type: pascalComp + "OpenInput", Tag: 1}}}, ProtoMessage{Name: pascalComp + "OpenResponse", Fields: []LegacyField{{Name: "item", Type: pascalComp + "OpenOutput", Tag: 1}}},
	)
	if spec.HasRole("datasource") {
		messages = append(messages, ProtoMessage{Name: pascalComp + "DataSourceReadRequest", Fields: []LegacyField{{Name: "id", Type: "string", Tag: 1}}}, ProtoMessage{Name: pascalComp + "DataSourceReadResponse", Fields: []LegacyField{{Name: "item", Type: pascalComp + "ReadOutput", Tag: 1}}})
	}
	return messages
}

func fieldToLegacy(componentName string, f component.Field, tag int) LegacyField {
	comment := strings.TrimSpace(f.Description)
	if f.Mode != "" {
		if comment != "" {
			comment += " "
		}
		comment += fmt.Sprintf("[mode:%s]", f.Mode)
	}
	lf := LegacyField{Name: f.Name, Tag: tag, Comment: comment}
	switch f.Type {
	case "string":
		lf.Type = "string"
	case "number":
		lf.Type = "double"
	case "bool":
		lf.Type = "bool"
	case "int64":
		lf.Type = "int64"
	case "int32":
		lf.Type = "int32"
	case "object":
		lf.Type = pascal(componentName) + pascal(f.Name)
	case "list":
		switch f.ElemType {
		case "string":
			lf.Type = "string"
		case "number":
			lf.Type = "double"
		case "bool":
			lf.Type = "bool"
		case "int64":
			lf.Type = "int64"
		case "int32":
			lf.Type = "int32"
		case "object":
			lf.Type = pascal(componentName) + pascal(f.Name)
		default:
			lf.Type = "string"
		}
		lf.Repeated = true
	default:
		lf.Type = "string"
	}
	return lf
}
