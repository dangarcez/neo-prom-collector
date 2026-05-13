package prometheus

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"neo_collector_go/internal/config"
)

const azurePrometheusScope = "https://prometheus.monitor.azure.com/.default"

type tokenCredential interface {
	GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error)
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(target config.PromTargetConfig) (*Client, error) {
	baseTransport := &http.Transport{}
	if !target.VerifyTLSEnabled() {
		baseTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	var transport http.RoundTripper = baseTransport
	if target.AzureAuth != nil {
		credential, err := newAzureManagedIdentityCredential(*target.AzureAuth)
		if err != nil {
			return nil, err
		}

		transport = newBearerTokenRoundTripper(transport, credential)
	}

	return &Client{
		baseURL: strings.TrimRight(target.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout:   time.Duration(target.TimeoutSeconds) * time.Second,
			Transport: transport,
		},
	}, nil
}

func newAzureManagedIdentityCredential(auth config.AzureAuthConfig) (tokenCredential, error) {
	id := strings.TrimSpace(auth.ManagedIdentityID)
	if id == "" {
		return azidentity.NewManagedIdentityCredential(nil)
	}

	options := azidentity.ManagedIdentityCredentialOptions{}
	if strings.HasPrefix(strings.ToLower(id), "/subscriptions/") {
		options.ID = azidentity.ResourceID(id)
	} else {
		options.ID = azidentity.ClientID(id)
	}

	return azidentity.NewManagedIdentityCredential(&options)
}

type bearerTokenRoundTripper struct {
	base       http.RoundTripper
	credential tokenCredential
}

func newBearerTokenRoundTripper(base http.RoundTripper, credential tokenCredential) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}

	return bearerTokenRoundTripper{
		base:       base,
		credential: credential,
	}
}

func (t bearerTokenRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	token, err := t.credential.GetToken(request.Context(), policy.TokenRequestOptions{
		Scopes: []string{azurePrometheusScope},
	})
	if err != nil {
		return nil, fmt.Errorf("get azure prometheus token: %w", err)
	}

	authenticatedRequest := request.Clone(request.Context())
	authenticatedRequest.Header = request.Header.Clone()
	authenticatedRequest.Header.Set("Authorization", "Bearer "+token.Token)

	return t.base.RoundTrip(authenticatedRequest)
}
