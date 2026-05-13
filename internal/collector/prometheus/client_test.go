package prometheus

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"neo_collector_go/internal/config"
)

func TestQueryWithoutAzureAuthDoesNotSetAuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("expected no authorization header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	}))
	defer server.Close()

	client, err := NewClient(config.PromTargetConfig{
		Name:           "prom",
		BaseURL:        server.URL,
		TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if _, err := client.Query(context.Background(), "up"); err != nil {
		t.Fatalf("expected query to succeed, got error: %v", err)
	}
}

func TestQueryWithBearerTokenRoundTripperAddsAuthorizationHeader(t *testing.T) {
	credential := &fakeTokenCredential{token: "test-token"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("expected bearer token header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: newBearerTokenRoundTripper(http.DefaultTransport, credential),
		},
	}

	if _, err := client.Query(context.Background(), "up"); err != nil {
		t.Fatalf("expected query to succeed, got error: %v", err)
	}
	if credential.calls != 1 {
		t.Fatalf("expected one token request, got %d", credential.calls)
	}
	if len(credential.scopes) != 1 || credential.scopes[0] != azurePrometheusScope {
		t.Fatalf("expected azure prometheus scope, got %#v", credential.scopes)
	}
}

func TestQueryReturnsTokenError(t *testing.T) {
	credential := &fakeTokenCredential{err: errors.New("identity unavailable")}
	client := &Client{
		baseURL: "http://prometheus.example",
		httpClient: &http.Client{
			Transport: newBearerTokenRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("base transport should not be called when token acquisition fails")
				return nil, nil
			}), credential),
		},
	}

	_, err := client.Query(context.Background(), "up")
	if err == nil {
		t.Fatal("expected query to fail")
	}
	if !strings.Contains(err.Error(), "get azure prometheus token") {
		t.Fatalf("expected token error context, got: %v", err)
	}
}

type fakeTokenCredential struct {
	token  string
	err    error
	calls  int
	scopes []string
}

func (c *fakeTokenCredential) GetToken(_ context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.calls++
	c.scopes = append([]string(nil), opts.Scopes...)
	if c.err != nil {
		return azcore.AccessToken{}, c.err
	}

	return azcore.AccessToken{Token: c.token}, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
