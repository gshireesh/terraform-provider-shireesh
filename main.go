// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/gshireesh/terraform-provider-shireesh/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version string = "dev"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

func main() {
	var debug bool
	var apiBase, tokenURL, clientID, clientSecret, scopes string

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.StringVar(&apiBase, "api-base-url", os.Getenv("SCAFFOLDING_API_BASE_URL"), "Override API base URL")
	flag.StringVar(&tokenURL, "oauth2-token-url", os.Getenv("SCAFFOLDING_OAUTH2_TOKEN_URL"), "OAuth2 token URL")
	flag.StringVar(&clientID, "client-id", os.Getenv("SCAFFOLDING_CLIENT_ID"), "OAuth2 client id")
	flag.StringVar(&clientSecret, "client-secret", os.Getenv("SCAFFOLDING_CLIENT_SECRET"), "OAuth2 client secret")
	flag.StringVar(&scopes, "scopes", os.Getenv("SCAFFOLDING_SCOPES"), "Comma separated OAuth2 scopes")
	flag.Parse()

	// Export to env so provider Configure picks them up without changing generated code
	if apiBase != "" {
		os.Setenv("SCAFFOLDING_API_BASE_URL", apiBase)
	}
	if tokenURL != "" {
		os.Setenv("SCAFFOLDING_OAUTH2_TOKEN_URL", tokenURL)
	}
	if clientID != "" {
		os.Setenv("SCAFFOLDING_CLIENT_ID", clientID)
	}
	if clientSecret != "" {
		os.Setenv("SCAFFOLDING_CLIENT_SECRET", clientSecret)
	}
	if scopes != "" {
		os.Setenv("SCAFFOLDING_SCOPES", scopes)
	}

	opts := providerserver.ServeOpts{
		// TODO: Update this string with the published name of your provider.
		// Also update the tfplugindocs generate command to either remove the
		// -provider-name flag or set its value to the updated provider name.
		Address: "registry.terraform.io/hashicorp/scaffolding",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
