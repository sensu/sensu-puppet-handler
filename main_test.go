package main

import (
	"testing"

	"github.com/sensu/sensu-go/types"
)

func TestMain(t *testing.T) {
}

func Test_validate(t *testing.T) {
	event := types.FixtureEvent("foo", "bar")

	tests := []struct {
		name         string
		testHandler  Handler
		event        *types.Event
		wantEndpoint string
		wantErr      bool
	}{
		{
			name:        "required endpoint",
			testHandler: Handler{},
			event:       event,
			wantErr:     true,
		},
		{
			name: "required certificate",
			testHandler: Handler{
				endpoint: "http://127.0.0.1",
			},
			event:   event,
			wantErr: true,
		},
		{
			name: "required private key",
			testHandler: Handler{
				endpoint:   "http://127.0.0.1",
				puppetCert: "certificate.pem",
			},
			event:   event,
			wantErr: true,
		},
		{
			name: "required CA certificate",
			testHandler: Handler{
				endpoint:   "http://127.0.0.1",
				puppetCert: "cert.pem",
				puppetKey:  "key.pem",
			},
			event:   event,
			wantErr: true,
		},
		{
			name: "required Sensu API URL",
			testHandler: Handler{
				endpoint:     "http://127.0.0.1",
				puppetCert:   "cert.pem",
				puppetKey:    "key.pem",
				puppetCACert: "ca.pem",
			},
			event:   event,
			wantErr: true,
		},
		{
			name: "required Sensu API key",
			testHandler: Handler{
				endpoint:     "http://127.0.0.1",
				puppetCert:   "cert.pem",
				puppetKey:    "key.pem",
				puppetCACert: "ca.pem",
				sensuAPIURL:  "http://localhost:8080",
			},
			event:   event,
			wantErr: true,
		},
		{
			name: "all required options are passed",
			testHandler: Handler{
				endpoint:     "http://127.0.0.1",
				puppetCert:   "cert.pem",
				puppetKey:    "key.pem",
				puppetCACert: "ca.pem",
				sensuAPIURL:  "http://localhost:8080",
				sensuAPIKey:  "xxxxxxxxxx",
			},
			event:   event,
			wantErr: false,
		},
		{
			name: "valid endpoint is required",
			testHandler: Handler{
				endpoint:     "foo",
				puppetCert:   "cert.pem",
				puppetKey:    "key.pem",
				puppetCACert: "ca.pem",
			},
			event:   event,
			wantErr: true,
		},
		{
			name: "default API path is appended if missing",
			testHandler: Handler{
				endpoint:     "http://127.0.0.1/",
				puppetCert:   "cert.pem",
				puppetKey:    "key.pem",
				puppetCACert: "ca.pem",
				sensuAPIURL:  "http://localhost:8080",
				sensuAPIKey:  "xxxxxxxxxx",
			},
			event:        event,
			wantErr:      false,
			wantEndpoint: "http://127.0.0.1/pdb/query/v4/nodes",
		},
		{
			name:        "valid event is required",
			testHandler: Handler{},
			event:       &types.Event{},
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = tt.testHandler
			if err := validate(tt.event); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantEndpoint != "" && tt.wantEndpoint != handler.endpoint {
				t.Errorf("validate() endpoint = %v, want %v", handler.endpoint, tt.wantEndpoint)
			}
		})
	}
}
