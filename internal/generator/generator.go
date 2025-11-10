// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gshireesh/terraform-provider-shireesh/internal/component"
)

// GenerateAll orchestrates tag loading, code generation, proto assembly and registry/defaults.
// This file kept small (<500 lines) by delegating to other modules.
func GenerateAll(specs []component.Spec, outDir string, tagFile string, includeHTTP bool, serviceName string, protoPackage string, goPackagePrefix string) error {
	return GenerateAllWithAuthProto(specs, outDir, tagFile, includeHTTP, serviceName, protoPackage, goPackagePrefix, "api", nil)
}

// Original signature (without protoOut) retained for compatibility.
func GenerateAllWithAuth(specs []component.Spec, outDir string, tagFile string, includeHTTP bool, serviceName string, protoPackage string, goPackagePrefix string, authOverride *AuthConfig) error {
	return GenerateAllWithAuthProto(specs, outDir, tagFile, includeHTTP, serviceName, protoPackage, goPackagePrefix, "api", authOverride)
}

// New signature with protoOut explicit.
func GenerateAllWithAuthProto(specs []component.Spec, outDir string, tagFile string, includeHTTP bool, serviceName string, protoPackage string, goPackagePrefix string, protoOut string, authOverride *AuthConfig) error {
	// Load or initialize tags
	ts := NewTagStore()
	if b, err := os.ReadFile(tagFile); err == nil {
		ts.Load(string(b))
	}
	// Ensure tags for current specs
	for _, spec := range specs {
		AssignTagsForSpec(ts, spec)
	}
	// Persist tags
	var tagBuf bytes.Buffer
	tagBuf.WriteString(ts.String())
	if err := os.WriteFile(tagFile, tagBuf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write tags: %w", err)
	}
	// Generate role & model files
	for _, spec := range specs {
		if err := GenerateModel(spec, outDir); err != nil {
			return err
		}
		if err := GenerateRoles(spec, outDir); err != nil {
			return err
		}
	}

	// proto output dir now root-level configurable
	protoOutDir := protoOut
	if err := os.MkdirAll(protoOutDir, 0o755); err != nil {
		return fmt.Errorf("create proto out dir: %w", err)
	}

	// Clean legacy proto files
	CleanLegacyProtoFiles(protoOutDir)
	// Also remove old internal path if different
	legacyInternalAPI := filepath.Join("internal", "provider", "generated", "api")
	if protoOutDir != legacyInternalAPI {
		_ = os.RemoveAll(legacyInternalAPI)
	}
	// Load optional auth OpenAPI config from spec directory or override
	authCfg := authOverride
	if authCfg == nil {
		authCfg = LoadAuthConfig(filepath.Dir(tagFile))
	}
	// Combined proto
	if err := GenerateCombinedProto(specs, protoOutDir, includeHTTP, serviceName, protoPackage, goPackagePrefix, ts, authCfg); err != nil {
		return err
	}
	// Registry & defaults
	if err := GenerateRegistry(specs, outDir); err != nil {
		return err
	}
	if err := GenerateDefaults(outDir); err != nil {
		return err
	}
	return nil
}
