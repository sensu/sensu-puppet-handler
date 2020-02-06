package main

import (
	"errors"
	"fmt"
	"net/url"
	"path"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
)

// Handler represents the sensu-puppet-handler plugin
type Handler struct {
	sensu.PluginConfig
	endpoint           string
	keystoreFile       string
	keystorePassword   string
	truststoreFile     string
	truststorePassword string
	httpProxy          string
	timeout            int
}

const (
	defaultAPIPath = "pdb/query/v4/nodes"
)

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
			Usage:     "the PuppetDB API endpoint (URL). If an API path is not specified, /pdb/query/v4/nodes/ will be used",
			Value:     &handler.endpoint,
		},
		&sensu.PluginConfigOption{
			Path:     "keystore_file",
			Env:      "PUPPET_KEYSTORE_FILE",
			Argument: "keystore_file",
			Usage:    "the file path for the SSL certificate keystore",
			Value:    &handler.keystoreFile,
		},
		&sensu.PluginConfigOption{
			Path:     "keystore_password",
			Env:      "PUPPET_KEYSTORE_PASSWORD",
			Argument: "keystore_password",
			Usage:    "the SSL certificate keystore password",
			Value:    &handler.keystorePassword,
		},
		&sensu.PluginConfigOption{
			Path:     "truststore_file",
			Env:      "PUPPET_TRUSTSTORE_FILE",
			Argument: "truststore_file",
			Usage:    "the file path for the SSL certificate truststore",
			Value:    &handler.truststoreFile,
		},
		&sensu.PluginConfigOption{
			Path:     "truststore_password",
			Env:      "PUPPET_TRUSTSTORE_PASSWORD",
			Argument: "truststore_password",
			Usage:    "the SSL certificate truststore password",
			Value:    &handler.truststorePassword,
		},
		&sensu.PluginConfigOption{
			Path:     "http_proxy",
			Env:      "PUPPET_HTTP_PROXY",
			Argument: "http_proxy",
			Usage:    "the URL of a proxy to be used for HTTP requests",
			Value:    &handler.httpProxy,
		},
		&sensu.PluginConfigOption{
			Path:     "timeout",
			Env:      "PUPPET_TIMEOUT",
			Argument: "timeout",
			Usage:    "the handler execution duration timeout in seconds (hard stop)",
			Value:    &handler.httpProxy,
		},
	}
)

func main() {
	handler := sensu.NewGoHandler(&handler.PluginConfig, options, validate, executeHandler)
	handler.Execute()
}

func validate(_ *types.Event) error {
	// Make sure all required options are provided
	if handler.endpoint == "" {
		return errors.New("the PuppetDB API endpoint is required")
	}
	if handler.keystoreFile == "" {
		return errors.New("the path to the SSL certificate keystore is required")
	}
	if handler.keystorePassword == "" {
		return errors.New("the SSL certificate keystore password is required")
	}
	if handler.truststoreFile == "" {
		return errors.New("the path for the SSL certificate truststore is required")
	}
	if handler.truststorePassword == "" {
		return errors.New("the SSL certificate truststore password is required")
	}

	// Make sure the endpoint URL is valid
	u, err := url.Parse(handler.endpoint)
	if err != nil {
		return fmt.Errorf("invalid PuppetDB API endpoint URL: %s", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return errors.New("invalid PuppetDB API endpoint URL")
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = path.Join(u.Path, defaultAPIPath)
		handler.endpoint = u.String()
	}

	return nil
}

func executeHandler(event *types.Event) error {
	return nil
}
