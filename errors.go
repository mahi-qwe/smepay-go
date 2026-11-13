package smepay

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Error is returned for every failed SDK call. Check it with errors.As:
//
//	var e *smepay.Error
//	if errors.As(err, &e) && e.Code == "duplicate_order_id" { ... }
//
// API errors: Code is the API's "error" field, StatusCode the HTTP status and
// Raw the API reply body. SDK-side errors use the codes "invalid_config",
// "network_error", "timeout" and "invalid_response"; StatusCode is 0 when no
// reply was received.
type Error struct {
	Code       string
	Message    string
	StatusCode int
	Raw        []byte

	cause error
}

func (e *Error) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("smepay: %s (HTTP %d): %s", e.Code, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("smepay: %s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause (e.g. a network or context error), if any.
func (e *Error) Unwrap() error { return e.cause }

// sensitiveKeys must never appear in an Error, even if an API reply echoes them.
var sensitiveKeys = map[string]bool{
	"access_token":  true,
	"client_secret": true,
	"authorization": true,
}

// redact replaces sensitive top-level values in a JSON object body.
// Non-object bodies are returned unchanged.
func redact(body []byte) []byte {
	var obj map[string]json.RawMessage
	if json.Unmarshal(body, &obj) != nil {
		return body
	}
	changed := false
	for k := range obj {
		if sensitiveKeys[strings.ToLower(k)] {
			obj[k] = json.RawMessage(`"[redacted]"`)
			changed = true
		}
	}
	if !changed {
		return body
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return nil
	}
	return out
}

// apiError builds an Error from an API error reply: {error, message} or {error, errors}.
func apiError(status int, body []byte) *Error {
	var reply struct {
		Error   string          `json:"error"`
		Message string          `json:"message"`
		Errors  json.RawMessage `json:"errors"`
	}
	_ = json.Unmarshal(body, &reply)

	code := reply.Error
	if code == "" {
		code = "api_error"
	}

	message := fmt.Sprintf("SMEPay API error (HTTP %d)", status)
	switch {
	case reply.Message != "":
		message = reply.Message
	case len(reply.Errors) > 0 && string(reply.Errors) != "null":
		var s string
		if json.Unmarshal(reply.Errors, &s) == nil {
			if s != "" {
				message = s
			}
		} else {
			message = string(reply.Errors)
		}
	}

	return &Error{Code: code, Message: message, StatusCode: status, Raw: redact(body)}
}
