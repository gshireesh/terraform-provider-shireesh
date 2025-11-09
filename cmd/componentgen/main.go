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
	)

	flag.StringVar(&outDir, "out", "internal/provider/generated", "output directory for generated files")
	flag.BoolVar(&includeHTTP, "http", false, "include grpc-gateway HTTP annotations in proto")
	flag.StringVar(&serviceName, "service-name", "GrpcTerraformService", "service name to use in combined proto")
	flag.StringVar(&protoPackage, "proto-package", "component", "protobuf package name")
	flag.StringVar(&goPackagePrefix, "go-package-prefix", "github.com/gshireesh/terraform-provider-shireesh/internal/provider/generated", "go package prefix for option go_package")
	flag.Parse()

	// Typed specs only
	specs := []component.Spec{components.SimpleSpec(), components.ComplexSpec(), components.ModesSpec()}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("create out dir: %v", err)
	}

	if err := generator.GenerateAll(specs, outDir, filepath.Join("components", ".tags"), includeHTTP, serviceName, protoPackage, goPackagePrefix); err != nil {
		log.Fatalf("generate: %v", err)
	}

	fmt.Printf("generated %d components into %s (http=%v service=%s proto_pkg=%s go_pkg_prefix=%s)\n", len(specs), outDir, includeHTTP, serviceName, protoPackage, goPackagePrefix)
}
