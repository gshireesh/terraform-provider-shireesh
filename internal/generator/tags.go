// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gshireesh/terraform-provider-shireesh/internal/component"
)

// TagStore tracks stable numeric tags per message.
type TagStore struct{ tags map[string]map[string]int }

func NewTagStore() *TagStore { return &TagStore{tags: map[string]map[string]int{}} }

func (s *TagStore) Load(content string) {
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

func (s *TagStore) max(msg string) int {
	m := 0
	for _, v := range s.tags[msg] {
		if v > m {
			m = v
		}
	}
	return m
}
func atoiSafe(in string) (int, error) {
	var n int
	for _, ch := range in {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("nan")
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}

func (s *TagStore) ensure(msg, field string) int {
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
func (s *TagStore) Tag(msg, field string) int {
	if m, ok := s.tags[msg]; ok {
		if t, ok := m[field]; ok {
			return t
		}
	}
	return s.ensure(msg, field)
}

// AssignTagsForSpec ensures tags for top-level and nested fields.
func AssignTagsForSpec(ts *TagStore, spec component.Spec) {
	for _, f := range spec.Fields {
		_ = ts.ensure(spec.Name, f.Name)
		if f.Type == "object" {
			nestedKey := spec.Name + "." + f.Name
			for _, nf := range f.Fields {
				_ = ts.ensure(nestedKey, nf.Name)
			}
		}
	}
}

func (s *TagStore) String() string {
	var msgs []string
	for k := range s.tags {
		msgs = append(msgs, k)
	}
	sort.Strings(msgs)
	var b strings.Builder
	for _, msg := range msgs {
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
