// Package payment provides the interface for payment providers.
package payment

// Provider defines the interface for payment providers.
type Provider interface {
	// CreateCheckout creates a checkout session for purchasing a course.
	// It takes the course ID, user ID, and price as parameters.
	// Returns the checkout URL or an error if creation fails.
	// CreateCheckout(courseID, userID, price string) (CheckoutURL string, err error)

	// HandleWebhook processes incoming payment webhook events.
	// Returns the processed event or an error if processing fails.
	// HandleWebhook(r *http.Request) (Event, error)

	// GetSubscriptionStatus retrieves the subscription status for a user.
	// Returns the status string or an error if retrieval fails.
	// GetSubscriptionStatus(userID string) (status string, err error)
}
