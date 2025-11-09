// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// Unit test ensuring New returns a non-nil provider instance.
func TestNewProviderCreated(t *testing.T) {
	p := New("test")()
	if p == nil {
		t.Fatalf("expected non-nil provider")
	}
}

// Verify Configure does not error when optional config omitted.
func TestConfigureWithoutConfig(t *testing.T) {
	prov, ok := New("test")().(*ScaffoldingProvider)
	if !ok {
		t.Fatalf("expected *ScaffoldingProvider type")
	}
	ctx := context.Background()
	// Initialize schema so Config.Get doesn't panic.
	var schemaResp provider.SchemaResponse
	prov.Schema(ctx, provider.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Attributes == nil {
		t.Fatalf("expected schema attributes to be initialized")
	}
	var req provider.ConfigureRequest
	// Provide an empty config matching schema.
	req.Config = tfsdk.Config{}
	var resp provider.ConfigureResponse
	prov.Configure(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		for _, d := range resp.Diagnostics {
			t.Logf("diagnostic: %#v", d)
		}
		t.Fatalf("unexpected diagnostics errors")
	}
	if resp.ResourceData == nil || resp.DataSourceData == nil {
		t.Fatalf("expected both resource and datasource data to be set")
	}
}
