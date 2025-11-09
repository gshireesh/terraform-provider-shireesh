// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// docsectioner post-processes generated docs to inject subcategory front matter based on component.Section.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gshireesh/terraform-provider-shireesh/components"
)

// Build map of type name -> section
func sections() map[string]string {
	return map[string]string{
		"scaffolding_simple":  components.SimpleSpec().Section,
		"scaffolding_complex": components.ComplexSpec().Section,
		"scaffolding_modes":   components.ModesSpec().Section,
	}
}

var fmLine = regexp.MustCompile(`^---\s*$`)

func main() {
	secMap := sections()
	root := "docs"
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		if !(strings.Contains(path, "/resources/") || strings.Contains(path, "/data-sources/") || strings.Contains(path, "/ephemeral-resources/")) {
			return nil
		}
		return rewriteFile(path, secMap)
	})
}

func rewriteFile(path string, secMap map[string]string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := bytes.Split(b, []byte("\n"))
	if len(lines) < 3 {
		return nil
	}
	// Determine type from file name
	name := strings.TrimSuffix(filepath.Base(path), ".md")
	full := "scaffolding_" + name
	section := secMap[full]
	// Locate YAML front matter start/end anywhere in the first 50 lines
	start, end := -1, -1
	for i := 0; i < len(lines) && i < 50; i++ {
		if fmLine.Match(lines[i]) {
			start = i
			break
		}
	}
	if start == -1 {
		return nil
	}
	for i := start + 1; i < len(lines); i++ {
		if fmLine.Match(lines[i]) {
			end = i
			break
		}
	}
	if end == -1 {
		return nil
	}
	mutated := false
	// Inject subcategory if available and missing
	if section != "" {
		present := false
		for i := start + 1; i < end; i++ {
			if bytes.HasPrefix(bytes.TrimSpace(lines[i]), []byte("subcategory:")) {
				present = true
				break
			}
		}
		if !present {
			insert := []byte(fmt.Sprintf("subcategory: \"%s\"", section))
			nl := make([][]byte, 0, len(lines)+1)
			nl = append(nl, lines[:end]...)
			nl = append(nl, insert)
			nl = append(nl, lines[end:]...)
			lines = nl
			end++
			mutated = true
		}
	}
	// Rewrite page_title to drop provider prefix once
	for i := start + 1; i < end; i++ {
		trim := bytes.TrimSpace(lines[i])
		if bytes.HasPrefix(trim, []byte("page_title:")) {
			// Replace first occurrence of full with name inside quotes
			repl := bytes.Replace(lines[i], []byte(full+" "), []byte(name+" "), 1)
			if !bytes.Equal(repl, lines[i]) {
				lines[i] = repl
				mutated = true
			}
			break
		}
	}
	// Rewrite first H1 heading after front matter
	for i := end + 1; i < len(lines); i++ {
		trim := bytes.TrimSpace(lines[i])
		if bytes.HasPrefix(trim, []byte("# ")) {
			repl := bytes.Replace(lines[i], []byte("# "+full+" "), []byte("# "+name+" "), 1)
			if !bytes.Equal(repl, lines[i]) {
				lines[i] = repl
				mutated = true
			}
			break
		}
	}
	if !mutated {
		return nil
	}
	// Write back
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for i, l := range lines {
		w.Write(l)
		if i < len(lines)-1 {
			w.WriteByte('\n')
		}
	}
	w.Flush()
	f.Close()
	return nil
}
