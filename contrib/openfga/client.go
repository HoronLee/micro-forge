// Package openfga maps generated Servora configuration to the official OpenFGA
// SDK client. It intentionally does not wrap the SDK data plane.
package openfga

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	openfgaconfpb "github.com/Servora-Kit/servora/api/gen/go/servora/contrib/openfga/v1"
	fgaclient "github.com/openfga/go-sdk/client"
	fgacredentials "github.com/openfga/go-sdk/credentials"
)

// NewClient constructs and returns the official OpenFGA SDK client.
func NewClient(cfg *openfgaconfpb.OpenFGA) (*fgaclient.OpenFgaClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("openfga: config is nil")
	}
	if err := cfg.ApplyConf(); err != nil {
		return nil, fmt.Errorf("openfga: config: %w", err)
	}

	sdkConfig := &fgaclient.ClientConfiguration{
		ApiUrl:               cfg.GetApiUrl(),
		StoreId:              cfg.GetStoreId(),
		AuthorizationModelId: cfg.GetModelId(),
	}
	if token := cfg.GetApiToken(); token != "" {
		endpoint, err := url.ParseRequestURI(cfg.GetApiUrl())
		if err != nil || endpoint.User != nil || endpoint.Hostname() == "" || !strings.EqualFold(endpoint.Scheme, "https") {
			return nil, fmt.Errorf("openfga: api_token requires an https api_url with a valid hostname")
		}
		baseClient := http.DefaultClient
		if baseClient == nil {
			baseClient = new(http.Client)
		}
		httpClient := *baseClient
		httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
			return fmt.Errorf("openfga: redirects are disabled when api_token is configured")
		}
		sdkConfig.HTTPClient = &httpClient
		sdkConfig.Credentials = &fgacredentials.Credentials{
			Method: fgacredentials.CredentialsMethodApiToken,
			Config: &fgacredentials.Config{ApiToken: token},
		}
	}

	client, err := fgaclient.NewSdkClient(sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("openfga: construct SDK client: %w", err)
	}
	return client, nil
}
