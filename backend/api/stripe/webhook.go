// Package stripe handles Stripe webhook callbacks for SaaS subscription management.
package stripe

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pkg/errors"
	stripego "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/webhook"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	stripeplugin "github.com/bytebase/bytebase/backend/plugin/stripe"
	"github.com/bytebase/bytebase/backend/store"
)

// WebhookHandler handles Stripe webhook events.
type WebhookHandler struct {
	store          *store.Store
	licenseService *enterprise.LicenseService
	webhookSecret  string
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(store *store.Store, licenseService *enterprise.LicenseService, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{
		store:          store,
		licenseService: licenseService,
		webhookSecret:  webhookSecret,
	}
}

// RegisterRoutes registers the Stripe webhook route.
func (h *WebhookHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/callback", h.handleCallback)
}

func (h *WebhookHandler) handleCallback(c *echo.Context) error {
	ctx := c.Request().Context()
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusBadRequest, "failed to read request body")
	}

	sig := c.Request().Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(body, sig, h.webhookSecret)
	if err != nil {
		slog.Error("stripe webhook signature verification failed", log.BBError(err))
		return c.String(http.StatusBadRequest, "invalid signature")
	}

	if err := h.processEvent(ctx, &event); err != nil {
		slog.Error("failed to process stripe webhook event",
			log.BBError(err),
			slog.String("type", string(event.Type)),
			slog.String("id", event.ID),
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (h *WebhookHandler) processEvent(ctx context.Context, event *stripego.Event) error {
	switch event.Type {
	case
		"customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted",
		"customer.subscription.paused",
		"customer.subscription.resumed":
		return h.handleSubscriptionStatusChange(ctx, event)
	case "invoice.paid":
		return h.handleInvoicePaid(ctx, event)
	default:
		return nil
	}
}

func (h *WebhookHandler) handleSubscriptionStatusChange(ctx context.Context, event *stripego.Event) error {
	var sub stripego.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return errors.Wrap(err, "failed to unmarshal subscription")
	}

	if sub.Metadata["source"] != stripeplugin.WebhookSource {
		return nil
	}

	workspace := sub.Metadata["workspace"]
	if workspace == "" {
		slog.Error("missing workspace in stripe subscription metadata",
			slog.String("stripe_subscription_id", sub.ID),
			slog.String("event_type", string(event.Type)),
		)
		return nil
	}

	status, err := mapStripeStatus(sub.Status)
	if err != nil {
		return err
	}

	stripeCustomer, err := getStripeCustomer(sub.Customer)
	if err != nil {
		return err
	}

	slog.Info("stripe subscription status changed",
		slog.String("event_type", string(event.Type)),
		slog.String("workspace", workspace),
		slog.String("stripe_subscription_id", sub.ID),
		slog.String("stripe_status", string(sub.Status)),
		slog.String("mapped_status", status.String()),
	)

	existing, err := h.store.GetSubscriptionByWorkspace(ctx, workspace)
	if err != nil {
		return errors.Wrapf(err, "failed to get subscription for workspace %s", workspace)
	}
	if existing == nil {
		slog.Error("no subscription found for workspace", slog.String("workspace", workspace))
		return nil
	}

	payload := existing.Payload
	payload.Status = status
	payload.StripeSubscriptionId = sub.ID
	payload.StripeCustomerId = stripeCustomer.ID

	if event.Type == "customer.subscription.created" && len(sub.Items.Data) > 0 {
		expiresAt := time.Unix(sub.Items.Data[0].CurrentPeriodEnd, 0).UTC()
		payload.ExpiresAt = timestamppb.New(expiresAt)
	}

	if _, err := h.store.UpdateSubscription(ctx, workspace, payload); err != nil {
		return errors.Wrapf(err, "failed to update subscription for workspace %s", workspace)
	}

	h.licenseService.InvalidateCache(workspace)
	return nil
}

func (h *WebhookHandler) handleInvoicePaid(ctx context.Context, event *stripego.Event) error {
	var inv stripego.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return errors.Wrap(err, "failed to unmarshal invoice")
	}

	if len(inv.Lines.Data) == 0 {
		return errors.Errorf("invoice %s has no line items", inv.ID)
	}

	metadata := inv.Lines.Data[0].Metadata
	if metadata["source"] != stripeplugin.WebhookSource {
		return nil
	}

	workspace := metadata["workspace"]
	if workspace == "" {
		slog.Error("missing workspace in invoice metadata", slog.String("invoice_id", inv.ID))
		return nil
	}

	stripeCustomer, err := getStripeCustomer(inv.Customer)
	if err != nil {
		return err
	}

	// Parse metadata fields.
	plan, err := parsePlan(metadata["plan"])
	if err != nil {
		return err
	}
	interval, err := parseInterval(metadata["interval"])
	if err != nil {
		return err
	}
	instanceCount, err := strconv.Atoi(metadata["instance_count"])
	if err != nil {
		return errors.Wrapf(err, "failed to parse instance_count %q", metadata["instance_count"])
	}
	userCount, err := strconv.Atoi(metadata["user_count"])
	if err != nil {
		return errors.Wrapf(err, "failed to parse user_count %q", metadata["user_count"])
	}

	// Get Stripe subscription ID from invoice parent.
	var stripeSubID string
	if inv.Parent != nil && inv.Parent.SubscriptionDetails != nil && inv.Parent.SubscriptionDetails.Subscription != nil {
		stripeSubID = inv.Parent.SubscriptionDetails.Subscription.ID
	}

	lineData := inv.Lines.Data[0]
	startedAt := time.Unix(lineData.Period.Start, 0).UTC()
	expiresAt := time.Unix(lineData.Period.End, 0).UTC()

	slog.Info("stripe invoice paid",
		slog.String("workspace", workspace),
		slog.String("stripe_subscription_id", stripeSubID),
		slog.String("plan", plan.String()),
		slog.Int("seats", userCount),
		slog.Int("instances", instanceCount),
		slog.Time("started_at", startedAt),
		slog.Time("expires_at", expiresAt),
	)

	existing, err := h.store.GetSubscriptionByWorkspace(ctx, workspace)
	if err != nil {
		return errors.Wrapf(err, "failed to get subscription for workspace %s", workspace)
	}
	if existing == nil {
		return errors.Errorf("no subscription found for workspace %s", workspace)
	}

	payload := &storepb.SubscriptionPayload{
		Status:               storepb.SubscriptionPayload_ACTIVE,
		StartedAt:            timestamppb.New(startedAt),
		ExpiresAt:            timestamppb.New(expiresAt),
		Plan:                 plan,
		Interval:             interval,
		Seat:                 int32(userCount),
		InstanceCount:        int32(instanceCount),
		StripeSubscriptionId: stripeSubID,
		StripeCustomerId:     stripeCustomer.ID,
	}

	if _, err := h.store.UpdateSubscription(ctx, workspace, payload); err != nil {
		return errors.Wrapf(err, "failed to update subscription for workspace %s", workspace)
	}

	h.licenseService.InvalidateCache(workspace)
	return nil
}

func mapStripeStatus(status stripego.SubscriptionStatus) (storepb.SubscriptionPayload_Status, error) {
	switch status {
	case stripego.SubscriptionStatusActive, stripego.SubscriptionStatusTrialing:
		return storepb.SubscriptionPayload_ACTIVE, nil
	case stripego.SubscriptionStatusCanceled:
		return storepb.SubscriptionPayload_CANCELED, nil
	case stripego.SubscriptionStatusPastDue, stripego.SubscriptionStatusUnpaid,
		stripego.SubscriptionStatusPaused, stripego.SubscriptionStatusIncomplete,
		stripego.SubscriptionStatusIncompleteExpired:
		return storepb.SubscriptionPayload_PAUSED, nil
	default:
		return storepb.SubscriptionPayload_STATUS_UNSPECIFIED, errors.Errorf("unsupported stripe subscription status %v", status)
	}
}

func parsePlan(s string) (storepb.SubscriptionPayload_Plan, error) {
	v, ok := storepb.SubscriptionPayload_Plan_value[s]
	if !ok {
		return storepb.SubscriptionPayload_PLAN_UNSPECIFIED, errors.Errorf("unknown plan %q", s)
	}
	return storepb.SubscriptionPayload_Plan(v), nil
}

func parseInterval(s string) (storepb.SubscriptionPayload_BillingInterval, error) {
	v, ok := storepb.SubscriptionPayload_BillingInterval_value[s]
	if !ok {
		return storepb.SubscriptionPayload_BILLING_INTERVAL_UNSPECIFIED, errors.Errorf("unknown interval %q", s)
	}
	return storepb.SubscriptionPayload_BillingInterval(v), nil
}

// getStripeCustomer gets the full customer object.
// Stripe webhooks may only include the customer ID, not the full object.
func getStripeCustomer(c *stripego.Customer) (*stripego.Customer, error) {
	if c == nil {
		return nil, errors.New("empty customer field")
	}
	if c.ID != "" && c.Email != "" {
		return c, nil
	}
	return customer.Get(c.ID, nil)
}
