// Package sslcommerz implements domain.PaymentGateway against SSLCommerz — the
// dominant Bangladeshi aggregator, which fronts cards plus every MFS wallet
// (bKash, Nagad, Rocket, …) behind one hosted checkout.
//
// Checkout opens a hosted session and returns the GatewayPageURL. IPN callbacks
// are authenticated by calling the SSLCommerz validation API with the callback's
// val_id — never by trusting the POST body alone.
package sslcommerz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
)

const (
	sandboxBase = "https://sandbox.sslcommerz.com"
	liveBase    = "https://securepay.sslcommerz.com"
)

type Gateway struct {
	storeID   string
	storePass string
	base      string
	appURL    string
	http      *http.Client
}

// New builds the gateway. sandbox selects the SSLCommerz test host; appURL is the
// app's base URL used to build the success/fail/IPN callback URLs.
func New(storeID, storePass, appURL string, sandbox bool) *Gateway {
	base := liveBase
	if sandbox {
		base = sandboxBase
	}
	return &Gateway{
		storeID:   storeID,
		storePass: storePass,
		base:      base,
		appURL:    appURL,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

func (g *Gateway) Name() string { return "sslcommerz" }

// Checkout initializes a hosted payment session and returns its GatewayPageURL.
func (g *Gateway) Checkout(ctx context.Context, req domain.PaymentRequest) (domain.PaymentCheckout, error) {
	form := url.Values{
		"store_id":         {g.storeID},
		"store_passwd":     {g.storePass},
		"total_amount":     {strconv.Itoa(req.AmountBDT)},
		"currency":         {"BDT"},
		"tran_id":          {req.PaymentID},
		"success_url":      {g.appURL + "/billing/return"},
		"fail_url":         {g.appURL + "/billing/return"},
		"cancel_url":       {g.appURL + "/billing/return"},
		"ipn_url":          {g.appURL + "/webhooks/payment"},
		"shipping_method":  {"NO"},
		"num_of_item":      {"1"},
		"product_name":     {fmt.Sprintf("%d Yaver credits", req.Credits)},
		"product_category": {"credits"},
		"product_profile":  {"non-physical-goods"},
		"cus_name":         {"Yaver merchant"},
		"cus_email":        {"billing@yaver.app"},
		"cus_add1":         {"N/A"},
		"cus_city":         {"Dhaka"},
		"cus_country":      {"Bangladesh"},
		"cus_phone":        {"0000000000"},
	}

	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, g.base+"/gwprocess/v4/api.php", strings.NewReader(form.Encode()))
	if err != nil {
		return domain.PaymentCheckout{}, err
	}
	reqHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.http.Do(reqHTTP)
	if err != nil {
		return domain.PaymentCheckout{}, err
	}
	defer resp.Body.Close()

	var out struct {
		Status         string `json:"status"`
		FailedReason   string `json:"failedreason"`
		GatewayPageURL string `json:"GatewayPageURL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return domain.PaymentCheckout{}, err
	}
	if out.Status != "SUCCESS" || out.GatewayPageURL == "" {
		return domain.PaymentCheckout{}, fmt.Errorf("sslcommerz session failed: %s", out.FailedReason)
	}
	return domain.PaymentCheckout{ProviderRef: req.PaymentID, RedirectURL: out.GatewayPageURL}, nil
}

// VerifyIPN authenticates the callback via the SSLCommerz validation API. The
// posted body is untrusted; only val_id validated against our store credentials
// confirms a real payment.
func (g *Gateway) VerifyIPN(ctx context.Context, form map[string]string) (domain.PaymentResult, bool, error) {
	valID := form["val_id"]
	tranID := form["tran_id"]
	if valID == "" || tranID == "" {
		return domain.PaymentResult{}, false, nil
	}

	q := url.Values{
		"val_id":       {valID},
		"store_id":     {g.storeID},
		"store_passwd": {g.storePass},
		"format":       {"json"},
	}
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodGet, g.base+"/validator/api/validationserverAPI.php?"+q.Encode(), nil)
	if err != nil {
		return domain.PaymentResult{}, false, err
	}
	resp, err := g.http.Do(reqHTTP)
	if err != nil {
		return domain.PaymentResult{}, false, err
	}
	defer resp.Body.Close()

	var out struct {
		Status string `json:"status"`
		TranID string `json:"tran_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return domain.PaymentResult{}, false, err
	}
	// The validated tran_id must match the callback's — otherwise the val_id
	// belongs to a different transaction and the callback is not authentic.
	if out.TranID != tranID {
		return domain.PaymentResult{}, false, nil
	}

	switch out.Status {
	case "VALID", "VALIDATED":
		return domain.PaymentResult{ProviderRef: tranID, Status: domain.PaymentPaid}, true, nil
	default:
		return domain.PaymentResult{ProviderRef: tranID, Status: domain.PaymentFailed}, true, nil
	}
}
