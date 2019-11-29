package main

import (
	"github.com/hashicorp/terraform/plugin"
	provider "github.com/randomcoww/terraform-provider-libvirt/libvirt"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.Provider,
	})
}
