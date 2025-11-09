package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// Basic unit test to ensure New() returns a Provider implementing expected interfaces
func TestNewProviderImplementsInterfaces(t *testing.T) {
	p := New("test")()
	if _, ok := p.(provider.Provider); !ok {
		t.Fatalf("provider does not implement provider.Provider")
	}
}

// Verify Configure does not error when optional config omitted
func TestConfigureWithoutConfig(t *testing.T) {
	p := New("test")().(*ScaffoldingProvider)
	ctx := context.Background()
	// Initialize schema so Config.Get doesn't panic
	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Attributes == nil {
		t.Fatalf("expected schema attributes to be initialized")
	}
	var req provider.ConfigureRequest
	// Provide an empty config matching schema
	req.Config = tfsdk.Config{} // zero value is acceptable for empty config
	var resp provider.ConfigureResponse
	p.Configure(ctx, req, &resp)
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
