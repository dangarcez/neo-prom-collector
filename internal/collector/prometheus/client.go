package prometheus

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"neo_collector_go/internal/config"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(target config.PromTargetConfig) *Client {
	transport := &http.Transport{}
	if !target.VerifyTLSEnabled() {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	return &Client{
		baseURL: strings.TrimRight(target.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout:   time.Duration(target.TimeoutSeconds) * time.Second,
			Transport: transport,
		},
	}
}
