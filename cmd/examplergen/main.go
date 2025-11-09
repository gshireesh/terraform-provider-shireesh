// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// examplergen generates Terraform examples for resources, data sources, and ephemeral resources
// for documentation. It will generate examples only for missing ones and leave any existing
// examples untouched.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/hashicorp/terraform-provider-scaffolding-framework/internal/provider/generated"
)

const providerType = "scaffolding"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "examplergen: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Always ensure provider example exists
	if err := ensureFile(
		filepath.Join("examples", "provider", "provider.tf"),
		"provider \""+providerType+"\" {\n  # example configuration here\n}\n",
	); err != nil {
		return err
	}

	// Resources
	for _, ctor := range generated.Components.Resources {
		r := ctor()
		typeName := resourceTypeName(r)
		if typeName == "" {
			continue
		}
		path := filepath.Join("examples", "resources", typeName, "resource.tf")
		content := fmt.Sprintf("resource \"%s\" \"example\" {\n  # add required arguments here\n}\n", typeName)
		if err := ensureFile(path, content); err != nil {
			return err
		}
	}

	// Data sources
	for _, ctor := range generated.Components.DataSources {
		ds := ctor()
		typeName := dataSourceTypeName(ds)
		if typeName == "" {
			continue
		}
		path := filepath.Join("examples", "data-sources", typeName, "data-source.tf")
		content := fmt.Sprintf("data \"%s\" \"example\" {\n  # lookup arguments here\n}\n", typeName)
		if err := ensureFile(path, content); err != nil {
			return err
		}
	}

	// Ephemeral
	for _, ctor := range generated.Components.EphemeralResources {
		e := ctor()
		typeName := ephemeralTypeName(e)
		if typeName == "" {
			continue
		}
		path := filepath.Join("examples", "ephemeral-resources", typeName, "ephemeral-resource.tf")
		content := fmt.Sprintf("ephemeral \"%s\" \"example\" {\n  # arguments here\n}\n", typeName)
		if err := ensureFile(path, content); err != nil {
			return err
		}
	}

	return nil
}

func ensureFile(path string, content string) error {
	if _, err := os.Stat(path); err == nil {
		// exists: skip
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func resourceTypeName(r resource.Resource) string {
	var resp resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: providerType}, &resp)
	return resp.TypeName
}

func dataSourceTypeName(d datasource.DataSource) string {
	var resp datasource.MetadataResponse
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: providerType}, &resp)
	return resp.TypeName
}

func ephemeralTypeName(e ephemeral.EphemeralResource) string {
	var resp ephemeral.MetadataResponse
	e.Metadata(context.Background(), ephemeral.MetadataRequest{ProviderTypeName: providerType}, &resp)
	return resp.TypeName
}
