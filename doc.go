// Package smepay is the official Go SDK for the SMEPay External API.
//
// It wraps the three payment calls (create, initiate, get status) and
// handles authentication, token reuse and errors for you.
//
//	client, err := smepay.New(smepay.Config{
//		ClientID:     os.Getenv("SMEPAY_CLIENT_ID"),
//		ClientSecret: os.Getenv("SMEPAY_CLIENT_SECRET"),
//		Environment:  smepay.Test, // or smepay.Live
//	})
//
// Use the SDK only on your server; never expose the client secret.
package smepay
