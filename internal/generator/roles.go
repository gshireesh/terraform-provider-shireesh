// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gshireesh/terraform-provider-shireesh/internal/component"
)

func GenerateRoles(spec component.Spec, outDir string) error {
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
