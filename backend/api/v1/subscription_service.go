package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

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

// UpdateSubscription updates the subscription license (self-hosted only).
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, req *connect.Request[v1pb.UpdateSubscriptionRequest]) (*connect.Response[v1pb.Subscription], error) {
	if s.profile.SaaS {
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

// CreatePurchase creates a new subscription purchase (SaaS only).
func (s *SubscriptionService) CreatePurchase(ctx context.Context, req *connect.Request[v1pb.CreatePurchaseRequest]) (*connect.Response[v1pb.CreatePurchaseResponse], error) {
	if !s.profile.SaaS {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("purchase is only available in SaaS mode"))
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)

	// Check no existing active subscription.
	existing, err := s.store.GetSubscriptionByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get subscription"))
	}
	if existing != nil && existing.Payload != nil && existing.Payload.Status == storepb.SubscriptionPayload_ACTIVE {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("workspace already has an active subscription"))
	}

	plan, interval, seats := convertV1PlanToStore(req.Msg.Plan), convertV1IntervalToStore(req.Msg.Interval), req.Msg.Seats
	priceModel, err := pricing.NewPriceModel(plan, interval, seats)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	planLimit := pricing.GetPlanLimit(plan)

	// Create or update subscription record with PENDING status.
	payload := &storepb.SubscriptionPayload{
		Status:        storepb.SubscriptionPayload_PENDING,
		Plan:          plan,
		Interval:      interval,
		Seat:          seats,
		InstanceCount: planLimit.InstanceCount,
		StartedAt:     timestamppb.Now(),
		ExpiresAt:     timestamppb.Now(),
	}

	if existing != nil {
		if _, err := s.store.UpdateSubscription(ctx, workspaceID, payload); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to update subscription"))
		}
	} else {
		if _, err := s.store.CreateSubscription(ctx, &store.SubscriptionMessage{
			Workspace: workspaceID,
			Payload:   payload,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to create subscription"))
		}
	}

	// Generate Stripe Checkout URL.
	redirectURL := fmt.Sprintf("%s/setting/subscription", s.profile.ExternalURL)
	paymentURL, err := stripeplugin.CreateCheckoutSession(&stripeplugin.CheckoutParams{
		Workspace:  workspaceID,
		PriceModel: priceModel,
		SuccessURL: fmt.Sprintf("%s?from=STRIPE", redirectURL),
		CancelURL:  redirectURL,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to create checkout session"))
	}

	return connect.NewResponse(&v1pb.CreatePurchaseResponse{
		PaymentUrl: paymentURL,
	}), nil
}

// UpdatePurchase updates an existing subscription (SaaS only).
func (s *SubscriptionService) UpdatePurchase(ctx context.Context, req *connect.Request[v1pb.UpdatePurchaseRequest]) (*connect.Response[v1pb.UpdatePurchaseResponse], error) {
	if !s.profile.SaaS {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("purchase is only available in SaaS mode"))
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)

	existing, err := s.store.GetSubscriptionByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get subscription"))
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no subscription found"))
	}
	if req.Msg.Etag != "" && req.Msg.Etag != existing.Etag {
		return nil, connect.NewError(connect.CodeAborted, errors.New("subscription is out of date, please refresh and try again"))
	}

	plan, interval, seats := convertV1PlanToStore(req.Msg.Plan), convertV1IntervalToStore(req.Msg.Interval), req.Msg.Seats
	priceModel, err := pricing.NewPriceModel(plan, interval, seats)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	planLimit := pricing.GetPlanLimit(plan)

	// If subscription is not active (PENDING/CANCELED), reset to PENDING and generate new checkout URL.
	if existing.Payload == nil || existing.Payload.Status != storepb.SubscriptionPayload_ACTIVE {
		payload := &storepb.SubscriptionPayload{
			Status:        storepb.SubscriptionPayload_PENDING,
			Plan:          plan,
			Interval:      interval,
			Seat:          seats,
			InstanceCount: planLimit.InstanceCount,
			StartedAt:     timestamppb.Now(),
			ExpiresAt:     timestamppb.Now(),
		}
		if _, err := s.store.UpdateSubscription(ctx, workspaceID, payload); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to update subscription"))
		}

		redirectURL := fmt.Sprintf("%s/setting/subscription", s.profile.ExternalURL)
		paymentURL, err := stripeplugin.CreateCheckoutSession(&stripeplugin.CheckoutParams{
			Workspace:  workspaceID,
			PriceModel: priceModel,
			SuccessURL: fmt.Sprintf("%s?from=STRIPE", redirectURL),
			CancelURL:  redirectURL,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to create checkout session"))
		}
		return connect.NewResponse(&v1pb.UpdatePurchaseResponse{PaymentUrl: paymentURL}), nil
	}

	// Active subscription — cancel old Stripe subscription and create new one.
	oldPayload := existing.Payload
	if oldPayload.StripeSubscriptionId == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("no Stripe subscription to update"))
	}

	// Clear metadata on old subscription so webhook ignores the cancel event.
	metadata := stripeplugin.GetMetadata(workspaceID, priceModel)
	metadata["workspace"] = ""
	if err := stripeplugin.UpdateSubscriptionMetadata(oldPayload.StripeSubscriptionId, metadata); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to clear old subscription metadata"))
	}

	// Cancel old subscription with forced refund.
	if _, err := stripeplugin.CancelSubscription(oldPayload.StripeSubscriptionId, workspaceID, true); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to cancel old subscription"))
	}

	// Try to create new Stripe subscription directly using existing payment method.
	oldStripeSub, err := stripeplugin.GetSubscription(oldPayload.StripeSubscriptionId, []string{"default_payment_method", "customer"})
	if err != nil {
		slog.Error("failed to get old stripe subscription, falling back to checkout",
			log.BBError(err),
			slog.String("workspace", workspaceID),
		)
		return s.fallbackToCheckout(ctx, workspaceID, priceModel, plan, interval, seats, planLimit.InstanceCount)
	}

	customerID := oldStripeSub.Customer.ID
	if customerID == "" {
		return s.fallbackToCheckout(ctx, workspaceID, priceModel, plan, interval, seats, planLimit.InstanceCount)
	}

	var paymentMethodID string
	if oldStripeSub.DefaultPaymentMethod != nil {
		paymentMethodID = oldStripeSub.DefaultPaymentMethod.ID
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
		return s.fallbackToCheckout(ctx, workspaceID, priceModel, plan, interval, seats, planLimit.InstanceCount)
	}

	slog.Info("created new stripe subscription for plan change",
		slog.String("workspace", workspaceID),
		slog.String("old_stripe_sub", oldPayload.StripeSubscriptionId),
		slog.String("new_stripe_sub", newSub.ID),
	)

	return connect.NewResponse(&v1pb.UpdatePurchaseResponse{}), nil
}

func (s *SubscriptionService) fallbackToCheckout(
	ctx context.Context,
	workspaceID string,
	priceModel *pricing.PriceModel,
	plan storepb.SubscriptionPayload_Plan,
	interval storepb.SubscriptionPayload_BillingInterval,
	seats int32,
	instanceCount int32,
) (*connect.Response[v1pb.UpdatePurchaseResponse], error) {
	payload := &storepb.SubscriptionPayload{
		Status:        storepb.SubscriptionPayload_PENDING,
		Plan:          plan,
		Interval:      interval,
		Seat:          seats,
		InstanceCount: instanceCount,
		StartedAt:     timestamppb.Now(),
		ExpiresAt:     timestamppb.Now(),
	}
	if _, err := s.store.UpdateSubscription(ctx, workspaceID, payload); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to update subscription"))
	}

	redirectURL := fmt.Sprintf("%s/setting/subscription", s.profile.ExternalURL)
	paymentURL, err := stripeplugin.CreateCheckoutSession(&stripeplugin.CheckoutParams{
		Workspace:  workspaceID,
		PriceModel: priceModel,
		SuccessURL: fmt.Sprintf("%s?from=STRIPE", redirectURL),
		CancelURL:  redirectURL,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to create checkout session"))
	}
	return connect.NewResponse(&v1pb.UpdatePurchaseResponse{PaymentUrl: paymentURL}), nil
}

// CancelPurchase cancels an active subscription (SaaS only).
func (s *SubscriptionService) CancelPurchase(ctx context.Context, req *connect.Request[v1pb.CancelPurchaseRequest]) (*connect.Response[v1pb.CancelPurchaseResponse], error) {
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

	// Update status. For prorated (monthly), set CANCELED + expire immediately.
	// For annual, the webhook will update the status when Stripe cancels at period end.
	if prorate {
		payload.Status = storepb.SubscriptionPayload_CANCELED
		payload.ExpiresAt = timestamppb.New(time.Now().Add(-time.Second))
		if _, err := s.store.UpdateSubscription(ctx, workspaceID, payload); err != nil {
			slog.Error("failed to update subscription after cancellation", log.BBError(err))
		}
		s.licenseService.InvalidateCache(workspaceID)
	}

	return connect.NewResponse(&v1pb.CancelPurchaseResponse{}), nil
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
		TotalPrice:  strconv.FormatInt(inv.Total, 10),
		Currency:    string(sub.Currency),
		PeriodStart: time.Unix(period.Start, 0).Format("2006-01-02"),
		PeriodEnd:   time.Unix(period.End, 0).Format("2006-01-02"),
	}

	// Create billing portal session for invoice management.
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

// ListPurchasePlans returns available plans for self-service purchase.
func (*SubscriptionService) ListPurchasePlans(_ context.Context, _ *connect.Request[v1pb.ListPurchasePlansRequest]) (*connect.Response[v1pb.ListPurchasePlansResponse], error) {
	plans := []*v1pb.PurchasePlan{
		{
			Type:                v1pb.PlanType_TEAM,
			SelfServicePurchase: true,
			Additionals: []*v1pb.PurchasePlanAdditional{
				{
					Type:         v1pb.PurchasePlanAdditional_USER,
					UnitPrice:    2000, // $20/user/month
					FreeCount:    0,
					MinimumCount: 1,
					MaximumCount: -1, // unlimited
				},
			},
			BillingMethods: []*v1pb.PurchaseBillingMethod{
				{Interval: v1pb.BillingInterval_MONTH},
				{Interval: v1pb.BillingInterval_YEAR},
			},
		},
		{
			Type:                v1pb.PlanType_ENTERPRISE,
			SelfServicePurchase: false,
		},
	}

	return connect.NewResponse(&v1pb.ListPurchasePlansResponse{Plans: plans}), nil
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
