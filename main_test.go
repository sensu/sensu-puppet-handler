package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	corev2 "github.com/sensu/core/v2"
)

func Test_validate(t *testing.T) {
	event := corev2.FixtureEvent("foo", "bar")

	tests := []struct {
		name         string
		testHandler  Handler
		event        *corev2.Event
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
			event:       &corev2.Event{},
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

func Test_puppetNodeExists(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
		wantErr    bool
	}{
		{
			name:       "node exists",
			statusCode: http.StatusOK,
			want:       true,
		},
		{
			name:       "node does not exist",
			statusCode: http.StatusNotFound,
			want:       false,
		},
		{
			name:       "unexpected status code",
			statusCode: http.StatusInternalServerError,
			want:       false,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"deactivated": time.Now().Unix()})
				}
			}))
			defer ts.Close()
			handler.endpoint = ts.URL

			event := corev2.FixtureEvent("foo", "check-cpu")
			got, err := puppetNodeExists(ts.Client(), event)
			if (err != nil) != tt.wantErr {
				t.Errorf("puppetNodeExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("puppetNodeExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deregisterEntity(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "entity deleted",
			statusCode: http.StatusNoContent,
		},
		{
			name:       "entity not deleted",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "unexpected status code",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer ts.Close()
			handler.sensuAPIURL = ts.URL

			event := corev2.FixtureEvent("foo", "check-cpu")
			if err := deregisterEntity(event); (err != nil) != tt.wantErr {
				t.Errorf("deregisterEntity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
