package smepay

import "context"

// PaymentStatus is a payment status returned by the API. Compare against the
// constants below; any new value the API adds is still accepted.
type PaymentStatus string

const (
	StatusCreated       PaymentStatus = "CREATED"
	StatusInitiated     PaymentStatus = "INITIATED"
	StatusPending       PaymentStatus = "PENDING"
	StatusSuccess       PaymentStatus = "SUCCESS"
	StatusFailed        PaymentStatus = "FAILED"
	StatusExpired       PaymentStatus = "EXPIRED"
	StatusSettled       PaymentStatus = "SETTLED"
	StatusRefunded      PaymentStatus = "REFUNDED"
	StatusRefundPending PaymentStatus = "REFUND_PENDING"
	// Test-mode businesses get these instead of SUCCESS / FAILED / PENDING.
	StatusTestSuccess PaymentStatus = "TEST_SUCCESS"
	StatusTestFailed  PaymentStatus = "TEST_FAILED"
	StatusTestPending PaymentStatus = "TEST_PENDING"
)

// CustomerDetails of the payer; at least Email or Mobile is required.
type CustomerDetails struct {
	Name   string `json:"name,omitempty"`
	Email  string `json:"email,omitempty"`
	Mobile string `json:"mobile,omitempty"`
}

// CreateParams for Payment.Create (POST /api/external/order/create).
type CreateParams struct {
	Amount          string          `json:"amount"`       // in rupees, as a string, e.g. "100.50"
	OrderID         string          `json:"order_id"`     // your own reference; unique for your business
	CallbackURL     string          `json:"callback_url"` // receives the payment result webhook
	CustomerDetails CustomerDetails `json:"customer_details"`
}

// CreateResponse from Payment.Create.
type CreateResponse struct {
	Status          bool            `json:"status"`
	OrderSlug       string          `json:"order_slug"` // pass to Payment.Initiate
	OrderID         string          `json:"order_id"`   // SMEPay's order ID
	Amount          float64         `json:"amount"`
	PaymentStatus   PaymentStatus   `json:"payment_status"`
	PaymentURL      string          `json:"payment_url"` // hosted SMEPay payment page
	RefID           string          `json:"ref_id"`      // your OrderID, echoed back
	CustomerDetails CustomerDetails `json:"customer_details"`
	Message         string          `json:"message"`
	CallbackURL     string          `json:"callback_url,omitempty"`
}

// InitiateParams for Payment.Initiate (POST /api/external/order/initiate).
type InitiateParams struct {
	Slug string `json:"slug"` // OrderSlug from Payment.Create
}

// InitiateResponse from Payment.Initiate.
type InitiateResponse struct {
	Status              bool              `json:"status"`
	OrderID             string            `json:"order_id"`              // SMEPay's order ID
	ExternalReferenceID string            `json:"external_reference_id"` // your OrderID
	PaymentLink         string            `json:"payment_link"`          // UPI payment link
	TransactionID       string            `json:"transaction_id"`
	QRCode              string            `json:"qr_code"` // base64-encoded PNG
	PaymentStatus       PaymentStatus     `json:"payment_status"`
	ExpiresAt           int64             `json:"expires_at"` // Unix seconds
	Intents             map[string]string `json:"intents"`    // UPI app key -> app payment link
	Message             string            `json:"message"`
}

// GetStatusParams for Payment.GetStatus (POST /api/external/order/status).
// Set OrderID (SMEPay's) or ExternalReferenceID (yours); ExternalReferenceID is used first if both are set.
type GetStatusParams struct {
	OrderID             string `json:"order_id,omitempty"`
	ExternalReferenceID string `json:"external_reference_id,omitempty"`
	CallbackURL         string `json:"callback_url,omitempty"` // saved only if the order has none yet
}

// GetStatusResponse from Payment.GetStatus.
type GetStatusResponse struct {
	Status              bool          `json:"status"`
	ExternalReferenceID string        `json:"external_reference_id"` // your OrderID
	OrderID             string        `json:"order_id"`              // SMEPay's order ID
	PaymentStatus       PaymentStatus `json:"payment_status"`
	Amount              float64       `json:"amount"`
	CreatedAt           int64         `json:"created_at"`    // Unix seconds
	ProcessedAt         *int64        `json:"processed_at"`  // Unix seconds; nil until processed
	UTR                 string        `json:"utr,omitempty"` // bank reference, when available
}

// PaymentService groups the payment calls: Create → Initiate → GetStatus.
type PaymentService struct {
	api *apiClient
}

// Create creates a payment order. Next step: Initiate with the returned OrderSlug.
func (s *PaymentService) Create(ctx context.Context, params CreateParams) (*CreateResponse, error) {
	var res CreateResponse
	if err := s.api.post(ctx, "/api/external/order/create", params, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Initiate starts the payment for an order and returns the UPI link, QR code and app intents.
func (s *PaymentService) Initiate(ctx context.Context, params InitiateParams) (*InitiateResponse, error) {
	var res InitiateResponse
	if err := s.api.post(ctx, "/api/external/order/initiate", params, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetStatus fetches the current payment status of an order.
func (s *PaymentService) GetStatus(ctx context.Context, params GetStatusParams) (*GetStatusResponse, error) {
	var res GetStatusResponse
	if err := s.api.post(ctx, "/api/external/order/status", params, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
