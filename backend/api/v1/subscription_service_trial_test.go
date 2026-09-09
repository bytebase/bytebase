package v1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestNewTrialLicenseParams(t *testing.T) {
	startedAt := time.Date(2026, time.September, 8, 3, 4, 5, 123, time.FixedZone("test", 8*60*60))
	params := newTrialLicenseParams("workspace-a", v1pb.PlanType_TEAM, startedAt)
	require.Equal(t, v1pb.PlanType_TEAM.String(), params.Plan)
	require.Equal(t, 20, params.Seats)
	require.Equal(t, 10, params.Instances)
	require.Equal(t, "workspace-a", params.WorkspaceID)
	require.True(t, params.Trialing)
	require.Equal(t, time.Date(2026, time.September, 21, 19, 4, 5, 0, time.UTC), params.ExpiresAt)

	subscription := subscriptionFromTrialParams(params)
	require.Equal(t, v1pb.PlanType_TEAM, subscription.Plan)
	require.Equal(t, int32(20), subscription.Seats)
	require.Equal(t, int32(10), subscription.Instances)
	require.Equal(t, int32(10), subscription.ActiveInstances)
	require.True(t, subscription.Trialing)
	require.Equal(t, params.ExpiresAt, subscription.ExpiresTime.AsTime())
}

func TestSubscriptionServiceStartTrial(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	_, stores, _ := testcontainer.NewMetadataDB(t)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, true, string(privateKeyPEM))
	require.NoError(t, err)
	setLicenseServicePublicKey(t, licenseService, &privateKey.PublicKey)

	service := NewSubscriptionService(&config.Profile{SaaS: true, Mode: common.ReleaseModeDev}, stores, licenseService)
	createWorkspace := func(t *testing.T, workspace string) context.Context {
		t.Helper()
		_, err := stores.GetDB().ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ($1)`, workspace)
		require.NoError(t, err)
		_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
			Name:      storepb.SettingName_SYSTEM,
			Workspace: workspace,
			Value:     &storepb.SystemSetting{},
		})
		require.NoError(t, err)
		return context.WithValue(ctx, common.WorkspaceIDContextKey, workspace)
	}

	t.Run("starts TEAM trial", func(t *testing.T) {
		requestContext := createWorkspace(t, "starts-team-trial")
		response, err := service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_TEAM}))
		require.NoError(t, err)
		require.Equal(t, v1pb.PlanType_TEAM, response.Msg.Plan)
		require.Equal(t, int32(20), response.Msg.Seats)
		require.Equal(t, int32(10), response.Msg.Instances)
		require.True(t, response.Msg.Trialing)
		require.WithinDuration(t, time.Now().Add(14*24*time.Hour), response.Msg.ExpiresTime.AsTime(), 5*time.Second)

		setting, err := stores.GetSystemSettingUncached(requestContext, "starts-team-trial")
		require.NoError(t, err)
		require.NotEmpty(t, setting.License)
		trialing, err := licenseService.IsTrialLicense(setting.License, "starts-team-trial")
		require.NoError(t, err)
		require.True(t, trialing)
		billingSubscription, err := stores.GetSubscriptionByWorkspace(requestContext, "starts-team-trial")
		require.NoError(t, err)
		require.Nil(t, billingSubscription, "a trial must not create Stripe subscription history")

		current, err := service.GetSubscription(requestContext, connect.NewRequest(&v1pb.GetSubscriptionRequest{}))
		require.NoError(t, err)
		require.Equal(t, v1pb.PlanType_TEAM, current.Msg.Plan)
		require.True(t, current.Msg.Trialing)

		_, err = service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_TEAM}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})

	t.Run("starts ENTERPRISE trial", func(t *testing.T) {
		requestContext := createWorkspace(t, "starts-enterprise-trial")
		response, err := service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_ENTERPRISE}))
		require.NoError(t, err)
		require.Equal(t, v1pb.PlanType_ENTERPRISE, response.Msg.Plan)
		require.True(t, response.Msg.Trialing)

		_, err = service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_ENTERPRISE}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})

	t.Run("upgrades TEAM trial without extending expiration", func(t *testing.T) {
		requestContext := createWorkspace(t, "upgrades-team-trial")
		team, err := service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_TEAM}))
		require.NoError(t, err)

		enterpriseTrial, err := service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_ENTERPRISE}))
		require.NoError(t, err)
		require.Equal(t, v1pb.PlanType_ENTERPRISE, enterpriseTrial.Msg.Plan)
		require.True(t, enterpriseTrial.Msg.Trialing)
		require.Equal(t, team.Msg.ExpiresTime.AsTime(), enterpriseTrial.Msg.ExpiresTime.AsTime())

		storedTrial, err := licenseService.LoadSubscriptionFromDB(requestContext, "upgrades-team-trial")
		require.NoError(t, err)
		require.Equal(t, v1pb.PlanType_ENTERPRISE, storedTrial.Plan)
		require.Equal(t, team.Msg.ExpiresTime.AsTime(), storedTrial.ExpiresTime.AsTime())
	})

	t.Run("rejects trial downgrade", func(t *testing.T) {
		requestContext := createWorkspace(t, "rejects-trial-downgrade")
		_, err := service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_ENTERPRISE}))
		require.NoError(t, err)

		_, err = service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_TEAM}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})

	for _, plan := range []v1pb.PlanType{v1pb.PlanType_PLAN_TYPE_UNSPECIFIED, v1pb.PlanType_FREE} {
		t.Run("rejects "+plan.String(), func(t *testing.T) {
			requestContext := createWorkspace(t, "rejects-plan-"+plan.String())
			_, err := service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: plan}))
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}

	t.Run("rejects subscription history", func(t *testing.T) {
		requestContext := createWorkspace(t, "subscription-history")
		_, err := stores.UpsertSubscription(requestContext, "subscription-history", &storepb.SubscriptionPayload{
			Status: storepb.SubscriptionPayload_CANCELED,
		})
		require.NoError(t, err)

		_, err = service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_TEAM}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})

	t.Run("upgrades expired TEAM trial without extending expiration", func(t *testing.T) {
		requestContext := createWorkspace(t, "expired-trial")
		expiresAt := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
		license, err := licenseService.CreateLicense(&enterprise.LicenseParams{
			Plan:        v1pb.PlanType_TEAM.String(),
			WorkspaceID: "expired-trial",
			Trialing:    true,
			ExpiresAt:   expiresAt,
		})
		require.NoError(t, err)
		require.NoError(t, stores.UpdateLicense(requestContext, "expired-trial", license))

		response, err := service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_ENTERPRISE}))
		require.NoError(t, err)
		require.Equal(t, v1pb.PlanType_ENTERPRISE, response.Msg.Plan)
		require.Equal(t, expiresAt, response.Msg.ExpiresTime.AsTime())
	})

	t.Run("rejects invalid stored license", func(t *testing.T) {
		requestContext := createWorkspace(t, "invalid-license")
		require.NoError(t, stores.UpdateLicense(requestContext, "invalid-license", "not-a-license"))

		_, err := service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_TEAM}))
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})

	t.Run("rejects paid license", func(t *testing.T) {
		requestContext := createWorkspace(t, "paid-license")
		license, err := licenseService.CreateLicense(&enterprise.LicenseParams{
			Plan:        v1pb.PlanType_TEAM.String(),
			WorkspaceID: "paid-license",
			ExpiresAt:   time.Now().Add(30 * 24 * time.Hour),
		})
		require.NoError(t, err)
		require.NoError(t, stores.UpdateLicense(requestContext, "paid-license", license))

		_, err = service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_ENTERPRISE}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})

	t.Run("rejects self-host", func(t *testing.T) {
		requestContext := createWorkspace(t, "self-host")
		selfHostService := NewSubscriptionService(&config.Profile{}, stores, licenseService)
		_, err := selfHostService.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_TEAM}))
		require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
	})

	t.Run("rejects SaaS production", func(t *testing.T) {
		requestContext := createWorkspace(t, "saas-production")
		productionService := NewSubscriptionService(&config.Profile{SaaS: true, Mode: common.ReleaseModeProd}, stores, licenseService)
		_, err := productionService.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{Plan: v1pb.PlanType_TEAM}))
		require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
	})
}

func setLicenseServicePublicKey(t *testing.T, service *enterprise.LicenseService, publicKey *rsa.PublicKey) {
	t.Helper()
	field := reflect.ValueOf(service).Elem().FieldByName("config")
	licenseConfig, ok := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(*enterprise.Config)
	require.True(t, ok)
	licenseConfig.PublicKey = publicKey
}
