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

func GenerateModel(spec component.Spec, outDir string) error {
	file := filepath.Join(outDir, fmt.Sprintf("%s_model.gen.go", spec.Name))
	var buf bytes.Buffer
	t := template.Must(template.New("model").Funcs(funcMap()).Parse(modelTemplate))
	if err := t.Execute(&buf, spec); err != nil {
		return err
	}
	return os.WriteFile(file, buf.Bytes(), 0o644)
}
