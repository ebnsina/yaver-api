// Package mock is a dev payment gateway — no real money moves. Checkout sends the
// customer to the app's dev-complete endpoint, and every IPN it "verifies" is
// trusted. It lets the whole top-up loop run locally without gateway credentials.
package mock

import (
	"context"
	"net/url"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type Gateway struct{ appURL string }

// New returns a mock gateway. appURL is the app's base URL used to build the
// dev-complete redirect.
func New(appURL string) *Gateway { return &Gateway{appURL: appURL} }

func (g *Gateway) Name() string { return "mock" }

func (g *Gateway) Checkout(_ context.Context, req domain.PaymentRequest) (domain.PaymentCheckout, error) {
	u := g.appURL + "/v1/dev/pay?ref=" + url.QueryEscape(req.PaymentID)
	return domain.PaymentCheckout{ProviderRef: req.PaymentID, RedirectURL: u}, nil
}

func (g *Gateway) VerifyIPN(_ context.Context, form map[string]string) (domain.PaymentResult, bool, error) {
	status := domain.PaymentPaid
	if form["status"] == "failed" {
		status = domain.PaymentFailed
	}
	return domain.PaymentResult{ProviderRef: form["tran_id"], Status: status}, true, nil
}
