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
		slog.Error("failed to read stripe webhook request body", log.BBError(err))
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
		slog.Error("failed to unmarshal stripe subscription from webhook",
			log.BBError(err),
			slog.String("event_type", string(event.Type)),
			slog.String("event_id", event.ID),
		)
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
			slog.String("event_id", event.ID),
		)
		return nil
	}

	status, err := mapStripeStatus(sub.Status)
	if err != nil {
		slog.Error("unsupported stripe subscription status",
			log.BBError(err),
			slog.String("workspace", workspace),
			slog.String("stripe_subscription_id", sub.ID),
			slog.String("stripe_status", string(sub.Status)),
		)
		return err
	}

	stripeCustomer, err := getStripeCustomer(sub.Customer)
	if err != nil {
		slog.Error("failed to get stripe customer",
			log.BBError(err),
			slog.String("workspace", workspace),
			slog.String("stripe_subscription_id", sub.ID),
		)
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
		slog.Error("failed to get subscription for workspace",
			log.BBError(err),
			slog.String("workspace", workspace),
			slog.String("stripe_subscription_id", sub.ID),
		)
		return errors.Wrapf(err, "failed to get subscription for workspace %s", workspace)
	}

	// Ignore stale events from an old Stripe subscription.
	// During plan changes, the old subscription is canceled and a new one is created.
	// Webhooks may arrive out of order — if the new subscription's invoice.paid
	// arrived first, the stored StripeSubscriptionId already points to the new one.
	// A late-arriving cancel event from the old subscription must not overwrite it.
	if existing != nil && existing.Payload != nil &&
		existing.Payload.StripeSubscriptionId != "" &&
		existing.Payload.StripeSubscriptionId != sub.ID {
		slog.Info("ignoring stale webhook for old stripe subscription",
			slog.String("workspace", workspace),
			slog.String("event_subscription_id", sub.ID),
			slog.String("current_subscription_id", existing.Payload.StripeSubscriptionId),
			slog.String("event_type", string(event.Type)),
		)
		return nil
	}

	var payload *storepb.SubscriptionPayload
	if existing != nil && existing.Payload != nil {
		payload = existing.Payload
	} else {
		// First webhook for this workspace — build payload from Stripe metadata.
		payload, err = buildPayloadFromMetadata(sub.Metadata)
		if err != nil {
			slog.Error("failed to build payload from stripe metadata",
				log.BBError(err),
				slog.String("workspace", workspace),
				slog.String("stripe_subscription_id", sub.ID),
			)
			return errors.Wrapf(err, "failed to build payload from metadata for workspace %s", workspace)
		}
	}

	payload.Status = status
	payload.StripeSubscriptionId = sub.ID
	payload.StripeCustomerId = stripeCustomer.ID

	if event.Type == "customer.subscription.created" && len(sub.Items.Data) > 0 {
		expiresAt := time.Unix(sub.Items.Data[0].CurrentPeriodEnd, 0).UTC()
		payload.ExpiresAt = timestamppb.New(expiresAt)
	}

	if _, err := h.store.UpsertSubscription(ctx, workspace, payload); err != nil {
		slog.Error("failed to upsert subscription",
			log.BBError(err),
			slog.String("workspace", workspace),
			slog.String("stripe_subscription_id", sub.ID),
		)
		return errors.Wrapf(err, "failed to upsert subscription for workspace %s", workspace)
	}

	if err := h.updateLicense(ctx, workspace, payload); err != nil {
		slog.Error("failed to update license after subscription change",
			log.BBError(err),
			slog.String("workspace", workspace),
			slog.String("stripe_subscription_id", sub.ID),
		)
	}
	return nil
}

func (h *WebhookHandler) handleInvoicePaid(ctx context.Context, event *stripego.Event) error {
	var inv stripego.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		slog.Error("failed to unmarshal stripe invoice from webhook",
			log.BBError(err),
			slog.String("event_id", event.ID),
		)
		return errors.Wrap(err, "failed to unmarshal invoice")
	}

	if len(inv.Lines.Data) == 0 {
		slog.Error("stripe invoice has no line items",
			slog.String("invoice_id", inv.ID),
			slog.String("event_id", event.ID),
		)
		return errors.Errorf("invoice %s has no line items", inv.ID)
	}

	metadata := inv.Lines.Data[0].Metadata
	if metadata["source"] != stripeplugin.WebhookSource {
		return nil
	}

	workspace := metadata["workspace"]
	if workspace == "" {
		slog.Error("missing workspace in invoice metadata",
			slog.String("invoice_id", inv.ID),
			slog.String("event_id", event.ID),
		)
		return nil
	}

	stripeCustomer, err := getStripeCustomer(inv.Customer)
	if err != nil {
		slog.Error("failed to get stripe customer from invoice",
			log.BBError(err),
			slog.String("workspace", workspace),
			slog.String("invoice_id", inv.ID),
		)
		return err
	}

	payload, err := buildPayloadFromMetadata(metadata)
	if err != nil {
		slog.Error("failed to build payload from invoice metadata",
			log.BBError(err),
			slog.String("workspace", workspace),
			slog.String("invoice_id", inv.ID),
		)
		return errors.Wrapf(err, "failed to build payload from invoice metadata for workspace %s", workspace)
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
		slog.String("invoice_id", inv.ID),
		slog.String("plan", payload.Plan.String()),
		slog.Int("seats", int(payload.Seat)),
		slog.Int("instances", int(payload.InstanceCount)),
		slog.Time("started_at", startedAt),
		slog.Time("expires_at", expiresAt),
	)

	payload.Status = storepb.SubscriptionPayload_ACTIVE
	payload.StartedAt = timestamppb.New(startedAt)
	payload.ExpiresAt = timestamppb.New(expiresAt)
	payload.StripeSubscriptionId = stripeSubID
	payload.StripeCustomerId = stripeCustomer.ID

	if _, err := h.store.UpsertSubscription(ctx, workspace, payload); err != nil {
		slog.Error("failed to upsert subscription from invoice",
			log.BBError(err),
			slog.String("workspace", workspace),
			slog.String("stripe_subscription_id", stripeSubID),
			slog.String("invoice_id", inv.ID),
		)
		return errors.Wrapf(err, "failed to upsert subscription for workspace %s", workspace)
	}

	if err := h.updateLicense(ctx, workspace, payload); err != nil {
		slog.Error("failed to update license after invoice paid",
			log.BBError(err),
			slog.String("workspace", workspace),
			slog.String("stripe_subscription_id", stripeSubID),
			slog.String("invoice_id", inv.ID),
		)
	}
	return nil
}

// updateLicense generates a license JWT from the subscription payload and stores it.
// For non-ACTIVE subscriptions, stores an empty license (reverts to FREE plan).
func (h *WebhookHandler) updateLicense(ctx context.Context, workspace string, payload *storepb.SubscriptionPayload) error {
	var license string

	if payload.Status == storepb.SubscriptionPayload_ACTIVE {
		var expiresAt time.Time
		if payload.ExpiresAt != nil {
			expiresAt = payload.ExpiresAt.AsTime()
		}

		var err error
		license, err = h.licenseService.CreateLicense(&enterprise.LicenseParams{
			Plan:        payload.Plan.String(),
			Seats:       int(payload.Seat),
			Instances:   int(payload.InstanceCount),
			WorkspaceID: workspace,
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create license JWT")
		}
	}

	return h.licenseService.StoreLicense(ctx, workspace, license)
}

// buildPayloadFromMetadata constructs a SubscriptionPayload from Stripe metadata.
// Returns error if any required field is missing or invalid.
func buildPayloadFromMetadata(metadata map[string]string) (*storepb.SubscriptionPayload, error) {
	plan, err := parsePlan(metadata["plan"])
	if err != nil {
		return nil, errors.Wrap(err, "metadata missing valid plan")
	}
	interval, err := parseInterval(metadata["interval"])
	if err != nil {
		return nil, errors.Wrap(err, "metadata missing valid interval")
	}
	instanceCount, err := strconv.Atoi(metadata["instance_count"])
	if err != nil {
		return nil, errors.Wrapf(err, "metadata missing valid instance_count %q", metadata["instance_count"])
	}
	userCount, err := strconv.Atoi(metadata["user_count"])
	if err != nil {
		return nil, errors.Wrapf(err, "metadata missing valid user_count %q", metadata["user_count"])
	}

	return &storepb.SubscriptionPayload{
		Plan:          plan,
		Interval:      interval,
		InstanceCount: int32(instanceCount),
		Seat:          int32(userCount),
	}, nil
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
