package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/enterprise/pricing"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	stripeplugin "github.com/bytebase/bytebase/backend/plugin/stripe"
	"github.com/bytebase/bytebase/backend/store"
)

// SubscriptionService implements the subscription service.
type SubscriptionService struct {
	v1connect.UnimplementedSubscriptionServiceHandler
	profile        *config.Profile
	store          *store.Store
	licenseService *enterprise.LicenseService
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	profile *config.Profile,
	stores *store.Store,
	licenseService *enterprise.LicenseService,
) *SubscriptionService {
	return &SubscriptionService{
		profile:        profile,
		store:          stores,
		licenseService: licenseService,
	}
}

// GetSubscription gets the subscription.
func (s *SubscriptionService) GetSubscription(ctx context.Context, _ *connect.Request[v1pb.GetSubscriptionRequest]) (*connect.Response[v1pb.Subscription], error) {
	subscription := s.licenseService.LoadSubscription(ctx, common.GetWorkspaceIDFromContext(ctx))
	return connect.NewResponse(subscription), nil
}

// UploadLicense uploads an enterprise license (self-hosted only).
func (s *SubscriptionService) UploadLicense(ctx context.Context, req *connect.Request[v1pb.UploadLicenseRequest]) (*connect.Response[v1pb.Subscription], error) {
	if s.profile.SaaS && s.profile.Mode == common.ReleaseModeProd {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("use purchase APIs in SaaS mode"))
	}

	if err := s.licenseService.StoreLicense(ctx, common.GetWorkspaceIDFromContext(ctx), req.Msg.License); err != nil {
		if common.ErrorCode(err) == common.Invalid {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to store license"))
	}

	subscription := s.licenseService.LoadSubscription(ctx, common.GetWorkspaceIDFromContext(ctx))
	return connect.NewResponse(subscription), nil
}

// CreatePurchase creates a Stripe Checkout session (SaaS only).
// Stateless — no subscription record is created. The subscription is only
// created when the Stripe webhook confirms payment.
func (s *SubscriptionService) CreatePurchase(ctx context.Context, req *connect.Request[v1pb.CreatePurchaseRequest]) (*connect.Response[v1pb.PurchaseResponse], error) {
	if !s.profile.SaaS {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("purchase is only available in SaaS mode"))
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)

	// Block if workspace already has an active or paused subscription.
	existing, err := s.store.GetSubscriptionByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get subscription"))
	}
	if existing != nil && existing.Payload != nil {
		switch existing.Payload.Status {
		case storepb.SubscriptionPayload_ACTIVE:
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("workspace already has an active subscription"))
		case storepb.SubscriptionPayload_PAUSED:
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("subscription is paused due to a payment issue, please resolve it before purchasing"))
		}
	}

	return s.createCheckout(workspaceID, req.Msg.Plan, req.Msg.Interval, req.Msg.Seats)
}

// UpdatePurchase updates an existing subscription (SaaS only).
func (s *SubscriptionService) UpdatePurchase(ctx context.Context, req *connect.Request[v1pb.UpdatePurchaseRequest]) (*connect.Response[v1pb.PurchaseResponse], error) {
	if !s.profile.SaaS {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("purchase is only available in SaaS mode"))
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)

	existing, err := s.store.GetSubscriptionByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get subscription"))
	}

	// No subscription or canceled — create a new checkout session.
	if existing == nil || existing.Payload == nil || existing.Payload.Status == storepb.SubscriptionPayload_CANCELED {
		return s.createCheckout(workspaceID, req.Msg.Plan, req.Msg.Interval, req.Msg.Seats)
	}

	// ACTIVE or PAUSED — cancel old Stripe subscription and create new one.
	if req.Msg.Etag != "" && req.Msg.Etag != existing.Etag {
		return nil, connect.NewError(connect.CodeAborted, errors.New("subscription is out of date, please refresh and try again"))
	}

	plan, interval, seats := convertV1PlanToStore(req.Msg.Plan), convertV1IntervalToStore(req.Msg.Interval), req.Msg.Seats
	priceModel, err := pricing.NewPriceModel(plan, interval, seats)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Cancel old Stripe subscription and create new one.
	// No need to clear metadata — the cancel webhook will set status=CANCELED,
	// and the invoice.paid webhook from the new subscription will upsert back to ACTIVE.
	oldPayload := existing.Payload
	if oldPayload.StripeSubscriptionId == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("no Stripe subscription to update"))
	}

	// Fetch old subscription to get customer ID and payment method before cancellation.
	oldStripeSub, err := stripeplugin.GetSubscription(oldPayload.StripeSubscriptionId, []string{"default_payment_method", "customer"})
	if err != nil {
		slog.Error("failed to get old stripe subscription, falling back to checkout",
			log.BBError(err),
			slog.String("workspace", workspaceID),
		)
		// No active subscription — just create a new checkout session.
		return s.createCheckout(workspaceID, req.Msg.Plan, req.Msg.Interval, req.Msg.Seats)
	}

	customerID := oldStripeSub.Customer.ID
	if customerID == "" {
		return s.createCheckout(workspaceID, req.Msg.Plan, req.Msg.Interval, req.Msg.Seats)
	}

	var paymentMethodID string
	if oldStripeSub.DefaultPaymentMethod != nil {
		paymentMethodID = oldStripeSub.DefaultPaymentMethod.ID
	}

	// Cancel old subscription. The webhook may briefly set status=CANCELED,
	// but the invoice.paid from the new subscription will upsert back to ACTIVE.
	if _, err := stripeplugin.CancelSubscription(oldPayload.StripeSubscriptionId, workspaceID, true); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to cancel old subscription"))
	}

	newSub, err := stripeplugin.CreateSubscriptionDirect(&stripeplugin.DirectSubscriptionParams{
		CustomerID:      customerID,
		PaymentMethodID: paymentMethodID,
		PriceModel:      priceModel,
		Metadata:        stripeplugin.GetMetadata(workspaceID, priceModel),
	})
	if err != nil {
		slog.Error("failed to create new stripe subscription directly, falling back to checkout",
			log.BBError(err),
			slog.String("workspace", workspaceID),
		)
		return s.createCheckout(workspaceID, req.Msg.Plan, req.Msg.Interval, req.Msg.Seats)
	}

	slog.Info("created new stripe subscription for plan change",
		slog.String("workspace", workspaceID),
		slog.String("old_stripe_sub", oldPayload.StripeSubscriptionId),
		slog.String("new_stripe_sub", newSub.ID),
	)

	return connect.NewResponse(&v1pb.PurchaseResponse{}), nil
}

// createCheckoutURL validates inputs and creates a Stripe Checkout session URL.
func (s *SubscriptionService) createCheckout(workspaceID string, v1Plan v1pb.PlanType, v1Interval v1pb.BillingInterval, seats int32) (*connect.Response[v1pb.PurchaseResponse], error) {
	plan, interval := convertV1PlanToStore(v1Plan), convertV1IntervalToStore(v1Interval)
	priceModel, err := pricing.NewPriceModel(plan, interval, seats)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	redirectURL := fmt.Sprintf("%s/setting/subscription", s.profile.ExternalURL)
	result, err := stripeplugin.CreateCheckoutSession(&stripeplugin.CheckoutParams{
		Workspace:     workspaceID,
		PriceModel:    priceModel,
		SuccessURL:    redirectURL + "?session_id={CHECKOUT_SESSION_ID}",
		CancelURL:     redirectURL,
		PromotionCode: priceModel.GetPromotionCode(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to create checkout session"))
	}
	return connect.NewResponse(&v1pb.PurchaseResponse{
		PaymentUrl: result.URL,
		SessionId:  result.SessionID,
	}), nil
}

// CancelPurchase cancels an active subscription (SaaS only).
func (s *SubscriptionService) CancelPurchase(ctx context.Context, _ *connect.Request[v1pb.CancelPurchaseRequest]) (*connect.Response[v1pb.PurchaseResponse], error) {
	if !s.profile.SaaS {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("purchase is only available in SaaS mode"))
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)

	existing, err := s.store.GetSubscriptionByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get subscription"))
	}
	if existing == nil || existing.Payload == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no subscription found"))
	}

	payload := existing.Payload
	if payload.StripeSubscriptionId == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("no Stripe subscription to cancel"))
	}

	// Monthly: immediate cancel with proration + refund.
	// Annual: cancel at period end.
	prorate := payload.Interval == storepb.SubscriptionPayload_MONTH
	if _, err := stripeplugin.CancelSubscription(payload.StripeSubscriptionId, workspaceID, prorate); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to cancel subscription"))
	}

	// Stripe webhook (customer.subscription.deleted) will update the subscription status and clear the license

	return connect.NewResponse(&v1pb.PurchaseResponse{}), nil
}

// GetPaymentInfo returns payment details (SaaS only).
func (s *SubscriptionService) GetPaymentInfo(ctx context.Context, _ *connect.Request[v1pb.GetPaymentInfoRequest]) (*connect.Response[v1pb.PaymentInfo], error) {
	if !s.profile.SaaS {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("purchase is only available in SaaS mode"))
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)

	existing, err := s.store.GetSubscriptionByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get subscription"))
	}
	if existing == nil || existing.Payload == nil || existing.Payload.StripeSubscriptionId == "" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no subscription found"))
	}

	payload := existing.Payload
	sub, err := stripeplugin.GetSubscription(payload.StripeSubscriptionId, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get stripe subscription"))
	}

	inv, err := stripeplugin.FindLastPaidInvoice(sub)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get invoice"))
	}

	period := inv.Lines.Data[0].Period
	info := &v1pb.PaymentInfo{
		TotalPrice:        strconv.FormatInt(inv.Total, 10),
		Currency:          string(sub.Currency),
		PeriodStart:       time.Unix(period.Start, 0).Format("2006-01-02"),
		PeriodEnd:         time.Unix(period.End, 0).Format("2006-01-02"),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
	}

	if payload.StripeCustomerId != "" {
		returnURL := fmt.Sprintf("%s/setting/subscription", s.profile.ExternalURL)
		portalURL, err := stripeplugin.CreateBillingPortalSession(payload.StripeCustomerId, returnURL)
		if err != nil {
			slog.Error("failed to create billing portal session", log.BBError(err))
		} else {
			info.InvoiceUrl = portalURL
		}
	}

	return connect.NewResponse(info), nil
}

// VerifyCheckoutSession verifies a Stripe Checkout Session status (SaaS only).
func (s *SubscriptionService) VerifyCheckoutSession(_ context.Context, req *connect.Request[v1pb.VerifyCheckoutSessionRequest]) (*connect.Response[v1pb.VerifyCheckoutSessionResponse], error) {
	if !s.profile.SaaS {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("purchase is only available in SaaS mode"))
	}

	if req.Msg.SessionId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}

	status, err := stripeplugin.GetCheckoutSessionStatus(req.Msg.SessionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to verify checkout session"))
	}

	return connect.NewResponse(&v1pb.VerifyCheckoutSessionResponse{
		Status: status,
	}), nil
}

// ListPurchasePlans returns available plans for self-service purchase.
func (s *SubscriptionService) ListPurchasePlans(_ context.Context, _ *connect.Request[v1pb.ListPurchasePlansRequest]) (*connect.Response[v1pb.ListPurchasePlansResponse], error) {
	// Non-SaaS: no purchase plans available.
	if !s.profile.SaaS {
		return connect.NewResponse(&v1pb.ListPurchasePlansResponse{}), nil
	}

	// TODO(ed): not correct. ENTERPRISE should NOT has Additionals
	allPlans := []storepb.SubscriptionPayload_Plan{
		storepb.SubscriptionPayload_TEAM,
		storepb.SubscriptionPayload_ENTERPRISE,
	}

	var plans []*v1pb.PurchasePlan
	for _, p := range allPlans {
		limit := pricing.GetPlanLimit(p)
		if limit == nil {
			continue
		}

		plan := &v1pb.PurchasePlan{
			Type:                convertStorePlanToV1(p),
			SelfServicePurchase: limit.SelfServicePurchase,
		}
		if limit.SelfServicePurchase {
			plan.Additionals = []*v1pb.PurchasePlanAdditional{
				{
					Type:         v1pb.PurchasePlanAdditional_USER,
					UnitPrice:    int32(limit.PricePerSeatPerMonth),
					FreeCount:    limit.FreeSeatCount,
					MinimumCount: 1,
					MaximumCount: limit.MaximumSeatCount,
				},
			}
		}

		for _, bm := range limit.BillingMethods {
			method := &v1pb.PurchaseBillingMethod{
				Interval: convertStoreIntervalToV1(bm.Interval),
				Discount: bm.Discount,
			}
			plan.BillingMethods = append(plan.BillingMethods, method)
		}

		plans = append(plans, plan)
	}

	return connect.NewResponse(&v1pb.ListPurchasePlansResponse{Plans: plans}), nil
}

func convertStorePlanToV1(plan storepb.SubscriptionPayload_Plan) v1pb.PlanType {
	switch plan {
	case storepb.SubscriptionPayload_TEAM:
		return v1pb.PlanType_TEAM
	case storepb.SubscriptionPayload_ENTERPRISE:
		return v1pb.PlanType_ENTERPRISE
	default:
		return v1pb.PlanType_FREE
	}
}

func convertStoreIntervalToV1(interval storepb.SubscriptionPayload_BillingInterval) v1pb.BillingInterval {
	switch interval {
	case storepb.SubscriptionPayload_MONTH:
		return v1pb.BillingInterval_MONTH
	case storepb.SubscriptionPayload_YEAR:
		return v1pb.BillingInterval_YEAR
	default:
		return v1pb.BillingInterval_BILLING_INTERVAL_UNSPECIFIED
	}
}

func convertV1PlanToStore(plan v1pb.PlanType) storepb.SubscriptionPayload_Plan {
	switch plan {
	case v1pb.PlanType_TEAM:
		return storepb.SubscriptionPayload_TEAM
	case v1pb.PlanType_ENTERPRISE:
		return storepb.SubscriptionPayload_ENTERPRISE
	default:
		return storepb.SubscriptionPayload_PLAN_UNSPECIFIED
	}
}

func convertV1IntervalToStore(interval v1pb.BillingInterval) storepb.SubscriptionPayload_BillingInterval {
	switch interval {
	case v1pb.BillingInterval_MONTH:
		return storepb.SubscriptionPayload_MONTH
	case v1pb.BillingInterval_YEAR:
		return storepb.SubscriptionPayload_YEAR
	default:
		return storepb.SubscriptionPayload_BILLING_INTERVAL_UNSPECIFIED
	}
}
