// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gshireesh/terraform-provider-shireesh/internal/component"
)

// AuthConfig holds optional swagger/auth info loaded from auth.yaml or similar.
type AuthConfig struct {
	Title      string `yaml:"title"`
	Version    string `yaml:"version"`
	Host       string `yaml:"host"`
	TokenURL   string `yaml:"token_url"`
	ReadScope  string `yaml:"read_scope"`
	WriteScope string `yaml:"write_scope"`
	Package    string `yaml:"package"`
	GoPackage  string `yaml:"go_package"`
}

// LoadAuthConfig stub (actual YAML loading added later if file present).
func LoadAuthConfig(dir string) *AuthConfig {
	path := filepath.Join(dir, "auth.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg AuthConfig
	// lazy YAML parse to keep file short
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		switch k {
		case "title":
			cfg.Title = v
		case "version":
			cfg.Version = v
		case "host":
			cfg.Host = v
		case "token_url":
			cfg.TokenURL = v
		case "read_scope":
			cfg.ReadScope = v
		case "write_scope":
			cfg.WriteScope = v
		case "package":
			cfg.Package = v
		case "go_package":
			cfg.GoPackage = v
		}
	}
	return &cfg
}

func CleanLegacyProtoFiles(outDir string) {
	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".proto" && e.Name() != "components.proto" {
			_ = os.Remove(filepath.Join(outDir, e.Name()))
		}
	}
}

func GenerateCombinedProto(specs []component.Spec, outDir string, includeHTTP bool, serviceName string, protoPackage string, goPackagePrefix string, ts *TagStore, authCfg *AuthConfig) error {
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
	data := struct {
		Specs                       []CombinedProtoSpec
		HTTP, HTTPImport            bool
		Service, Package, GoPackage string
		Auth                        *AuthConfig
	}{Specs: all, HTTP: includeHTTP, HTTPImport: includeHTTP && anyGateway, Service: serviceName, Package: protoPackage, GoPackage: goPackagePrefix, Auth: authCfg}
	t := template.Must(template.New("combined_proto").Funcs(funcMap()).Parse(combinedProtoTemplateAuth))
	if err := t.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(file, buf.Bytes(), 0o644)
}

// CombinedProtoSpec holds spec + derived messages for template.
type CombinedProtoSpec struct {
	Spec                                     component.Spec
	Messages                                 []ProtoMessage
	HasResource, HasDataSource, HasEphemeral bool
}

// ProtoMessage and field structures
// ...existing code...

type ProtoField struct {
	Name, stringType string
	Tag              int
	Comment          string
	Repeated         bool
}

// buildProtoMessages minimal wrapper referencing tag store
func buildProtoMessages(spec component.Spec, ts *TagStore) []ProtoMessage {
	// simplified passthrough - reuse previous logic by delegating (could move full logic if needed)
	return buildProtoMessagesLegacy(spec, ts)
}

// Legacy full logic moved to legacy_build.go for clarity
