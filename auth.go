package smepay

import (
	"context"
	"sync"
	"time"
)

const (
	authPath = "/api/external/auth"
	// refreshMargin: fetch a new token when less than this much lifetime is left.
	refreshMargin = 60 * time.Second
)

// tokenManager logs in with client_id/client_secret and caches the access token.
// The mutex is held during login, so concurrent callers share a single login.
type tokenManager struct {
	transport    *transport
	clientID     string
	clientSecret string

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

type authResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Environment string `json:"environment"`
}

// getToken returns the cached token, or logs in if there is none or it is about to expire.
func (m *tokenManager) getToken(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.token != "" && time.Now().Before(m.expiresAt.Add(-refreshMargin)) {
		return m.token, nil
	}

	// Measured before the call so network time counts against the token's lifetime.
	startedAt := time.Now()
	var res authResponse
	body := map[string]string{"client_id": m.clientID, "client_secret": m.clientSecret}
	if err := m.transport.post(ctx, authPath, body, "", &res); err != nil {
		return "", err
	}
	if res.AccessToken == "" || res.ExpiresIn <= 0 {
		return "", &Error{
			Code:       "invalid_response",
			Message:    "SMEPay auth reply did not contain an access token",
			StatusCode: 200,
		}
	}

	m.token = res.AccessToken
	m.expiresAt = startedAt.Add(time.Duration(res.ExpiresIn) * time.Second)
	return m.token, nil
}

// invalidate drops the cached token if it is the one the server just rejected.
func (m *tokenManager) invalidate(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.token == token {
		m.token = ""
		m.expiresAt = time.Time{}
	}
}
