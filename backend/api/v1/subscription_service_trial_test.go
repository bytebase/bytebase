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
	"google.golang.org/protobuf/types/known/timestamppb"

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
	params := newTrialLicenseParams("workspace-a", startedAt)
	require.Equal(t, v1pb.PlanType_ENTERPRISE.String(), params.Plan)
	require.Equal(t, 20, params.Seats)
	require.Equal(t, 10, params.Instances)
	require.Equal(t, "workspace-a", params.WorkspaceID)
	require.True(t, params.Trialing)
	require.Equal(t, time.Date(2026, time.September, 21, 19, 4, 5, 0, time.UTC), params.ExpiresAt)

	subscription := subscriptionFromTrialParams(params)
	require.Equal(t, v1pb.PlanType_ENTERPRISE, subscription.Plan)
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

	t.Run("starts ENTERPRISE trial", func(t *testing.T) {
		requestContext := createWorkspace(t, "starts-enterprise-trial")
		response, err := service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{}))
		require.NoError(t, err)
		require.Equal(t, v1pb.PlanType_ENTERPRISE, response.Msg.Plan)
		require.Equal(t, int32(20), response.Msg.Seats)
		require.Equal(t, int32(10), response.Msg.Instances)
		require.True(t, response.Msg.Trialing)
		require.WithinDuration(t, time.Now().Add(14*24*time.Hour), response.Msg.ExpiresTime.AsTime(), 5*time.Second)

		setting, err := stores.GetSystemSettingUncached(requestContext, "starts-enterprise-trial")
		require.NoError(t, err)
		require.NotEmpty(t, setting.License)
		trialing, err := licenseService.IsTrialLicense(setting.License, "starts-enterprise-trial")
		require.NoError(t, err)
		require.True(t, trialing)
		billingSubscription, err := stores.GetSubscriptionByWorkspace(requestContext, "starts-enterprise-trial")
		require.NoError(t, err)
		require.Nil(t, billingSubscription, "a trial must not create Stripe subscription history")

		current, err := service.GetSubscription(requestContext, connect.NewRequest(&v1pb.GetSubscriptionRequest{}))
		require.NoError(t, err)
		require.Equal(t, v1pb.PlanType_ENTERPRISE, current.Msg.Plan)
		require.True(t, current.Msg.Trialing)

		_, err = service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})

	t.Run("rejects subscription history", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			payload *storepb.SubscriptionPayload
		}{
			{name: "active", payload: &storepb.SubscriptionPayload{Status: storepb.SubscriptionPayload_ACTIVE}},
			{name: "paused", payload: &storepb.SubscriptionPayload{Status: storepb.SubscriptionPayload_PAUSED}},
			{name: "canceled", payload: &storepb.SubscriptionPayload{Status: storepb.SubscriptionPayload_CANCELED}},
			{name: "unspecified", payload: &storepb.SubscriptionPayload{}},
			{name: "expired", payload: &storepb.SubscriptionPayload{
				Status:    storepb.SubscriptionPayload_ACTIVE,
				ExpiresAt: timestamppb.New(time.Now().Add(-time.Hour)),
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				workspace := "subscription-history-" + tc.name
				requestContext := createWorkspace(t, workspace)
				_, err := stores.UpsertSubscription(requestContext, workspace, tc.payload)
				require.NoError(t, err)

				_, err = service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{}))
				require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
			})
		}
	})

	t.Run("rejects expired trial", func(t *testing.T) {
		requestContext := createWorkspace(t, "expired-trial")
		license, err := licenseService.CreateLicense(&enterprise.LicenseParams{
			Plan:        v1pb.PlanType_TEAM.String(),
			WorkspaceID: "expired-trial",
			Trialing:    true,
			ExpiresAt:   time.Now().Add(-time.Hour),
		})
		require.NoError(t, err)
		require.NoError(t, stores.UpdateLicense(requestContext, "expired-trial", license))

		_, err = service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})

	t.Run("rejects invalid stored license", func(t *testing.T) {
		requestContext := createWorkspace(t, "invalid-license")
		require.NoError(t, stores.UpdateLicense(requestContext, "invalid-license", "not-a-license"))

		_, err := service.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{}))
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})

	t.Run("rejects self-host", func(t *testing.T) {
		requestContext := createWorkspace(t, "self-host")
		selfHostService := NewSubscriptionService(&config.Profile{}, stores, licenseService)
		_, err := selfHostService.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{}))
		require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
	})

	t.Run("rejects SaaS production", func(t *testing.T) {
		requestContext := createWorkspace(t, "saas-production")
		productionService := NewSubscriptionService(&config.Profile{SaaS: true, Mode: common.ReleaseModeProd}, stores, licenseService)
		_, err := productionService.StartTrial(requestContext, connect.NewRequest(&v1pb.StartTrialRequest{}))
		require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
	})

	t.Run("rejects license upload in SaaS development", func(t *testing.T) {
		requestContext := createWorkspace(t, "saas-development-upload")
		_, err := service.UploadLicense(requestContext, connect.NewRequest(&v1pb.UploadLicenseRequest{}))
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
