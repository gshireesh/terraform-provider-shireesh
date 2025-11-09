// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package generator

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/gshireesh/terraform-provider-shireesh/internal/component"
)

// GenerateAll now supports optional HTTP (grpc-gateway) annotations in combined proto.
func GenerateAll(specs []component.Spec, outDir string, tagFile string, includeHTTP bool, serviceName string, protoPackage string, goPackagePrefix string) error {
	// Tag tracking with numeric IDs. .tags format supports:
	// message:field1=1,field2=2  (preferred)
	// message:field1,field2      (legacy; assigns 1..N in listed order)
	ts := newTagStore()
	if b, err := os.ReadFile(tagFile); err == nil {
		ts.load(string(b))
	}

	// Collect all message names before ordering (top-level + nested) so we can preserve ordering.
	// We'll build a flat list of message specs to tag.
	for _, spec := range specs {
		// Ensure tags exist for current fields; assign if missing. Leave removed fields reserved.
		for _, f := range spec.Fields {
			_ = ts.ensure(spec.Name, f.Name)
			if f.Type == "object" {
				nestedKey := nestedMessageName(spec.Name, f.Name)
				for _, nf := range f.Fields {
					_ = ts.ensure(nestedKey, nf.Name)
				}
			}
		}

		// Also preserve nested ordering/assignment for object children in spec
		for _, f := range spec.Fields {
			if f.Type == "object" {
				// Update back in spec not needed for tag assignment; we keep original spec order
			}
		}
	}

	// Persist tags
	var tagBuf bytes.Buffer
	tagBuf.WriteString(ts.String())
	if err := os.WriteFile(tagFile, tagBuf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write tags: %w", err)
	}

	// Generate role & model files
	for _, spec := range specs {
		if err := generateRoleFiles(spec, outDir); err != nil {
			return err
		}
	}

	// Remove legacy per-component proto files if present
	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".proto" && e.Name() != "components.proto" {
			_ = os.Remove(filepath.Join(outDir, e.Name()))
		}
	}

	if err := generateCombinedProto(specs, outDir, includeHTTP, serviceName, protoPackage, goPackagePrefix, ts); err != nil {
		return err
	}
	if err := generateRegistry(specs, outDir); err != nil {
		return err
	}
	if err := generateDefaults(outDir); err != nil {
		return err
	}
	return nil
}

// tagStore keeps stable numeric tags per message.
type tagStore struct {
	tags map[string]map[string]int // message -> field -> tag
}

func newTagStore() *tagStore { return &tagStore{tags: map[string]map[string]int{}} }

func (s *tagStore) load(content string) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		msg := strings.TrimSpace(parts[0])
		fields := strings.Split(parts[1], ",")
		if _, ok := s.tags[msg]; !ok {
			s.tags[msg] = map[string]int{}
		}
		next := s.max(msg) + 1
		for _, f := range fields {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			name := f
			tag := 0
			if eq := strings.Index(f, "="); eq >= 0 {
				name = strings.TrimSpace(f[:eq])
				if v := strings.TrimSpace(f[eq+1:]); v != "" {
					if n, err := atoiSafe(v); err == nil && n > 0 {
						tag = n
					}
				}
			}
			if tag == 0 {
				// legacy: assign next sequential but do not override if already present
				if _, exists := s.tags[msg][name]; !exists {
					s.tags[msg][name] = next
					next++
				}
			} else {
				s.tags[msg][name] = tag
				if tag >= next {
					next = tag + 1
				}
			}
		}
	}
}

func atoiSafe(s string) (int, error) {
	var n int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("nan")
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}

func (s *tagStore) max(msg string) int {
	m := 0
	for _, v := range s.tags[msg] {
		if v > m {
			m = v
		}
	}
	return m
}

func (s *tagStore) ensure(msg, field string) int {
	if _, ok := s.tags[msg]; !ok {
		s.tags[msg] = map[string]int{}
	}
	if t, ok := s.tags[msg][field]; ok {
		return t
	}
	t := s.max(msg) + 1
	s.tags[msg][field] = t
	return t
}

func (s *tagStore) tag(msg, field string) int {
	if m, ok := s.tags[msg]; ok {
		if t, ok := m[field]; ok {
			return t
		}
	}
	return s.ensure(msg, field)
}

func (s *tagStore) String() string {
	// Deterministic ordering by message then tag
	var msgs []string
	for k := range s.tags {
		msgs = append(msgs, k)
	}
	sort.Strings(msgs)
	var b strings.Builder
	for _, msg := range msgs {
		// sort fields by tag
		type ft struct {
			name string
			tag  int
		}
		var arr []ft
		for name, tag := range s.tags[msg] {
			arr = append(arr, ft{name, tag})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].tag < arr[j].tag })
		b.WriteString(msg)
		b.WriteString(":")
		for i, f := range arr {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(f.name)
			b.WriteString("=")
			b.WriteString(fmt.Sprintf("%d", f.tag))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Generate role & model files
func generateRoleFiles(spec component.Spec, outDir string) error {
	// Generate shared model first
	if err := generateModelFile(spec, outDir); err != nil {
		return err
	}
	roles := []string{"resource", "datasource", "ephemeral"}
	for _, role := range roles {
		if !spec.HasRole(role) {
			continue
		}
		file := filepath.Join(outDir, fmt.Sprintf("%s_%s.gen.go", spec.Name, role))
		var buf bytes.Buffer
		data := struct {
			Spec component.Spec
			Role string
		}{Spec: spec, Role: role}
		// role template now without model
		roleTpl := template.Must(template.New("role_go").Funcs(funcMap()).Parse(goRoleTemplateV2))
		if err := roleTpl.Execute(&buf, data); err != nil {
			return err
		}
		if err := os.WriteFile(file, buf.Bytes(), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func generateModelFile(spec component.Spec, outDir string) error {
	file := filepath.Join(outDir, fmt.Sprintf("%s_model.gen.go", spec.Name))
	var buf bytes.Buffer
	data := spec
	t := template.Must(template.New("model").Funcs(funcMap()).Parse(modelTemplate))
	if err := t.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(file, buf.Bytes(), 0o644)
}

func generateCombinedProto(specs []component.Spec, outDir string, includeHTTP bool, serviceName string, protoPackage string, goPackagePrefix string, ts *tagStore) error { // includeHTTP retained for global override
	file := filepath.Join(outDir, "components.proto")
	var buf bytes.Buffer
	var all []CombinedProtoSpec
	anyGateway := false
	for _, s := range specs {
		msgs := buildProtoMessages(s, ts)
		all = append(all, CombinedProtoSpec{Spec: s, Messages: msgs, HasResource: s.HasRole("resource"), HasDataSource: s.HasRole("datasource"), HasEphemeral: s.HasRole("ephemeral")})
		if s.Gateway {
			anyGateway = true
		}
	}
	t := template.Must(template.New("combined_proto").Funcs(funcMap()).Parse(combinedProtoTemplate))
	data := struct {
		Specs                       []CombinedProtoSpec
		HTTP, HTTPImport            bool
		Service, Package, GoPackage string
	}{Specs: all, HTTP: includeHTTP, HTTPImport: includeHTTP && anyGateway, Service: serviceName, Package: protoPackage, GoPackage: goPackagePrefix}
	if err := t.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(file, buf.Bytes(), 0o644)
}

func generateDefaults(outDir string) error {
	content := `// Code generated by componentgen; DO NOT EDIT.
package generated

import (
	"net/http"
	"strings"
)

type APIClient struct {
	BaseURL  string
	HTTP    *http.Client
	Insecure bool
	Bearer   string
}

// GRPCTarget returns a host:port derived from BaseURL by stripping scheme and path.
func (c *APIClient) GRPCTarget() string {
	base := c.BaseURL
	if strings.HasPrefix(base, "http://") {
		base = strings.TrimPrefix(base, "http://")
	} else if strings.HasPrefix(base, "https://") {
		base = strings.TrimPrefix(base, "https://")
	}
	if i := strings.Index(base, "/"); i >= 0 { base = base[:i] }
	return base
}

 type Defaults struct {
	ServiceName string
	APIBaseURL  string
	Insecure    bool
	Environment string // dev, stage, prod
}

var DefaultConfig = Defaults{
	ServiceName: "GrpcTerraformService",
	APIBaseURL:  "http://localhost:8080",
	Insecure:    true,
	Environment: "dev",
}
`
	return os.WriteFile(filepath.Join(outDir, "defaults.gen.go"), []byte(content), 0o644)
}

// CombinedProtoSpec holds spec + derived messages for template.
type CombinedProtoSpec struct {
	Spec                                     component.Spec
	Messages                                 []ProtoMessage
	HasResource, HasDataSource, HasEphemeral bool
}

// Add HTTP annotation helpers
func httpCreatePath(name string) string { return fmt.Sprintf("/v1/%s", name) }
func httpReadPath(name string) string   { return fmt.Sprintf("/v1/%s/{id}", name) }
func httpUpdatePath(name string) string { return fmt.Sprintf("/v1/%s/{id}", name) }
func httpDeletePath(name string) string { return fmt.Sprintf("/v1/%s/{id}", name) }
func httpDsReadPath(name string) string { return fmt.Sprintf("/v1/datasource/%s/{id}", name) }
func httpOpenPath(name string) string   { return fmt.Sprintf("/v1/%s/open", name) }

// New helpers for richer proto
type ProtoField struct {
	Name     string
	Type     string
	Tag      int
	Comment  string
	Repeated bool
}
type ProtoMessage struct {
	Name   string
	Fields []ProtoField
}

func nestedMessageName(componentName, fieldName string) string {
	return componentName + "." + fieldName
}

// fieldToProto translates a component.Field to a ProtoField with provided numeric tag.
func fieldToProto(componentName string, f component.Field, tag int) ProtoField {
	comment := strings.TrimSpace(f.Description)
	if f.Mode != "" {
		if comment != "" {
			comment += " "
		}
		comment += fmt.Sprintf("[mode:%s]", string(f.Mode))
	}
	pf := ProtoField{Name: f.Name, Tag: tag, Comment: comment}
	switch f.Type {
	case "string":
		pf.Type = "string"
	case "number":
		pf.Type = "double"
	case "bool":
		pf.Type = "bool"
	case "int64":
		pf.Type = "int64"
	case "int32":
		pf.Type = "int32"
	case "object":
		pf.Type = pascal(componentName) + pascal(f.Name)
	case "list":
		switch f.ElemType {
		case "string":
			pf.Type = "string"
		case "number":
			pf.Type = "double"
		case "bool":
			pf.Type = "bool"
		case "int64":
			pf.Type = "int64"
		case "int32":
			pf.Type = "int32"
		case "object":
			pf.Type = pascal(componentName) + pascal(f.Name)
		default:
			pf.Type = "string"
		}
		pf.Repeated = true
	default:
		pf.Type = "string"
	}
	return pf
}

func buildProtoMessages(spec component.Spec, tags *tagStore) []ProtoMessage {
	// Variant filtering helpers
	isWriteOnly := func(f component.Field) bool { return f.Mode == component.WriteOnlyAttributeMode }
	isReadOnly := func(f component.Field) bool { return f.Mode == component.ReadOnlyAttributeMode }
	isReadOnlyOnce := func(f component.Field) bool { return f.Mode == component.ReadOnlyOnceAttributeMode }
	isID := func(f component.Field) bool { return f.Mode == component.IDAttributeMode }
	isImmutable := func(f component.Field) bool { return f.Mode == component.ImmutableAttributeMode }

	// Inclusion logic per variant (top-level fields inside Item structs)
	includeInCreateInput := func(f component.Field) bool { return !isID(f) && !isReadOnly(f) && !isReadOnlyOnce(f) }
	includeInCreateOutput := func(f component.Field) bool { return !isWriteOnly(f) }
	includeInReadOutput := func(f component.Field) bool { return !isWriteOnly(f) && !isReadOnlyOnce(f) }
	includeInUpdateInput := func(f component.Field) bool {
		return !isID(f) && !isReadOnly(f) && !isReadOnlyOnce(f) && !isImmutable(f)
	}
	includeInUpdateOutput := func(f component.Field) bool { return !isWriteOnly(f) && !isReadOnlyOnce(f) }
	includeInOpenInput := includeInCreateInput
	includeInOpenOutput := includeInCreateOutput

	// Helper to filter nested fields with same predicate
	filterNested := func(fields []component.Field, pred func(component.Field) bool) []component.Field {
		var out []component.Field
		for _, f := range fields {
			if pred(f) {
				out = append(out, f)
			}
		}
		return out
	}

	// Build variant-specific nested message name
	variantNestedName := func(componentName, fieldName, variant string) string {
		return pascal(componentName) + pascal(fieldName) + variant
	}

	makeNestedVariantMessages := func(componentName string, f component.Field, variant string, pred func(component.Field) bool) []ProtoMessage {
		// For object/list-of-object produce a message with filtered inner fields
		filtered := filterNested(f.Fields, pred)
		var pf []ProtoField
		baseKey := nestedMessageName(componentName, f.Name)
		for _, nf := range filtered {
			pf = append(pf, fieldToProto(componentName, nf, tags.tag(baseKey, nf.Name)))
		}
		name := variantNestedName(componentName, f.Name, variant)
		return []ProtoMessage{{Name: name, Fields: pf}}
	}

	componentName := spec.Name
	pascalComp := pascal(componentName)
	messages := []ProtoMessage{}

	// Collect top-level variant field sets
	collectVariantFields := func(pred func(component.Field) bool) []ProtoField {
		var out []ProtoField
		for _, f := range spec.Fields {
			if pred(f) {
				out = append(out, fieldToProto(componentName, f, tags.tag(componentName, f.Name)))
			}
		}
		return out
	}

	// Nested variant messages for each variant (needed when referenced)
	addNestedVariants := func(variant string, pred func(component.Field) bool) {
		for _, f := range spec.Fields {
			if f.Type == "object" && len(filterNested(f.Fields, pred)) > 0 {
				messages = append(messages, makeNestedVariantMessages(componentName, f, variant, pred)...)
			}
			if f.Type == "list" && f.ElemType == "object" && len(filterNested(f.Fields, pred)) > 0 {
				messages = append(messages, makeNestedVariantMessages(componentName, f, variant, pred)...)
			}
		}
	}

	// Variants
	createInputFields := collectVariantFields(includeInCreateInput)
	createOutputFields := collectVariantFields(includeInCreateOutput)
	readOutputFields := collectVariantFields(includeInReadOutput)
	updateInputFields := collectVariantFields(includeInUpdateInput)
	updateOutputFields := collectVariantFields(includeInUpdateOutput)
	openInputFields := collectVariantFields(includeInOpenInput)
	openOutputFields := collectVariantFields(includeInOpenOutput)

	addNestedVariants("CreateInput", includeInCreateInput)
	addNestedVariants("CreateOutput", includeInCreateOutput)
	addNestedVariants("ReadOutput", includeInReadOutput)
	addNestedVariants("UpdateInput", includeInUpdateInput)
	addNestedVariants("UpdateOutput", includeInUpdateOutput)
	addNestedVariants("OpenInput", includeInOpenInput)
	addNestedVariants("OpenOutput", includeInOpenOutput)

	// Helper to reference nested variant type names in top-level messages (adjust field types)
	adjustVariantFieldTypes := func(fields []ProtoField, variant string) []ProtoField {
		// Build a name->Field map once
		origByName := map[string]component.Field{}
		for _, of := range spec.Fields {
			origByName[of.Name] = of
		}
		for i := range fields {
			orig, ok := origByName[fields[i].Name]
			if !ok {
				continue
			}
			if orig.Type == "object" {
				fields[i].Type = variantNestedName(componentName, orig.Name, variant)
			} else if orig.Type == "list" && orig.ElemType == "object" {
				fields[i].Type = variantNestedName(componentName, orig.Name, variant)
			}
		}
		return fields
	}

	createInputFields = adjustVariantFieldTypes(createInputFields, "CreateInput")
	createOutputFields = adjustVariantFieldTypes(createOutputFields, "CreateOutput")
	readOutputFields = adjustVariantFieldTypes(readOutputFields, "ReadOutput")
	updateInputFields = adjustVariantFieldTypes(updateInputFields, "UpdateInput")
	updateOutputFields = adjustVariantFieldTypes(updateOutputFields, "UpdateOutput")
	openInputFields = adjustVariantFieldTypes(openInputFields, "OpenInput")
	openOutputFields = adjustVariantFieldTypes(openOutputFields, "OpenOutput")

	// Top-level item wrapper messages (fields sorted by tag for readability)
	sortFieldsByTag := func(fs []ProtoField) []ProtoField {
		sort.Slice(fs, func(i, j int) bool { return fs[i].Tag < fs[j].Tag })
		return fs
	}
	createInputFields = sortFieldsByTag(createInputFields)
	createOutputFields = sortFieldsByTag(createOutputFields)
	readOutputFields = sortFieldsByTag(readOutputFields)
	updateInputFields = sortFieldsByTag(updateInputFields)
	updateOutputFields = sortFieldsByTag(updateOutputFields)
	openInputFields = sortFieldsByTag(openInputFields)
	openOutputFields = sortFieldsByTag(openOutputFields)
	messages = append(messages,
		ProtoMessage{Name: pascalComp + "CreateInput", Fields: createInputFields},
		ProtoMessage{Name: pascalComp + "CreateOutput", Fields: createOutputFields},
		ProtoMessage{Name: pascalComp + "ReadOutput", Fields: readOutputFields},
		ProtoMessage{Name: pascalComp + "UpdateInput", Fields: updateInputFields},
		ProtoMessage{Name: pascalComp + "UpdateOutput", Fields: updateOutputFields},
		ProtoMessage{Name: pascalComp + "OpenInput", Fields: openInputFields},
		ProtoMessage{Name: pascalComp + "OpenOutput", Fields: openOutputFields},
	)

	// Request/Response envelopes referencing variant messages
	messages = append(messages,
		ProtoMessage{Name: pascalComp + "CreateRequest", Fields: []ProtoField{{Name: "item", Type: pascalComp + "CreateInput", Tag: 1}}},
		ProtoMessage{Name: pascalComp + "CreateResponse", Fields: []ProtoField{{Name: "item", Type: pascalComp + "CreateOutput", Tag: 1}}},
		ProtoMessage{Name: pascalComp + "ReadRequest", Fields: []ProtoField{{Name: "id", Type: "string", Tag: 1}}},
		ProtoMessage{Name: pascalComp + "ReadResponse", Fields: []ProtoField{{Name: "item", Type: pascalComp + "ReadOutput", Tag: 1}}},
		ProtoMessage{Name: pascalComp + "UpdateRequest", Fields: []ProtoField{{Name: "id", Type: "string", Tag: 1}, {Name: "item", Type: pascalComp + "UpdateInput", Tag: 2}}},
		ProtoMessage{Name: pascalComp + "UpdateResponse", Fields: []ProtoField{{Name: "item", Type: pascalComp + "UpdateOutput", Tag: 1}}},
		ProtoMessage{Name: pascalComp + "DeleteRequest", Fields: []ProtoField{{Name: "id", Type: "string", Tag: 1}}},
		ProtoMessage{Name: pascalComp + "DeleteResponse", Fields: []ProtoField{{Name: "success", Type: "bool", Tag: 1}}},
		ProtoMessage{Name: pascalComp + "OpenRequest", Fields: []ProtoField{{Name: "item", Type: pascalComp + "OpenInput", Tag: 1}}},
		ProtoMessage{Name: pascalComp + "OpenResponse", Fields: []ProtoField{{Name: "item", Type: pascalComp + "OpenOutput", Tag: 1}}},
	)
	if spec.HasRole("datasource") {
		messages = append(messages,
			ProtoMessage{Name: pascalComp + "DataSourceReadRequest", Fields: []ProtoField{{Name: "id", Type: "string", Tag: 1}}},
			ProtoMessage{Name: pascalComp + "DataSourceReadResponse", Fields: []ProtoField{{Name: "item", Type: pascalComp + "ReadOutput", Tag: 1}}},
		)
	}

	return messages
}

func generateRegistry(specs []component.Spec, outDir string) error {
	file := filepath.Join(outDir, "registry.gen.go")
	var buf bytes.Buffer
	// ensure deterministic ordering
	sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	t := template.Must(template.New("registry").Funcs(funcMap()).Parse(registryTemplate))
	if err := t.Execute(&buf, specs); err != nil {
		return err
	}
	return os.WriteFile(file, buf.Bytes(), 0o644)
}

// Role-specific Go template (kept concise)
const goRoleTemplateV2 = `// Code generated by componentgen; DO NOT EDIT.
 package generated
 
 import (
 	"context"
 	{{- if eq .Role "resource" }}
 	// grpc client imports
 	"github.com/hashicorp/terraform-plugin-framework/resource"
 	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
 	{{- if HasDefault .Spec }}
 	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"{{ end }}
 	{{- if or (HasReplaceString .Spec) (HasReplaceFloat64 .Spec) (HasReplaceBool .Spec) (HasReplaceInt64 .Spec) }}
 	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
 	{{- end }}
 	{{- if HasReplaceString .Spec }}
 	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
 	{{- end }}
 	{{- if HasReplaceInt64 .Spec }}
 	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
 	{{- end }}
 	{{- if HasReplaceFloat64 .Spec }}
 	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
 	{{- end }}
 	{{- if HasReplaceBool .Spec }}
 	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
 	{{- end }}
 	"google.golang.org/grpc"
 	"google.golang.org/grpc/credentials/insecure"
 	"google.golang.org/grpc/metadata"
 	{{- end }}
 	{{- if eq .Role "datasource" }}
 	"github.com/hashicorp/terraform-plugin-framework/datasource"
 	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
 	"google.golang.org/grpc"
 	"google.golang.org/grpc/credentials/insecure"
 	"google.golang.org/grpc/metadata"
 	{{- end }}
 	{{- if eq .Role "ephemeral" }}
 	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
 	ephschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
 	"google.golang.org/grpc"
 	"google.golang.org/grpc/credentials/insecure"
 	"google.golang.org/grpc/metadata"
 	{{- end }}
 	"github.com/hashicorp/terraform-plugin-framework/types"
 )

{{- if eq .Role "resource" }}
func New{{ .Spec.Name | Pascal }}Resource() resource.Resource { return &{{ .Spec.Name | Pascal }}Resource{} }

type {{ .Spec.Name | Pascal }}Resource struct{ client *APIClient }

func (r *{{ .Spec.Name | Pascal }}Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) { resp.TypeName = req.ProviderTypeName + "_{{ .Spec.Name }}" }
func (r *{{ .Spec.Name | Pascal }}Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := map[string]schema.Attribute{}
	{{- range .Spec.Fields }}
	attrs["{{ .Name }}"] = {{ ResourceAttr . }}
	{{- end }}
	resp.Schema = schema.Schema{ Attributes: attrs }
}
func (r *{{ .Spec.Name | Pascal }}Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	c, ok := req.ProviderData.(*APIClient)
	if ok { r.client = c }
}
func (r *{{ .Spec.Name | Pascal }}Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil { return }
	conn, err := grpc.Dial(r.client.GRPCTarget(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { return }
	defer conn.Close()
	cli := NewGrpcTerraformServiceClient(conn)
	if r.client.Bearer != "" { ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+r.client.Bearer) }
	_, _ = cli.{{ .Spec.Name | Pascal }}Create(ctx, &{{ .Spec.Name | Pascal }}CreateRequest{})
}
func (r *{{ .Spec.Name | Pascal }}Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil { return }
	conn, err := grpc.Dial(r.client.GRPCTarget(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { return }
	defer conn.Close()
	cli := NewGrpcTerraformServiceClient(conn)
	if r.client.Bearer != "" { ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+r.client.Bearer) }
	_, _ = cli.{{ .Spec.Name | Pascal }}Read(ctx, &{{ .Spec.Name | Pascal }}ReadRequest{})
}
func (r *{{ .Spec.Name | Pascal }}Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil { return }
	conn, err := grpc.Dial(r.client.GRPCTarget(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { return }
	defer conn.Close()
	cli := NewGrpcTerraformServiceClient(conn)
	if r.client.Bearer != "" { ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+r.client.Bearer) }
	_, _ = cli.{{ .Spec.Name | Pascal }}Update(ctx, &{{ .Spec.Name | Pascal }}UpdateRequest{})
}
func (r *{{ .Spec.Name | Pascal }}Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil { return }
	conn, err := grpc.Dial(r.client.GRPCTarget(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { return }
	defer conn.Close()
	cli := NewGrpcTerraformServiceClient(conn)
	if r.client.Bearer != "" { ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+r.client.Bearer) }
	_, _ = cli.{{ .Spec.Name | Pascal }}Delete(ctx, &{{ .Spec.Name | Pascal }}DeleteRequest{})
}
{{- end }}

{{- if eq .Role "datasource" }}
func New{{ .Spec.Name | Pascal }}DataSource() datasource.DataSource { return &{{ .Spec.Name | Pascal }}DataSource{} }

type {{ .Spec.Name | Pascal }}DataSource struct{ client *APIClient }

func (d *{{ .Spec.Name | Pascal }}DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) { resp.TypeName = req.ProviderTypeName + "_{{ .Spec.Name }}" }
func (d *{{ .Spec.Name | Pascal }}DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := map[string]dsschema.Attribute{}
	{{- range .Spec.Fields }}
	attrs["{{ .Name }}"] = {{ DSAttr . }}
	{{- end }}
	resp.Schema = dsschema.Schema{ Attributes: attrs }
}
func (d *{{ .Spec.Name | Pascal }}DataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	c, ok := req.ProviderData.(*APIClient)
	if ok { d.client = c }
}
func (d *{{ .Spec.Name | Pascal }}DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil { return }
	conn, err := grpc.Dial(d.client.GRPCTarget(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { return }
	defer conn.Close()
	cli := NewGrpcTerraformServiceClient(conn)
	if d.client.Bearer != "" { ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+d.client.Bearer) }
	_, _ = cli.{{ .Spec.Name | Pascal }}DataSourceRead(ctx, &{{ .Spec.Name | Pascal }}DataSourceReadRequest{})
}
{{- end }}

{{- if eq .Role "ephemeral" }}
func New{{ .Spec.Name | Pascal }}Ephemeral() ephemeral.EphemeralResource { return &{{ .Spec.Name | Pascal }}Ephemeral{} }

type {{ .Spec.Name | Pascal }}Ephemeral struct{ client *APIClient }

func (e *{{ .Spec.Name | Pascal }}Ephemeral) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) { resp.TypeName = req.ProviderTypeName + "_{{ .Spec.Name }}" }
func (e *{{ .Spec.Name | Pascal }}Ephemeral) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	attrs := map[string]ephschema.Attribute{}
	{{- range .Spec.Fields }}
	attrs["{{ .Name }}"] = {{ EphAttr . }}
	{{- end }}
	resp.Schema = ephschema.Schema{ Attributes: attrs }
}
func (e *{{ .Spec.Name | Pascal }}Ephemeral) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil { return }
	c, ok := req.ProviderData.(*APIClient)
	if ok { e.client = c }
}
func (e *{{ .Spec.Name | Pascal }}Ephemeral) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if e.client == nil { return }
	conn, err := grpc.Dial(e.client.GRPCTarget(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { return }
	defer conn.Close()
	cli := NewGrpcTerraformServiceClient(conn)
	if e.client.Bearer != "" { ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+e.client.Bearer) }
	_, _ = cli.{{ .Spec.Name | Pascal }}Open(ctx, &{{ .Spec.Name | Pascal }}OpenRequest{})
}
{{- end }}
`

const modelTemplate = `// Code generated by componentgen; DO NOT EDIT.
package generated

import "github.com/hashicorp/terraform-plugin-framework/types"

type {{ .Name | Pascal }}Model struct {
{{ range .Fields }}	{{ .Name | Pascal }} types.String ` + "`tfsdk:\"{{ .Name }}\"`" + `
{{ end }}}
`

const combinedProtoTemplate = `// Code generated by componentgen; DO NOT EDIT.
syntax = "proto3";
package {{ .Package }};
{{ if .HTTPImport }}import "google/api/annotations.proto";{{ end }}

option go_package = "{{ .GoPackage }}";

// Messages
{{ range .Specs }}
// Component: {{ .Spec.Name }}
{{ range .Messages }}
message {{ .Name }} {
{{ range .Fields }}	{{ if .Repeated }}repeated {{ end }}{{ .Type }} {{ .Name }} = {{ .Tag }}; // {{ .Comment }}
{{ end }}
}
{{ end }}
{{ end }}

// Combined Service
service {{ .Service }} {
{{ range .Specs }}
{{ if .HasResource }}	rpc {{ .Spec.Name | Pascal }}Create({{ .Spec.Name | Pascal }}CreateRequest) returns ({{ .Spec.Name | Pascal }}CreateResponse) {{ if and $.HTTP .Spec.Gateway }}{
		option (google.api.http) = { post: "{{ HttpCreate .Spec.Name }}" body: "item" };
	}{{ else }};{{ end }}
	rpc {{ .Spec.Name | Pascal }}Read({{ .Spec.Name | Pascal }}ReadRequest) returns ({{ .Spec.Name | Pascal }}ReadResponse) {{ if and $.HTTP .Spec.Gateway }}{
		option (google.api.http) = { get: "{{ HttpRead .Spec.Name }}" };
	}{{ else }};{{ end }}
	rpc {{ .Spec.Name | Pascal }}Update({{ .Spec.Name | Pascal }}UpdateRequest) returns ({{ .Spec.Name | Pascal }}UpdateResponse) {{ if and $.HTTP .Spec.Gateway }}{
		option (google.api.http) = { put: "{{ HttpUpdate .Spec.Name }}" body: "item" };
	}{{ else }};{{ end }}
	rpc {{ .Spec.Name | Pascal }}Delete({{ .Spec.Name | Pascal }}DeleteRequest) returns ({{ .Spec.Name | Pascal }}DeleteResponse) {{ if and $.HTTP .Spec.Gateway }}{
		option (google.api.http) = { delete: "{{ HttpDelete .Spec.Name }}" };
	}{{ else }};{{ end }}
{{ end }}
{{ if .HasDataSource }}	rpc {{ .Spec.Name | Pascal }}DataSourceRead({{ .Spec.Name | Pascal }}DataSourceReadRequest) returns ({{ .Spec.Name | Pascal }}DataSourceReadResponse) {{ if and $.HTTP .Spec.Gateway }}{
		option (google.api.http) = { get: "{{ HttpDsRead .Spec.Name }}" };
	}{{ else }};{{ end }}
{{ end }}
{{ if .HasEphemeral }}	rpc {{ .Spec.Name | Pascal }}Open({{ .Spec.Name | Pascal }}OpenRequest) returns ({{ .Spec.Name | Pascal }}OpenResponse) {{ if and $.HTTP .Spec.Gateway }}{
		option (google.api.http) = { post: "{{ HttpOpen .Spec.Name }}" body: "item" };
	}{{ else }};{{ end }}
{{ end }}
{{ end }}
}
`

// Registry template remains and is used by generateRegistry
const registryTemplate = `// Code generated by componentgen; DO NOT EDIT.
package generated

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
)

// Components exposes helper constructors; used by provider for registration.
var Components = struct {
	Resources []func() resource.Resource
	DataSources []func() datasource.DataSource
	EphemeralResources []func() ephemeral.EphemeralResource
}{
	Resources: []func() resource.Resource{ {{- range . }} {{ if .HasRole "resource" }} New{{ .Name | Pascal }}Resource, {{ end }} {{- end }} },
	DataSources: []func() datasource.DataSource{ {{- range . }} {{ if .HasRole "datasource" }} New{{ .Name | Pascal }}DataSource, {{ end }} {{- end }} },
	EphemeralResources: []func() ephemeral.EphemeralResource{ {{- range . }} {{ if .HasRole "ephemeral" }} New{{ .Name | Pascal }}Ephemeral, {{ end }} {{- end }} },
}
`

// Validate templates existence at init.
func init() {
	for _, tpl := range []string{goRoleTemplateV2, registryTemplate} {
		if strings.TrimSpace(tpl) == "" {
			panic(errors.New("empty template"))
		}
	}
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"Title":             strings.Title,
		"Pascal":            pascal,
		"ProtoType":         protoType,
		"Index":             func(i int) int { return i + 1 },
		"Join":              strings.Join,
		"Quote":             func(s string) string { return fmt.Sprintf("\"%s\"", s) },
		"ModelType":         modelFieldType,
		"HasDefault":        specHasDefault,
		"HasReplaceString":  specHasReplaceString,
		"HasReplaceFloat64": specHasReplaceFloat64,
		"HasReplaceBool":    specHasReplaceBool,
		"HasReplaceInt64":   specHasReplaceInt64,
		"HttpCreate":        httpCreatePath,
		"HttpRead":          httpReadPath,
		"HttpUpdate":        httpUpdatePath,
		"HttpDelete":        httpDeletePath,
		"HttpDsRead":        httpDsReadPath,
		"HttpOpen":          httpOpenPath,
		"ResourceAttr":      resourceAttrCode,
		"DSAttr":            dsAttrCode,
		"EphAttr":           ephAttrCode,
	}
}

// String helpers
func pascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		// Title deprecated; implement simple Pascal casing
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, "")
}

// deriveModeFlags determines effective Optional/Computed and extra flags based on Field.Mode.
func deriveModeFlags(f component.Field) (optional bool, computed bool, extras []string) {
	optional = f.Optional
	computed = f.Computed
	switch f.Mode {
	case component.KeyAttributeMode:
		optional = false
		computed = false
	case component.IDAttributeMode:
		optional = false
		computed = true
	case component.ReadOnlyAttributeMode:
		optional = false
		computed = true
	case component.ReadOnlyOnceAttributeMode:
		optional = false
		computed = true
	case component.WriteOnlyAttributeMode:
		optional = true
		computed = false
		extras = append(extras, "Sensitive: true")
	case component.ImmutableAttributeMode:
		// Leave flags as declared; immutability enforcement occurs via plan modifiers outside generator
	}
	return
}

// Common flag parts for attributes
func commonFlags(f component.Field) []string {
	var parts []string
	if f.Description != "" {
		parts = append(parts, fmt.Sprintf("MarkdownDescription: %q", f.Description))
	}
	opt, comp, extras := deriveModeFlags(f)
	if opt {
		parts = append(parts, "Optional: true")
	}
	if comp {
		parts = append(parts, "Computed: true")
	}
	if !opt && !comp {
		parts = append(parts, "Required: true")
	}
	parts = append(parts, extras...)
	return parts
}

func elementTypeToken(elem string) string {
	switch elem {
	case "string":
		return "types.StringType"
	case "number":
		return "types.Float64Type"
	case "bool":
		return "types.BoolType"
	case "int64":
		return "types.Int64Type"
	case "int32":
		// Framework uses Int64Type; coercing
		return "types.Int64Type"
	default:
		return "types.StringType"
	}
}

func resourceAttrCode(f component.Field) string {
	replacement := func(typ string) string {
		if !f.RequiresReplace {
			return ""
		}
		// All attribute types support generic RequiresReplace plan modifier; for primitives use concrete helpers
		switch typ {
		case "string":
			return "PlanModifiers: []planmodifier.String{ stringplanmodifier.RequiresReplace() }"
		case "number":
			return "PlanModifiers: []planmodifier.Float64{ float64planmodifier.RequiresReplace() }"
		case "bool":
			return "PlanModifiers: []planmodifier.Bool{ boolplanmodifier.RequiresReplace() }"
		case "int64", "int32":
			return "PlanModifiers: []planmodifier.Int64{ int64planmodifier.RequiresReplace() }"
		default:
			// Nested/dynamic/list: apply to container when possible
			return "PlanModifiers: []planmodifier.Object{ }" // placeholder no-op for non-primitive
		}
	}
	switch f.Type {
	case "string":
		flags := append(commonFlags(f), defaultPart(f))
		if r := replacement("string"); r != "" {
			flags = append(flags, r)
		}
		return fmt.Sprintf("schema.StringAttribute{ %s }", strings.Join(filterEmpty(flags), ", "))
	case "number":
		flags := commonFlags(f)
		if r := replacement("number"); r != "" {
			flags = append(flags, r)
		}
		return fmt.Sprintf("schema.Float64Attribute{ %s }", strings.Join(flags, ", "))
	case "bool":
		flags := commonFlags(f)
		if r := replacement("bool"); r != "" {
			flags = append(flags, r)
		}
		return fmt.Sprintf("schema.BoolAttribute{ %s }", strings.Join(flags, ", "))
	case "int64":
		flags := commonFlags(f)
		if r := replacement("int64"); r != "" {
			flags = append(flags, r)
		}
		return fmt.Sprintf("schema.Int64Attribute{ %s }", strings.Join(flags, ", "))
	case "int32":
		flags := commonFlags(f)
		if r := replacement("int32"); r != "" {
			flags = append(flags, r)
		}
		return fmt.Sprintf("schema.Int64Attribute{ %s }", strings.Join(flags, ", "))
	case "dynamic":
		flags := commonFlags(f)
		return fmt.Sprintf("schema.DynamicAttribute{ %s }", strings.Join(flags, ", "))
	case "object":
		var b strings.Builder
		fmt.Fprintf(&b, "schema.SingleNestedAttribute{ ")
		fmt.Fprintf(&b, "Attributes: map[string]schema.Attribute{")
		for _, nf := range f.Fields {
			fmt.Fprintf(&b, "\"%s\": %s, ", nf.Name, resourceAttrCode(nf))
		}
		flags := commonFlags(f)
		fmt.Fprintf(&b, "}, %s }", strings.Join(flags, ", "))
		return b.String()
	case "list":
		if f.ElemType == "object" {
			var b strings.Builder
			fmt.Fprintf(&b, "schema.ListNestedAttribute{ ")
			fmt.Fprintf(&b, "NestedObject: schema.NestedAttributeObject{ Attributes: map[string]schema.Attribute{")
			for _, nf := range f.Fields {
				fmt.Fprintf(&b, "\"%s\": %s, ", nf.Name, resourceAttrCode(nf))
			}
			flags := commonFlags(f)
			fmt.Fprintf(&b, "}}, %s }", strings.Join(flags, ", "))
			return b.String()
		}
		flags := commonFlags(f)
		flags = append([]string{fmt.Sprintf("ElementType: %s", elementTypeToken(f.ElemType))}, flags...)
		return fmt.Sprintf("schema.ListAttribute{ %s }", strings.Join(flags, ", "))
	default:
		return fmt.Sprintf("schema.StringAttribute{ %s }", strings.Join(commonFlags(f), ", "))
	}
}

func dsAttrCode(f component.Field) string {
	switch f.Type {
	case "string":
		return fmt.Sprintf("dsschema.StringAttribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "number":
		return fmt.Sprintf("dsschema.Float64Attribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "bool":
		return fmt.Sprintf("dsschema.BoolAttribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "int64":
		return fmt.Sprintf("dsschema.Int64Attribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "int32":
		return fmt.Sprintf("dsschema.Int64Attribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "dynamic":
		return fmt.Sprintf("dsschema.DynamicAttribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "object":
		var b strings.Builder
		fmt.Fprintf(&b, "dsschema.SingleNestedAttribute{ ")
		fmt.Fprintf(&b, "Attributes: map[string]dsschema.Attribute{")
		for _, nf := range f.Fields {
			fmt.Fprintf(&b, "\"%s\": %s, ", nf.Name, dsAttrCode(nf))
		}
		fmt.Fprintf(&b, "}, %s }", strings.Join(commonFlags(f), ", "))
		return b.String()
	case "list":
		if f.ElemType == "object" {
			var b strings.Builder
			fmt.Fprintf(&b, "dsschema.ListNestedAttribute{ ")
			fmt.Fprintf(&b, "NestedObject: dsschema.NestedAttributeObject{ Attributes: map[string]dsschema.Attribute{")
			for _, nf := range f.Fields {
				fmt.Fprintf(&b, "\"%s\": %s, ", nf.Name, dsAttrCode(nf))
			}
			fmt.Fprintf(&b, "}}, %s }", strings.Join(commonFlags(f), ", "))
			return b.String()
		}
		flags := commonFlags(f)
		flags = append([]string{fmt.Sprintf("ElementType: %s", elementTypeToken(f.ElemType))}, flags...)
		return fmt.Sprintf("dsschema.ListAttribute{ %s }", strings.Join(flags, ", "))
	default:
		return fmt.Sprintf("dsschema.StringAttribute{ %s }", strings.Join(commonFlags(f), ", "))
	}
}

func ephAttrCode(f component.Field) string {
	switch f.Type {
	case "string":
		return fmt.Sprintf("ephschema.StringAttribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "number":
		return fmt.Sprintf("ephschema.Float64Attribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "bool":
		return fmt.Sprintf("ephschema.BoolAttribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "int64":
		return fmt.Sprintf("ephschema.Int64Attribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "int32":
		return fmt.Sprintf("ephschema.Int64Attribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "dynamic":
		return fmt.Sprintf("ephschema.DynamicAttribute{ %s }", strings.Join(commonFlags(f), ", "))
	case "object":
		var b strings.Builder
		fmt.Fprintf(&b, "ephschema.SingleNestedAttribute{ ")
		fmt.Fprintf(&b, "Attributes: map[string]ephschema.Attribute{")
		for _, nf := range f.Fields {
			fmt.Fprintf(&b, "\"%s\": %s, ", nf.Name, ephAttrCode(nf))
		}
		fmt.Fprintf(&b, "}, %s }", strings.Join(commonFlags(f), ", "))
		return b.String()
	case "list":
		flags := commonFlags(f)
		if f.ElemType == "object" {
			var b strings.Builder
			fmt.Fprintf(&b, "ephschema.ListNestedAttribute{ ")
			fmt.Fprintf(&b, "NestedObject: ephschema.NestedAttributeObject{ Attributes: map[string]ephschema.Attribute{")
			for _, nf := range f.Fields {
				fmt.Fprintf(&b, "\"%s\": %s, ", nf.Name, ephAttrCode(nf))
			}
			fmt.Fprintf(&b, "}}, %s }", strings.Join(flags, ", "))
			return b.String()
		}
		flags = append([]string{fmt.Sprintf("ElementType: %s", elementTypeToken(f.ElemType))}, flags...)
		return fmt.Sprintf("ephschema.ListAttribute{ %s }", strings.Join(flags, ", "))
	default:
		return fmt.Sprintf("ephschema.StringAttribute{ %s }", strings.Join(commonFlags(f), ", "))
	}
}

func defaultPart(f component.Field) string {
	if f.Default != "" {
		return fmt.Sprintf("Default: stringdefault.StaticString(\"%s\")", f.Default)
	}
	return ""
}

func filterEmpty(parts []string) []string {
	var out []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// Return proto scalar type for a field (string/double/bool/int64/int32)
func protoType(f component.Field) string {
	switch f.Type {
	case "string":
		return "string"
	case "number":
		return "double"
	case "bool":
		return "bool"
	case "int64":
		return "int64"
	case "int32":
		return "int32"
	default:
		return "string"
	}
}

// Model types for Terraform framework model struct
func modelFieldType(f component.Field) string {
	switch f.Type {
	case "string":
		return "types.String"
	case "number":
		return "types.Float64"
	case "bool":
		return "types.Bool"
	case "int64":
		return "types.Int64"
	case "int32":
		return "types.Int64"
	default:
		return "types.String"
	}
}

func specHasDefault(s component.Spec) bool {
	for _, f := range s.Fields {
		if f.Default != "" {
			return true
		}
	}
	return false
}

func specHasReplaceString(s component.Spec) bool  { return hasReplaceOnType(s.Fields, "string") }
func specHasReplaceFloat64(s component.Spec) bool { return hasReplaceOnType(s.Fields, "number") }
func specHasReplaceBool(s component.Spec) bool    { return hasReplaceOnType(s.Fields, "bool") }
func specHasReplaceInt64(s component.Spec) bool {
	return hasReplaceOnType(s.Fields, "int64") || hasReplaceOnType(s.Fields, "int32")
}

func hasReplaceOnType(fields []component.Field, typ string) bool {
	for _, f := range fields {
		if f.Type == typ && f.RequiresReplace {
			return true
		}
		if f.Type == "object" {
			if hasReplaceOnType(f.Fields, typ) {
				return true
			}
		}
		if f.Type == "list" && f.ElemType == "object" {
			if hasReplaceOnType(f.Fields, typ) {
				return true
			}
		}
	}
	return false
}
