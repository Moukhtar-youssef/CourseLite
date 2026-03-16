// Package payment contain the code and the main interface for the payment
// provider
package payment

type Provider interface {
	// CreateCheckout(courseID , userID , price string) (CheckoutURL string,
	// 	err error)
	// HandleWebhook(r *http.Request) (Event,error)
	// GetSubscriptionStatus(userID string)(status,error)
}
