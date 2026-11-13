# smepay-go

Official Go SDK for the SMEPay External API. Accept UPI payments in three calls; login, tokens and errors are handled for you.

Go 1.21+ · standard library only · safe for concurrent use.

```bash
go get github.com/mahi-qwe/smepay-go
```

## Usage

```go
import smepay "github.com/mahi-qwe/smepay-go"

client, err := smepay.New(smepay.Config{
	ClientID:     os.Getenv("SMEPAY_CLIENT_ID"), // from the SMEPay dashboard
	ClientSecret: os.Getenv("SMEPAY_CLIENT_SECRET"),
	Environment:  smepay.Test, // smepay.Test (default) or smepay.Live
})
if err != nil { /* invalid_config */ }

// 1. Create an order
order, err := client.Payment.Create(ctx, smepay.CreateParams{
	Amount:          "100.50",   // string, in rupees
	OrderID:         "ORD-1001", // your reference, unique per order
	CallbackURL:     "https://yoursite.com/smepay/callback",
	CustomerDetails: smepay.CustomerDetails{Name: "Asha", Mobile: "9876543210"}, // email or mobile required
})

// 2. Start the payment: show payment.QRCode (base64 PNG) or payment.Intents (UPI app links)
payment, err := client.Payment.Initiate(ctx, smepay.InitiateParams{Slug: order.OrderSlug})

// 3. Check the status (or use OrderID: order.OrderID)
status, err := client.Payment.GetStatus(ctx, smepay.GetStatusParams{ExternalReferenceID: "ORD-1001"})
if status.PaymentStatus == smepay.StatusSuccess { /* paid; status.UTR has the bank reference */ }
```

Create one `Client` and reuse it. Every call takes a `context.Context`; each request waits at most 60 seconds.

| Method | Returns |
|---|---|
| `Payment.Create` | `OrderSlug`, `OrderID` (SMEPay's), `RefID` (yours), `PaymentURL`, `PaymentStatus` |
| `Payment.Initiate` | `PaymentLink`, `QRCode`, `Intents`, `ExpiresAt`, `PaymentStatus` |
| `Payment.GetStatus` | `PaymentStatus`, `Amount`, `CreatedAt`, `ProcessedAt` (nil until processed), `UTR` |

`Initiate` can be called again if a payment failed or expired. Alternatively, skip `Initiate` and redirect the customer to `PaymentURL` (the hosted page); use one or the other per order.

**Statuses** (`smepay.Status…` constants): `CREATED` → `INITIATED` / `PENDING` → `SUCCESS` or `FAILED` / `EXPIRED`; also `SETTLED`, `REFUNDED`, `REFUND_PENDING`. Test-mode businesses get `TEST_SUCCESS`, `TEST_FAILED`, `TEST_PENDING`.

## Errors

Every failure is a `*smepay.Error` with `Code`, `Message`, `StatusCode` (0 if no reply) and `Raw` (API reply).

```go
var e *smepay.Error
if errors.As(err, &e) && e.Code == "duplicate_order_id" { /* ... */ }
```

| Code | Meaning |
|---|---|
| `invalid_config` | Missing `ClientID` / `ClientSecret`, or bad `Environment` |
| `invalid_client_id`, `invalid_credentials` | Wrong credentials |
| `environment_mismatch` | `Environment` doesn't match your business's mode |
| `ip_not_whitelisted` | Live: your server IP isn't whitelisted |
| `insufficient_wallet_balance`, `wallet_validation_failed` | Live: wallet balance too low |
| `invalid_request`, `validation_error` | Missing or invalid field |
| `amount_below_minimum`, `amount_exceeds_maximum` | Amount outside your allowed range |
| `duplicate_order_id` | `OrderID` already used |
| `transaction_not_found` | No such order for your business |
| `payment_already_processed` | `Initiate` on an order already paid / in progress |
| `network_error`, `timeout`, `invalid_response` | Couldn't reach SMEPay / no reply in time / unreadable reply |

Calls are never retried automatically, so a failed `Create` is never sent twice. A cancelled context gives `network_error` with `errors.Is(err, context.Canceled)` true.

## Test vs live

| | `smepay.Test` | `smepay.Live` |
|---|---|---|
| Server | `staging.smepay.in` | `extranet.smepay.in` |
| Money / fees | None | Real; fees from your SMEPay wallet |
| Requirements | — | Server IP whitelisted, wallet ≥ ₹10 |

## Webhooks

When a payment finishes, SMEPay POSTs `ref_id`, `transaction_id`, `status`, `amount`, `created_at`, `processed_at` and `utr` to your `CallbackURL`. Confirm with `Payment.GetStatus` before marking an order paid.

## Security

Use the SDK only on your server. Keep `ClientSecret` in an environment variable, never in code. The SDK never logs your secret or token, and printing a `Client` doesn't show them.

## Example

[`examples/full-flow`](examples/full-flow/main.go): put `SMEPAY_CLIENT_ID` / `SMEPAY_CLIENT_SECRET` in `.env`, then `set -a; source .env; set +a; go run ./examples/full-flow`.

## License

MIT
