package smepay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	// requestTimeout is the fixed maximum per request; deliberately not configurable.
	requestTimeout = 60 * time.Second
	// rawTextLimit is the longest slice of a non-JSON reply kept in Error.Raw.
	rawTextLimit = 500
	// maxReplyBytes caps how much of a reply is read.
	maxReplyBytes = 10 << 20
)

// transport POSTs JSON to the SMEPay API.
type transport struct {
	baseURL string
	http    *http.Client
}

func newTransport(baseURL string) *transport {
	return &transport{baseURL: baseURL, http: &http.Client{Timeout: requestTimeout}}
}

// post sends body as JSON to path and decodes a 2xx reply into out.
// Every failure is returned as *Error.
func (t *transport) post(ctx context.Context, path string, body any, token string, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return &Error{Code: "invalid_request", Message: fmt.Sprintf("could not encode request for %s: %v", path, err), cause: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return &Error{Code: "network_error", Message: fmt.Sprintf("could not build request for %s: %v", path, err), cause: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "smepay-go/"+Version)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return transportError(ctx, path, err)
	}
	defer resp.Body.Close()

	reply, err := io.ReadAll(io.LimitReader(resp.Body, maxReplyBytes))
	if err != nil {
		return transportError(ctx, path, err)
	}

	if !json.Valid(reply) {
		raw := reply
		if len(raw) > rawTextLimit {
			raw = raw[:rawTextLimit]
		}
		return &Error{
			Code:       "invalid_response",
			Message:    fmt.Sprintf("SMEPay returned a non-JSON reply (HTTP %d) for %s", resp.StatusCode, path),
			StatusCode: resp.StatusCode,
			Raw:        raw,
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiError(resp.StatusCode, reply)
	}

	if out != nil {
		if err := json.Unmarshal(reply, out); err != nil {
			return &Error{
				Code:       "invalid_response",
				Message:    fmt.Sprintf("could not decode SMEPay reply for %s: %v", path, err),
				StatusCode: resp.StatusCode,
				Raw:        redact(reply),
				cause:      err,
			}
		}
	}
	return nil
}

// transportError maps a failed send/read to a timeout or network_error *Error.
func transportError(ctx context.Context, path string, err error) *Error {
	var netErr net.Error
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return &Error{Code: "timeout", Message: fmt.Sprintf("request to %s timed out", path), cause: err}
	}
	return &Error{Code: "network_error", Message: fmt.Sprintf("could not reach SMEPay (%s): %v", path, err), cause: err}
}
