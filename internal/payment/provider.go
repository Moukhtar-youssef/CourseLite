// Package payment contain the code and the main interface for the payment
// provider
package payment

type Provider interface {
	// CreateCheckout(courseID, userID, price string) (checkoutURL string, err error)
	// HandleWebhook(r *http.Request) (Event, error)
	// GetSubscriptionStatus(userID string) (Status, error)
}
