// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package generator

import (
	"fmt"
	"strings"
)

func pascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, "")
}

func httpCreatePath(name string) string { return fmt.Sprintf("/v1/%s", name) }
func httpReadPath(name string) string   { return fmt.Sprintf("/v1/%s/{id}", name) }
func httpUpdatePath(name string) string { return fmt.Sprintf("/v1/%s/{id}", name) }
func httpDeletePath(name string) string { return fmt.Sprintf("/v1/%s/{id}", name) }
func httpDsReadPath(name string) string { return fmt.Sprintf("/v1/datasource/%s/{id}", name) }
func httpOpenPath(name string) string   { return fmt.Sprintf("/v1/%s/open", name) }
