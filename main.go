// Copyright (c) slop-incubator
// SPDX-License-Identifier: MPL-2.0

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name kanidm

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/slop-incubator/terraform-provider-kanidm/internal/provider"
)

// version is set by GoReleaser via -ldflags at build time.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.opentofu.org/slop-incubator/kanidm",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err)
	}
}
