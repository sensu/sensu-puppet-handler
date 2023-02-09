package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"

	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-plugin-sdk/httpclient"
	"github.com/sensu/sensu-plugin-sdk/sensu"
)

// Handler represents the sensu-puppet-handler plugin
type Handler struct {
	sensu.PluginConfig
	endpoint                 string
	puppetCert               string
	puppetKey                string
	puppetCACert             string
	puppetInsecureSkipVerify bool
	puppetNodeName           string
	sensuAPIURL              string
	sensuAPIKey              string
	sensuCACert              string
}

const (
	defaultAPIPath = "pdb/query/v4/nodes"
)

var (
	handler = Handler{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-puppet-handler",
			Short:    "Deregister Sensu entities without an associated Puppet node",
			Keyspace: "sensu.io/plugins/sensu-puppet-handler/config",
		},
	}

	options = []sensu.ConfigOption{
		&sensu.PluginConfigOption[string]{
			Path:      "endpoint",
			Env:       "PUPPET_ENDPOINT",
			Argument:  "endpoint",
			Shorthand: "e",
			Usage:     "the PuppetDB API endpoint (URL). If an API path is not specified, /pdb/query/v4/nodes/ will be used",
			Value:     &handler.endpoint,
		},
		&sensu.PluginConfigOption[string]{
			Path:     "cert",
			Env:      "PUPPET_CERT",
			Argument: "cert",
			Usage:    "path to the SSL certificate PEM file signed by your site's Puppet CA",
			Value:    &handler.puppetCert,
		},
		&sensu.PluginConfigOption[string]{
			Path:     "key",
			Env:      "PUPPET_KEY",
			Argument: "key",
			Usage:    "path to the private key PEM file for that certificate",
			Value:    &handler.puppetKey,
		},
		&sensu.PluginConfigOption[string]{
			Path:     "ca-cert",
			Env:      "PUPPET_CA_CERT",
			Argument: "ca-cert",
			Usage:    "path to the site's Puppet CA certificate PEM file",
			Value:    &handler.puppetCACert,
		},
		&sensu.PluginConfigOption[bool]{
			Path:     "insecure-skip-tls-verify",
			Env:      "PUPPET_INSECURE_SKIP_TLS_VERIFY",
			Argument: "insecure-skip-tls-verify",
			Usage:    "skip TLS verification for Puppet and sensu-backend",
			Value:    &handler.puppetInsecureSkipVerify,
		},
		&sensu.PluginConfigOption[string]{
			Path:     "node-name",
			Env:      "PUPPET_NODE_NAME",
			Argument: "node-name",
			Usage:    "node name to use for the entity when querying PuppetDB",
			Value:    &handler.puppetNodeName,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "sensu-api-url",
			Env:       "SENSU_API_URL",
			Argument:  "sensu-api-url",
			Shorthand: "u",
			Default:   "http://localhost:8080",
			Usage:     "The Sensu API URL",
			Value:     &handler.sensuAPIURL,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "sensu-api-key",
			Env:       "SENSU_API_KEY",
			Argument:  "sensu-api-key",
			Shorthand: "a",
			Usage:     "The Sensu API key",
			Value:     &handler.sensuAPIKey,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "sensu-ca-cert",
			Env:       "SENSU_CA_CERT",
			Argument:  "sensu-ca-cert",
			Shorthand: "c",
			Usage:     "The Sensu Go CA Certificate",
			Value:     &handler.sensuCACert,
		},
	}
)

func main() {
	handler := sensu.NewGoHandler(&handler.PluginConfig, options, validate, executeHandler)
	handler.Execute()
}

func validate(event *corev2.Event) error {
	// Make sure we have a valid event
	if event.Check == nil || event.Entity == nil {
		return errors.New("invalid event")
	}

	// Make sure all required options are provided
	if len(handler.endpoint) == 0 {
		return errors.New("the PuppetDB API endpoint is required")
	}
	if len(handler.puppetCert) == 0 {
		return errors.New("the path to the SSL certificate is required")
	}
	if len(handler.puppetKey) == 0 {
		return errors.New("the path to the private key is required")
	}
	if len(handler.sensuAPIURL) == 0 {
		return errors.New("the Sensu API URL is required")
	}
	if len(handler.sensuAPIKey) == 0 {
		return errors.New("the Sensu API key is required")
	}

	// Make sure the PuppetDB endpoint URL is valid
	u, err := url.Parse(handler.endpoint)
	if err != nil {
		return fmt.Errorf("invalid PuppetDB API endpoint URL: %s", err)
	}
	if u.Scheme == "" {
		u.Host = "https://"
	}
	if u.Host == "" {
		return errors.New("invalid PuppetDB API endpoint URL")
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = path.Join(u.Path, defaultAPIPath)
	}
	handler.endpoint = u.String()

	// Make sure the Sensu API URL is valid
	u, err = url.Parse(handler.sensuAPIURL)
	if err != nil {
		return fmt.Errorf("invalid Sensu API URL: %s", err)
	}
	if u.Scheme == "" {
		return errors.New("invalid Sensu API URL, missing scheme")
	}
	if u.Host == "" {
		return errors.New("invalid Sensu API URL, missing host")
	}

	return nil
}

func executeHandler(event *corev2.Event) error {
	if event.Check.Name != "keepalive" {
		log.Print("received non-keepalive event, not checking for puppet node")
		return nil
	}

	puppetClient, err := puppetHTTPClient()
	if err != nil {
		return err
	}

	exists, err := puppetNodeExists(puppetClient, event)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return deregisterEntity(event)
}

// puppetHTTPClient configures an HTTP client for PuppetDB
func puppetHTTPClient() (*http.Client, error) {
	// Load the public/private key pair
	cert, err := tls.LoadX509KeyPair(handler.puppetCert, handler.puppetKey)
	if err != nil {
		return nil, fmt.Errorf("could not read the certificate/key: %s", err)
	}

	// Load the CA certificate
	caCert, err := ioutil.ReadFile(handler.puppetCACert)
	if err != nil {
		return nil, fmt.Errorf("could not read the CA certificate: %s", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup the HTTPS client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: handler.puppetInsecureSkipVerify,
	}
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}

	return client, nil
}

// puppetNodeExists returns whether a given node exists in Puppet and any error
// encountered. The Puppet node name defaults to the entity name but can be
// overriden through the entity label "puppet_node_name"
func puppetNodeExists(client *http.Client, event *corev2.Event) (bool, error) {
	// Determine the Puppet node name via the annotations and fallback to the
	// entity name
	name := handler.puppetNodeName
	if handler.puppetNodeName == "" {
		name = event.Entity.Name
	}

	// Get the puppet node
	endpoint := strings.TrimRight(handler.endpoint, "/")
	endpoint = fmt.Sprintf("%s/%s", endpoint, name)
	resp, err := client.Get(endpoint)
	if err != nil {
		log.Printf("error getting puppet node: %s", err)
		return false, err
	}
	defer resp.Body.Close()

	// Determine if the node exists
	if resp.StatusCode == http.StatusOK {
		var info map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			log.Printf("puppet node returned invalid response: %s", err)
			return false, err
		}
		nodeInfo := make(map[string]interface{})
		timeDeactivated := nodeInfo["deactivated"]

		log.Printf("puppet node %q exists, checking if deactivated", name)
		if timeDeactivated != nil {
			return false, nil
		}
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		log.Printf("puppet node %q does not exist", name)
		return false, nil
	}

	return false, fmt.Errorf("unexpected HTTP status %s while querying PuppetDB", http.StatusText(resp.StatusCode))
}

func deregisterEntity(event *corev2.Event) error {
	// First authenticate against the Sensu API
	config := httpclient.CoreClientConfig{
		URL:    handler.sensuAPIURL,
		APIKey: handler.sensuAPIKey,
	}
	if handler.sensuCACert != "" {
		pemCert, err := ioutil.ReadFile(handler.sensuCACert)
		if err != nil {
			return fmt.Errorf("unable to load sensu-ca-cert: %s", err)
		}

		block, _ := pem.Decode([]byte(pemCert))
		if block == nil {
			return errors.New("failed to decode sensu-ca-cert PEM")
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("invalid sensu-ca-cert: %s", err)
		}
		config.CACert = cert

	}
	if handler.puppetInsecureSkipVerify {
		config.InsecureSkipVerify = true
	}
	client := httpclient.NewCoreClient(config)
	request, err := httpclient.NewResourceRequest("core/v2", "Entity", event.Entity.Namespace, event.Entity.Name)
	if err != nil {
		return err
	}

	// Delete the Sensu entity
	log.Printf("deleting entity (%s/%s)\n", event.Entity.Namespace, event.Entity.Name)
	if _, err := client.DeleteResource(context.Background(), request); err != nil {
		if httperr, ok := err.(httpclient.HTTPError); ok {
			if httperr.StatusCode < 500 {
				log.Printf("entity already deleted (%s/%s)", event.Entity.Namespace, event.Entity.Name)
				return nil
			}
		}
		return err
	}

	return nil
}
