package smepay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Environment selects the SMEPay server. It must match your business's mode on SMEPay.
type Environment string

const (
	// Test uses staging.smepay.in: no real money, no fees.
	Test Environment = "test"
	// Live uses extranet.smepay.in: real payments.
	Live Environment = "live"
)

var baseURLs = map[Environment]string{
	Test: "https://staging.smepay.in",
	Live: "https://extranet.smepay.in",
}

// Config holds the credentials from the SMEPay dashboard.
type Config struct {
	ClientID     string
	ClientSecret string // keep in an environment variable, never in code
	Environment  Environment
}

// Client calls the SMEPay External API. It is safe for concurrent use;
// create one and reuse it.
type Client struct {
	// Payment holds the payment calls: Create → Initiate → GetStatus.
	Payment *PaymentService

	environment Environment
	api         *apiClient
}

// New creates a Client. Environment defaults to Test.
// A missing ClientID/ClientSecret or unknown Environment returns *Error with code "invalid_config".
func New(cfg Config) (*Client, error) {
	clientID := strings.TrimSpace(cfg.ClientID)
	clientSecret := strings.TrimSpace(cfg.ClientSecret)
	env := cfg.Environment
	if env == "" {
		env = Test
	}

	if clientID == "" {
		return nil, invalidConfig("ClientID is required")
	}
	if clientSecret == "" {
		return nil, invalidConfig("ClientSecret is required")
	}
	baseURL, ok := baseURLs[env]
	if !ok {
		return nil, invalidConfig(fmt.Sprintf("Environment must be smepay.Test or smepay.Live, got %q", string(env)))
	}

	api := newAPIClient(baseURL, clientID, clientSecret)
	return &Client{Payment: &PaymentService{api: api}, environment: env, api: api}, nil
}

// Environment reports which SMEPay environment the client uses.
func (c *Client) Environment() Environment { return c.environment }

// String and GoString keep credentials out of logs and fmt output.
func (c *Client) String() string   { return fmt.Sprintf("smepay.Client{Environment: %s}", c.environment) }
func (c *Client) GoString() string { return c.String() }

func invalidConfig(message string) *Error {
	return &Error{Code: "invalid_config", Message: message}
}

// tokenErrorCodes are 401 codes meaning the token itself was rejected; a fresh login may fix these.
var tokenErrorCodes = map[string]bool{
	"invalid_token":          true,
	"invalid_external_token": true,
	"invalid_token_type":     true,
}

// apiClient makes authenticated POSTs with client_id added to the body.
type apiClient struct {
	transport *transport
	clientID  string
	auth      *tokenManager
}

func newAPIClient(baseURL, clientID, clientSecret string) *apiClient {
	t := newTransport(baseURL)
	return &apiClient{
		transport: t,
		clientID:  clientID,
		auth:      &tokenManager{transport: t, clientID: clientID, clientSecret: clientSecret},
	}
}

// post sends params (with client_id added) to path and decodes the reply into out.
// On a rejected token it logs in again once and repeats the call once.
func (a *apiClient) post(ctx context.Context, path string, params any, out any) error {
	body, err := a.withClientID(params)
	if err != nil {
		return err
	}

	token, err := a.auth.getToken(ctx)
	if err != nil {
		return err
	}
	err = a.transport.post(ctx, path, body, token, out)

	var e *Error
	if !errors.As(err, &e) || e.StatusCode != 401 || !tokenErrorCodes[e.Code] {
		return err
	}
	a.auth.invalidate(token)
	if token, err = a.auth.getToken(ctx); err != nil {
		return err
	}
	return a.transport.post(ctx, path, body, token, out)
}

// withClientID encodes params as a JSON object and sets client_id on it.
func (a *apiClient) withClientID(params any) (map[string]json.RawMessage, error) {
	body := map[string]json.RawMessage{}
	if params != nil {
		raw, err := json.Marshal(params)
		if err == nil {
			err = json.Unmarshal(raw, &body)
		}
		if err != nil {
			return nil, &Error{Code: "invalid_request", Message: fmt.Sprintf("could not encode request: %v", err), cause: err}
		}
	}
	id, _ := json.Marshal(a.clientID)
	body["client_id"] = id
	return body, nil
}
