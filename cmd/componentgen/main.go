// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gshireesh/terraform-provider-shireesh/components"
	"github.com/gshireesh/terraform-provider-shireesh/internal/component"
	"github.com/gshireesh/terraform-provider-shireesh/internal/generator"
)

func main() {
	var (
		outDir          string
		includeHTTP     bool
		serviceName     string
		protoPackage    string
		goPackagePrefix string
		// auth override flags
		authEnable     bool
		authPackage    string
		authGoPackage  string
		authTitle      string
		authVersion    string
		authHost       string
		authTokenURL   string
		authReadScope  string
		authWriteScope string
	)

	flag.StringVar(&outDir, "out", "internal/provider/generated", "output directory for generated files")
	flag.BoolVar(&includeHTTP, "http", true, "include grpc-gateway HTTP annotations in proto")
	flag.StringVar(&serviceName, "service-name", "GrpcTerraformService", "service name to use in combined proto")
	flag.StringVar(&protoPackage, "proto-package", "component", "protobuf package name")
	flag.StringVar(&goPackagePrefix, "go-package-prefix", "github.com/gshireesh/terraform-provider-shireesh/internal/provider/generated", "go package prefix for option go_package")
	flag.BoolVar(&authEnable, "auth", true, "enable carbon style auth swagger header override")
	flag.StringVar(&authPackage, "auth-package", "shireesh.com.api.carbon.v1", "auth proto package")
	flag.StringVar(&authGoPackage, "auth-go-package", "shireesh.com/api/carbon/v1", "auth go_package override")
	flag.StringVar(&authTitle, "auth-title", "Carbon API", "auth swagger title")
	flag.StringVar(&authVersion, "auth-version", "1.0.0", "auth swagger version")
	flag.StringVar(&authHost, "auth-host", "carbon.shireesh.com", "auth swagger host")
	flag.StringVar(&authTokenURL, "auth-token-url", "https://auth.shireesh.com/v1/oauth/token", "oauth2 token url")
	flag.StringVar(&authReadScope, "auth-read-scope", "read", "oauth2 read scope")
	flag.StringVar(&authWriteScope, "auth-write-scope", "write", "oauth2 write scope")
	flag.Parse()

	// Typed specs only
	specs := []component.Spec{
		components.SimpleSpec(),
		//components.ComplexSpec(),
		//components.ModesSpec(),
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("create out dir: %v", err)
	}

	var authCfg *generator.AuthConfig
	if authEnable {
		authCfg = &generator.AuthConfig{Title: authTitle, Version: authVersion, Host: authHost, TokenURL: authTokenURL, ReadScope: authReadScope, WriteScope: authWriteScope, Package: authPackage, GoPackage: authGoPackage}
	}

	if err := generator.GenerateAllWithAuth(specs, outDir, filepath.Join("components", ".tags"), includeHTTP, serviceName, protoPackage, goPackagePrefix, authCfg); err != nil {
		log.Fatalf("generate: %v", err)
	}

	fmt.Printf("generated %d components into %s (http=%v service=%s proto_pkg=%s go_pkg_prefix=%s)\n", len(specs), outDir, includeHTTP, serviceName, protoPackage, goPackagePrefix)
}
