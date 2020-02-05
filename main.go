package main

import (
	"fmt"
	"log"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
)

// Handler represents the sensu-puppet-handler plugin
type Handler struct {
	sensu.PluginConfig
	endpoint string
}

var (
	handler = Handler{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-puppet-handler",
			Short:    "Deregister Sensu entities without an associated Puppet node",
			Timeout:  10,
			Keyspace: "sensu.io/plugins/sensu-puppet-handler/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		&sensu.PluginConfigOption{
			Path:      "endpoint",
			Env:       "PUPPET_ENDPOINT",
			Argument:  "endpoint",
			Shorthand: "e",
			Default:   "",
			Usage:     "The PuppetDB API endpoint (URL). If an API path is not specified, /pdb/query/v4/nodes/ will be used",
			Value:     &handler.endpoint,
		},
	}
)

func main() {
	handler := sensu.NewGoHandler(&handler.PluginConfig, options, checkArgs, executeHandler)
	handler.Execute()
}

func checkArgs(_ *types.Event) error {
	if len(handler.endpoint) == 0 {
		return fmt.Errorf("--example or HANDLER_EXAMPLE environment variable is required")
	}
	return nil
}

func executeHandler(event *types.Event) error {
	log.Println("executing handler")
	return nil
}
