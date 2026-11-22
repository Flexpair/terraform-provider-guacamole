package main

import (
	"log"

	"github.com/Flexpair/terraform-provider-guacamole/guacamole"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

// version and commit are stamped by GoReleaser via -ldflags (see .goreleaser.yml).
var (
	version = "dev"
	commit  = "none"
)

func main() {
	log.Printf("[INFO] terraform-provider-guacamole %s (commit %s)", version, commit)
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return guacamole.Provider()
		},
	})
}
