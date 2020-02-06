package main

import (
	"testing"
)

func TestMain(t *testing.T) {
}

func Test_validate(t *testing.T) {
	tests := []struct {
		name         string
		testHandler  Handler
		wantEndpoint string
		wantErr      bool
	}{
		{
			name:        "required endpoint",
			testHandler: Handler{},
			wantErr:     true,
		},
		{
			name: "required keystore file",
			testHandler: Handler{
				endpoint: "http://127.0.0.1",
			},
			wantErr: true,
		},
		{
			name: "required keystore password",
			testHandler: Handler{
				endpoint:     "http://127.0.0.1",
				keystoreFile: "keystore.jks",
			},
			wantErr: true,
		},
		{
			name: "required truststore file",
			testHandler: Handler{
				endpoint:         "http://127.0.0.1",
				keystoreFile:     "keystore.jks",
				keystorePassword: "P@ssw0rd!",
			},
			wantErr: true,
		},
		{
			name: "required truststore password",
			testHandler: Handler{
				endpoint:         "http://127.0.0.1",
				keystoreFile:     "keystore.jks",
				keystorePassword: "P@ssw0rd!",
				truststoreFile:   "truststore.jks",
			},
			wantErr: true,
		},
		{
			name: "all required options are passed",
			testHandler: Handler{
				endpoint:           "http://127.0.0.1",
				keystoreFile:       "keystore.jks",
				keystorePassword:   "P@ssw0rd!",
				truststoreFile:     "truststore.jks",
				truststorePassword: "P@ssw0rd!",
			},
			wantErr: false,
		},
		{
			name: "valid endpoint is required",
			testHandler: Handler{
				endpoint:           "foo",
				keystoreFile:       "keystore.jks",
				keystorePassword:   "P@ssw0rd!",
				truststoreFile:     "truststore.jks",
				truststorePassword: "P@ssw0rd!",
			},
			wantErr: true,
		},
		{
			name: "default API path is appended if missing",
			testHandler: Handler{
				endpoint:           "http://127.0.0.1/",
				keystoreFile:       "keystore.jks",
				keystorePassword:   "P@ssw0rd!",
				truststoreFile:     "truststore.jks",
				truststorePassword: "P@ssw0rd!",
			},
			wantErr:      false,
			wantEndpoint: "http://127.0.0.1/pdb/query/v4/nodes",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = tt.testHandler
			if err := validate(nil); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantEndpoint != "" && tt.wantEndpoint != handler.endpoint {
				t.Errorf("validate() endpoint = %v, want %v", handler.endpoint, tt.wantEndpoint)
			}
		})
	}
}
