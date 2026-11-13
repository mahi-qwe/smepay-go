// Full flow: Create → Initiate → GetStatus.
//
// Credentials come from environment variables (keep them in a .env file, never commit it):
//
//	SMEPAY_CLIENT_ID=...
//	SMEPAY_CLIENT_SECRET=...
//	SMEPAY_ENVIRONMENT=test          # optional: "test" (default) or "live"
//	SMEPAY_CALLBACK_URL=https://...  # optional: where the result webhook is sent
//
// Run from the repo root:
//
//	set -a; source .env; set +a
//	go run ./examples/full-flow
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	smepay "github.com/mahi-qwe/smepay-go"
)

func main() {
	if err := run(); err != nil {
		var e *smepay.Error
		if errors.As(err, &e) {
			fmt.Printf("\nsmepay error [%s] (HTTP %d): %s\n", e.Code, e.StatusCode, e.Message)
		} else {
			fmt.Println("\nerror:", err)
		}
		os.Exit(1)
	}
}

func run() error {
	client, err := smepay.New(smepay.Config{
		ClientID:     os.Getenv("SMEPAY_CLIENT_ID"),
		ClientSecret: os.Getenv("SMEPAY_CLIENT_SECRET"),
		Environment:  smepay.Environment(os.Getenv("SMEPAY_ENVIRONMENT")), // empty = test
	})
	if err != nil {
		return err
	}
	fmt.Println("Environment:", client.Environment())

	ctx := context.Background()
	callbackURL := os.Getenv("SMEPAY_CALLBACK_URL")
	if callbackURL == "" {
		callbackURL = "https://example.com/smepay/callback"
	}

	// 1. Create the order. OrderID is your own reference and must be unique.
	order, err := client.Payment.Create(ctx, smepay.CreateParams{
		Amount:          "1.00",
		OrderID:         fmt.Sprintf("DEMO-%d", time.Now().UnixMilli()),
		CallbackURL:     callbackURL,
		CustomerDetails: smepay.CustomerDetails{Name: "Demo Customer", Mobile: "9999999999"},
	})
	if err != nil {
		return err
	}
	fmt.Println("\n1. Create")
	fmt.Println("   order_id (SMEPay):", order.OrderID)
	fmt.Println("   ref_id (yours):   ", order.RefID)
	fmt.Println("   order_slug:       ", order.OrderSlug)
	fmt.Println("   payment_status:   ", order.PaymentStatus)
	fmt.Println("   payment_url:      ", order.PaymentURL)

	// 2. Start the payment: show the QR code or UPI app links to the customer.
	payment, err := client.Payment.Initiate(ctx, smepay.InitiateParams{Slug: order.OrderSlug})
	if err != nil {
		return err
	}
	apps := make([]string, 0, len(payment.Intents))
	for app := range payment.Intents {
		apps = append(apps, app)
	}
	sort.Strings(apps)
	fmt.Println("\n2. Initiate")
	fmt.Println("   payment_status:   ", payment.PaymentStatus)
	fmt.Println("   payment_link:     ", payment.PaymentLink)
	fmt.Printf("   qr_code:           base64, %d chars\n", len(payment.QRCode))
	fmt.Println("   intents:          ", strings.Join(apps, ", "))
	fmt.Println("   expires_at:       ", time.Unix(payment.ExpiresAt, 0).Format(time.RFC3339))

	// 3. Check the status by your reference (or by SMEPay's OrderID).
	status, err := client.Payment.GetStatus(ctx, smepay.GetStatusParams{ExternalReferenceID: order.RefID})
	if err != nil {
		return err
	}
	fmt.Println("\n3. GetStatus")
	fmt.Println("   payment_status:   ", status.PaymentStatus)
	fmt.Println("   amount:           ", status.Amount)
	if status.ProcessedAt != nil {
		fmt.Println("   processed_at:     ", time.Unix(*status.ProcessedAt, 0).Format(time.RFC3339))
	}
	if status.UTR != "" {
		fmt.Println("   utr:              ", status.UTR)
	}
	return nil
}
