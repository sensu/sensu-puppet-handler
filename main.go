package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/sensu-community/sensu-plugin-sdk/httpclient"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
)

// Handler represents the sensu-puppet-handler plugin
type Handler struct {
	sensu.PluginConfig
	endpoint                 string
	puppetCert               string
	puppetKey                string
	puppetCACert             string
	puppetInsecureSkipVerify bool
	sensuAPIURL              string
	sensuAPIKey              string
	sensuCACert              string
}

const (
	defaultAPIPath      = "pdb/query/v4/nodes"
	labelPuppetNodeName = "puppet_node_name"
)

var (
	handler = Handler{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-puppet-handler",
			Short:    "Deregister Sensu entities without an associated Puppet node",
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
			Path:     "cert",
			Env:      "PUPPET_CERT",
			Argument: "cert",
			Usage:    "path to the SSL certificate PEM file signed by your site's Puppet CA",
			Value:    &handler.puppetCert,
		},
		&sensu.PluginConfigOption{
			Path:     "key",
			Env:      "PUPPET_KEY",
			Argument: "key",
			Usage:    "path to the private key PEM file for that certificate",
			Value:    &handler.puppetKey,
		},
		&sensu.PluginConfigOption{
			Path:     "cacert",
			Env:      "PUPPET_CACERT",
			Argument: "cacert",
			Usage:    "path to the site's Puppet CA certificate PEM file",
			Value:    &handler.puppetCACert,
		},
		&sensu.PluginConfigOption{
			Path:     "insecure-skip-tls-verify",
			Env:      "PUPPET_INSECURE_SKIP_TLS_VERIFY",
			Argument: "insecure-skip-tls-verify",
			Usage:    "skip SSL verification",
			Value:    &handler.puppetInsecureSkipVerify,
		},
		{
			Path:      "sensu-api-url",
			Env:       "SENSU_API_URL",
			Argument:  "sensu-api-url",
			Shorthand: "u",
			Default:   "http://localhost:8080",
			Usage:     "The Sensu API URL",
			Value:     &handler.sensuAPIURL,
		},
		{
			Path:      "sensu-api-key",
			Env:       "SENSU_API_KEY",
			Argument:  "sensu-api-key",
			Shorthand: "a",
			Usage:     "The Sensu API key",
			Value:     &handler.sensuAPIKey,
		},
		{
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

func validate(event *types.Event) error {
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
	if u.Scheme == "" || u.Host == "" {
		return errors.New("invalid PuppetDB API endpoint URL")
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = path.Join(u.Path, defaultAPIPath)
		handler.endpoint = u.String()
	}

	// Make sure the Sensu API URL is valid
	u, err = url.Parse(handler.sensuAPIURL)
	if err != nil {
		return fmt.Errorf("invalid Sensu API URL: %s", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return errors.New("invalid Sensu API URL")
	}

	return nil
}

func executeHandler(event *types.Event) error {
	if event.Check.Name != "keepalive" {
		log.Print("received non-keepalive event, not checking for puppet node")
		return nil
	}

	exists, err := puppetNodeExists(event)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return deregisterEntity(event)
}

// puppetNodeExists returns whether a given node exists in Puppet and any error
// encountered. The Puppet node name defaults to the entity name but can be
// overriden through the entity label "puppet_node_name"
func puppetNodeExists(event *types.Event) (bool, error) {
	// Determine the Puppet node name
	name := event.Entity.Name
	if event.Entity.Labels[labelPuppetNodeName] != "" {
		name = event.Entity.Labels[labelPuppetNodeName]
	}

	// Load the public/private key pair
	cert, err := tls.LoadX509KeyPair(handler.puppetCert, handler.puppetKey)
	if err != nil {
		return false, err
	}

	// Load the CA certificate
	caCert, err := ioutil.ReadFile(handler.puppetCACert)
	if err != nil {
		log.Println(err.Error())
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup the HTTPS client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: handler.puppetInsecureSkipVerify,
	}
	tlsConfig.BuildNameToCertificate()
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}

	// Get the puppet node
	endpoint := strings.TrimRight(handler.endpoint, "/")
	endpoint = fmt.Sprintf("%s/%s", endpoint, name)
	resp, err := client.Get(endpoint)
	if err != nil {
		return false, err
	}
	_ = resp.Body.Close()

	// Determine if the node exists
	if resp.StatusCode == http.StatusOK {
		log.Printf("puppet node %q exists", name)
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		log.Printf("puppet node %q does not exist", name)
		return false, nil
	}

	return false, fmt.Errorf("unexpected HTTP status %s while querying PuppetDB", http.StatusText(resp.StatusCode))
}

func deregisterEntity(event *types.Event) error {
	// First authenticate against the Sensu API
	config := httpclient.CoreClientConfig{
		URL:    handler.sensuAPIURL,
		APIKey: handler.sensuAPIKey,
	}
	if handler.sensuCACert != "" {
		asn1Data, err := ioutil.ReadFile(handler.sensuCACert)
		if err != nil {
			return fmt.Errorf("unable to load sensu-ca-cert: %s", err)
		}
		cert, err := x509.ParseCertificate(asn1Data)
		if err != nil {
			return fmt.Errorf("invalid sensu-ca-cert: %s", err)
		}
		config.CACert = cert

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
