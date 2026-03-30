// Package stripe provides Stripe integration for SaaS subscription management.
package stripe

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/pkg/errors"
	stripego "github.com/stripe/stripe-go/v82"
	billingportalsession "github.com/stripe/stripe-go/v82/billingportal/session"
	"github.com/stripe/stripe-go/v82/charge"
	checkoutsession "github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/invoice"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/promotioncode"
	"github.com/stripe/stripe-go/v82/refund"
	stripesubscription "github.com/stripe/stripe-go/v82/subscription"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/enterprise/pricing"
)

// WebhookSource is used to filter webhook events from our checkout sessions.
const WebhookSource = "bytebase"

// Init sets the Stripe API key. Must be called before any other Stripe operations.
func Init(apiKey string) {
	stripego.Key = apiKey
}

// CheckoutParams contains the parameters for creating a Stripe Checkout Session.
type CheckoutParams struct {
	Workspace     string
	PriceModel    *pricing.PriceModel
	SuccessURL    string
	CancelURL     string
	PromotionCode string // optional promotion code string
}

// CheckoutResult contains the result of creating a checkout session.
type CheckoutResult struct {
	URL       string
	SessionID string
}

// CreateCheckoutSession creates a Stripe Checkout Session and returns the URL and session ID.
func CreateCheckoutSession(params *CheckoutParams) (*CheckoutResult, error) {
	priceInCents := params.PriceModel.GetPrice()
	metadata := GetMetadata(params.Workspace, params.PriceModel)

	checkoutParams := &stripego.CheckoutSessionParams{
		AutomaticTax: &stripego.CheckoutSessionAutomaticTaxParams{
			Enabled: stripego.Bool(false),
		},
		Mode: stripego.String(string(stripego.CheckoutSessionModeSubscription)),
		LineItems: []*stripego.CheckoutSessionLineItemParams{
			{
				PriceData: &stripego.CheckoutSessionLineItemPriceDataParams{
					ProductData: &stripego.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripego.String(fmt.Sprintf("%s License", params.PriceModel.GetPlanText())),
						Description: stripego.String(fmt.Sprintf(
							"%d Instances, %d Users",
							params.PriceModel.Plan.InstanceCount,
							params.PriceModel.Seats,
						)),
					},
					UnitAmount: stripego.Int64(priceInCents),
					Currency:   stripego.String(string(stripego.CurrencyUSD)),
					Recurring: &stripego.CheckoutSessionLineItemPriceDataRecurringParams{
						Interval:      stripego.String(params.PriceModel.GetStripeInterval()),
						IntervalCount: stripego.Int64(1),
					},
				},
				Quantity: stripego.Int64(1),
			},
		},
		SuccessURL: stripego.String(params.SuccessURL),
		CancelURL:  stripego.String(params.CancelURL),
		SubscriptionData: &stripego.CheckoutSessionSubscriptionDataParams{
			Metadata: metadata,
		},
		PaymentMethodCollection: stripego.String(stripego.CheckoutSessionPaymentMethodCollectionAlways),
	}

	if params.PromotionCode != "" {
		promo := GetPromotionByCode(params.PromotionCode, priceInCents)
		if promo != nil {
			checkoutParams.Discounts = []*stripego.CheckoutSessionDiscountParams{
				{PromotionCode: stripego.String(promo.ID)},
			}
		}
	}

	s, err := checkoutsession.New(checkoutParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create checkout session")
	}
	return &CheckoutResult{URL: s.URL, SessionID: s.ID}, nil
}

// GetCheckoutSessionStatus returns the status of a Stripe Checkout Session.
// Returns "complete", "expired", or "open".
func GetCheckoutSessionStatus(sessionID string) (string, error) {
	s, err := checkoutsession.Get(sessionID, nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get checkout session %s", sessionID)
	}
	return string(s.Status), nil
}

// CancelSubscription cancels a Stripe subscription.
// If prorate is true, cancels immediately with proration and processes refund.
// If prorate is false, schedules cancellation at the end of the current billing period.
func CancelSubscription(stripeSubID string, workspace string, prorate bool) (*stripego.Subscription, error) {
	if !prorate {
		// Schedule cancellation at period end — subscription remains active until then.
		sub, err := stripesubscription.Update(stripeSubID, &stripego.SubscriptionParams{
			CancelAtPeriodEnd: stripego.Bool(true),
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to schedule cancellation for stripe subscription %s", stripeSubID)
		}
		return sub, nil
	}

	// Immediate cancellation with proration.
	sub, err := stripesubscription.Cancel(stripeSubID, &stripego.SubscriptionCancelParams{
		InvoiceNow: stripego.Bool(true),
		Prorate:    stripego.Bool(true),
		Expand:     []*string{stripego.String("customer")},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to cancel stripe subscription %s", stripeSubID)
	}

	if err := RefundFromSubscription(sub, workspace); err != nil {
		slog.Error("failed to refund after cancellation",
			log.BBError(err),
			slog.String("stripe_subscription_id", stripeSubID),
			slog.String("workspace", workspace),
		)
		// Don't fail the cancellation itself
	}

	return sub, nil
}

// DirectSubscriptionParams contains the parameters for creating a Stripe subscription directly.
type DirectSubscriptionParams struct {
	CustomerID      string
	PaymentMethodID string // optional, from old subscription
	PriceModel      *pricing.PriceModel
	Metadata        map[string]string
}

// CreateSubscriptionDirect creates a new Stripe subscription using an existing customer's payment method.
func CreateSubscriptionDirect(params *DirectSubscriptionParams) (*stripego.Subscription, error) {
	priceInCents := params.PriceModel.GetPrice()

	newPrice, err := price.New(&stripego.PriceParams{
		ProductData: &stripego.PriceProductDataParams{
			Name: stripego.String(fmt.Sprintf("%s License", params.PriceModel.GetPlanText())),
		},
		UnitAmount: stripego.Int64(priceInCents),
		Currency:   stripego.String(string(stripego.CurrencyUSD)),
		Recurring: &stripego.PriceRecurringParams{
			Interval:      stripego.String(params.PriceModel.GetStripeInterval()),
			IntervalCount: stripego.Int64(1),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create stripe price")
	}

	subParams := &stripego.SubscriptionParams{
		Customer: stripego.String(params.CustomerID),
		Items: []*stripego.SubscriptionItemsParams{
			{
				Price:    stripego.String(newPrice.ID),
				Quantity: stripego.Int64(1),
			},
		},
		Metadata:        params.Metadata,
		PaymentBehavior: stripego.String("allow_incomplete"),
		Expand:          []*string{stripego.String("customer")},
	}

	if params.PaymentMethodID != "" {
		subParams.DefaultPaymentMethod = stripego.String(params.PaymentMethodID)
	}

	sub, err := stripesubscription.New(subParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create stripe subscription")
	}
	return sub, nil
}

// GetSubscription retrieves a Stripe subscription by ID with optional expand fields.
func GetSubscription(stripeSubID string, expand []string) (*stripego.Subscription, error) {
	params := &stripego.SubscriptionParams{}
	for _, e := range expand {
		params.AddExpand(e)
	}
	sub, err := stripesubscription.Get(stripeSubID, params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get stripe subscription %s", stripeSubID)
	}
	return sub, nil
}

// GetMetadata builds the metadata map for a Stripe subscription.
func GetMetadata(workspace string, model *pricing.PriceModel) map[string]string {
	return map[string]string{
		"workspace":      workspace,
		"plan":           model.Plan.Plan.String(),
		"interval":       model.Interval.String(),
		"instance_count": strconv.Itoa(int(model.Plan.InstanceCount)),
		"user_count":     strconv.Itoa(int(model.Seats)),
		"source":         WebhookSource,
	}
}

// FindLastPaidInvoice finds the last paid invoice for a subscription.
func FindLastPaidInvoice(sub *stripego.Subscription) (*stripego.Invoice, error) {
	return findLastInvoiceByStatus(sub, stripego.InvoiceStatusPaid)
}

func findLastInvoiceByStatus(sub *stripego.Subscription, status stripego.InvoiceStatus) (*stripego.Invoice, error) {
	if sub.Customer == nil || sub.Customer.ID == "" {
		return nil, errors.Errorf("no customer in subscription %s", sub.ID)
	}

	invoices := invoice.List(&stripego.InvoiceListParams{
		Subscription: stripego.String(sub.ID),
		Customer:     stripego.String(sub.Customer.ID),
		Status:       stripego.String(string(status)),
		ListParams:   stripego.ListParams{Limit: stripego.Int64(1)},
	})
	if !invoices.Next() {
		if err := invoices.Err(); err != nil {
			return nil, errors.Wrapf(err, "error listing invoices for subscription %s", sub.ID)
		}
		return nil, errors.Errorf("no %s invoices found for subscription %s", status, sub.ID)
	}

	inv := invoices.Invoice()
	if len(inv.Lines.Data) == 0 {
		return nil, errors.Errorf("no line items in invoice for subscription %s", sub.ID)
	}
	if inv.Lines.Data[0].Period == nil {
		return nil, errors.Errorf("no period in invoice line item for subscription %s", sub.ID)
	}

	return inv, nil
}

// RefundFromSubscription refunds proration credit after subscription cancellation.
func RefundFromSubscription(sub *stripego.Subscription, workspace string) error {
	customerID := sub.Customer.ID
	if customerID == "" {
		return errors.Errorf("no customer ID in subscription %s", sub.ID)
	}

	draftInvoice, err := findLastInvoiceByStatus(sub, stripego.InvoiceStatusDraft)
	if err != nil {
		return err
	}

	var refundAmount int64
	for _, line := range draftInvoice.Lines.Data {
		if line.Amount < 0 {
			refundAmount += -line.Amount
		}
	}

	if refundAmount < 100 { // Skip refunds under $1
		return nil
	}

	slog.Info("processing proration refund",
		slog.String("stripe_subscription_id", sub.ID),
		slog.String("workspace", workspace),
		slog.Int64("refund_amount_cents", refundAmount),
	)

	return refundCustomerBalance(customerID, refundAmount, workspace, sub.ID)
}

// refundCustomerBalance refunds a credit balance across multiple charges.
func refundCustomerBalance(customerID string, totalRefundAmount int64, workspace string, stripeSubID string) error {
	charges := charge.List(&stripego.ChargeListParams{
		Customer:   stripego.String(customerID),
		ListParams: stripego.ListParams{Limit: stripego.Int64(20)},
	})

	remaining := totalRefundAmount
	refundCount := 0

	for charges.Next() && remaining > 0 {
		ch := charges.Charge()
		if !ch.Paid || ch.Refunded {
			continue
		}
		available := ch.Amount - ch.AmountRefunded
		if available <= 0 {
			continue
		}

		amount := min(remaining, available)
		result, err := refund.New(&stripego.RefundParams{
			Charge: stripego.String(ch.ID),
			Amount: stripego.Int64(amount),
			Reason: stripego.String(string(stripego.RefundReasonRequestedByCustomer)),
			Metadata: map[string]string{
				"workspace":             workspace,
				"stripe_subscription":   stripeSubID,
				"refund_reason":         "subscription_cancellation_proration",
				"total_balance_refund":  fmt.Sprintf("%d", totalRefundAmount),
				"partial_refund_amount": fmt.Sprintf("%d", amount),
			},
		})
		if err != nil {
			slog.Error("failed to create partial refund",
				slog.String("charge_id", ch.ID),
				slog.Int64("refund_amount", amount),
				log.BBError(err),
			)
			continue
		}

		slog.Info("created partial refund",
			slog.String("refund_id", result.ID),
			slog.String("charge_id", ch.ID),
			slog.Int64("amount_cents", amount),
		)
		remaining -= amount
		refundCount++
	}

	if err := charges.Err(); err != nil {
		return errors.Wrap(err, "error listing charges for refund")
	}

	if remaining > 0 {
		return errors.Errorf("incomplete refund for workspace %s: %d of %d cents remain unrefunded after %d charges",
			workspace, remaining, totalRefundAmount, refundCount)
	}

	slog.Info("refund complete",
		slog.String("workspace", workspace),
		slog.Int64("total_refunded_cents", totalRefundAmount),
		slog.Int("refund_count", refundCount),
	)
	return nil
}

// CreateBillingPortalSession creates a Stripe Billing Portal session for invoice management.
func CreateBillingPortalSession(customerID string, returnURL string) (string, error) {
	s, err := billingportalsession.New(&stripego.BillingPortalSessionParams{
		Customer:  stripego.String(customerID),
		ReturnURL: stripego.String(returnURL),
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to create billing portal session")
	}
	return s.URL, nil
}

// GetPromotionByCode looks up an active promotion code eligible for first-time customers.
func GetPromotionByCode(code string, priceInCents int64) *stripego.PromotionCode {
	iter := promotioncode.List(&stripego.PromotionCodeListParams{
		Code:   stripego.String(code),
		Active: stripego.Bool(true),
	})
	for iter.Next() {
		promo := iter.PromotionCode()
		if v := promo.Restrictions; v != nil {
			if v.MinimumAmount <= priceInCents && v.FirstTimeTransaction {
				return promo
			}
		}
	}
	if err := iter.Err(); err != nil {
		slog.Error("failed to get promotion", log.BBError(err), slog.String("code", code))
	}
	return nil
}
