package enterprise

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func newTestLicenseService(sub *v1pb.Subscription) *LicenseService {
	s := &LicenseService{
		cache: expirable.NewLRU[string, *v1pb.Subscription](8, nil, time.Minute),
	}
	s.cache.Add(licenseCacheKey("test-workspace"), sub)
	return s
}

func TestIsUnifiedInstanceLimit(t *testing.T) {
	tests := []struct {
		name           string
		instanceLimit  int
		activatedLimit int
		want           bool
	}{
		{name: "equal finite caps", instanceLimit: 10, activatedLimit: 10, want: true},
		{name: "activated cap larger than registration cap", instanceLimit: 10, activatedLimit: 20, want: true},
		{name: "split cap", instanceLimit: 50, activatedLimit: 20, want: false},
		{name: "unlimited both sides", instanceLimit: math.MaxInt, activatedLimit: math.MaxInt, want: true},
		{name: "unlimited registration finite activation", instanceLimit: math.MaxInt, activatedLimit: 20, want: false},
		{name: "finite registration unlimited activation", instanceLimit: 20, activatedLimit: math.MaxInt, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUnifiedInstanceLimit(tt.instanceLimit, tt.activatedLimit); got != tt.want {
				t.Fatalf("isUnifiedInstanceLimit(%d, %d) = %v, want %v", tt.instanceLimit, tt.activatedLimit, got, tt.want)
			}
		})
	}
}

func TestIsFeatureEnabledForInstanceUnifiedLicense(t *testing.T) {
	ctx := context.Background()
	instance := &store.InstanceMessage{
		ResourceID: "prod",
		Workspace:  "test-workspace",
		Metadata:   &storepb.Instance{Activation: false},
	}
	service := newTestLicenseService(&v1pb.Subscription{
		Plan:            v1pb.PlanType_ENTERPRISE,
		Instances:       10,
		ActiveInstances: 10,
	})

	if err := service.IsFeatureEnabledForInstance(ctx, "test-workspace", v1pb.PlanFeature_FEATURE_DATA_MASKING, instance); err != nil {
		t.Fatalf("unified license should enable feature for inactive stored instance: %v", err)
	}
}

func TestIsFeatureEnabledForInstanceSplitLicense(t *testing.T) {
	ctx := context.Background()
	instance := &store.InstanceMessage{
		ResourceID: "prod",
		Workspace:  "test-workspace",
		Metadata:   &storepb.Instance{Activation: false},
	}
	service := newTestLicenseService(&v1pb.Subscription{
		Plan:            v1pb.PlanType_ENTERPRISE,
		Instances:       50,
		ActiveInstances: 20,
	})

	if err := service.IsFeatureEnabledForInstance(ctx, "test-workspace", v1pb.PlanFeature_FEATURE_DATA_MASKING, instance); err == nil {
		t.Fatal("split license should still require stored activation")
	}
}

func TestIsInstanceEffectivelyActivated(t *testing.T) {
	ctx := context.Background()
	instance := &store.InstanceMessage{
		ResourceID: "prod",
		Workspace:  "test-workspace",
		Metadata:   &storepb.Instance{Activation: false},
	}

	unifiedService := newTestLicenseService(&v1pb.Subscription{
		Plan:            v1pb.PlanType_ENTERPRISE,
		Instances:       10,
		ActiveInstances: 10,
	})
	if !unifiedService.IsInstanceEffectivelyActivated(ctx, "test-workspace", instance) {
		t.Fatal("unified license should effectively activate stored inactive instance")
	}

	splitService := newTestLicenseService(&v1pb.Subscription{
		Plan:            v1pb.PlanType_ENTERPRISE,
		Instances:       50,
		ActiveInstances: 20,
	})
	if splitService.IsInstanceEffectivelyActivated(ctx, "test-workspace", instance) {
		t.Fatal("split license should use stored inactive state")
	}

	instance.Metadata.Activation = true
	if !splitService.IsInstanceEffectivelyActivated(ctx, "test-workspace", instance) {
		t.Fatal("split license should keep stored active state")
	}
}

func TestCreateLicenseUsesEqualInstanceClaims(t *testing.T) {
	claims := newLicenseClaims(&LicenseParams{
		Plan:        v1pb.PlanType_ENTERPRISE.String(),
		Seats:       5,
		Instances:   10,
		WorkspaceID: "test-workspace",
	})
	if claims.Instances != 10 {
		t.Fatalf("Instances = %d, want 10", claims.Instances)
	}
	if claims.ActiveInstances != 10 {
		t.Fatalf("ActiveInstances = %d, want 10", claims.ActiveInstances)
	}
}

func TestGetUserLimitUncached(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort())
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	// Sign licenses with a test-only keypair so finite and expired licenses can
	// be exercised.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)})
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
	require.NoError(t, err)
	licenseService := &LicenseService{
		store: s,
		config: &Config{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
			Version:    keyID,
			Issuer:     issuer,
			Audience:   audience,
			Mode:       common.ReleaseModeDev,
		},
		cache: expirable.NewLRU[string, *v1pb.Subscription](8, nil, time.Minute),
	}
	licenseService.replicaCache.Store(&replicaCacheState{replicaCount: 1, loadedAt: time.Now()})

	// No workspace: the Free plan limit applies.
	limit, err := licenseService.GetUserLimitUncached(ctx, "")
	require.NoError(t, err)
	require.Equal(t, userLimitValues[v1pb.PlanType_FREE], limit)

	// Workspace without a license: the Free plan limit applies.
	_, err = s.CreateWorkspace(ctx, &store.WorkspaceMessage{ResourceID: "ws-a"}, "admin@example.com")
	require.NoError(t, err)
	limit, err = licenseService.GetUserLimitUncached(ctx, "ws-a")
	require.NoError(t, err)
	require.Equal(t, userLimitValues[v1pb.PlanType_FREE], limit)

	storeLicense := func(t *testing.T, params *LicenseParams) {
		t.Helper()
		token, err := licenseService.CreateLicense(params)
		require.NoError(t, err)
		require.NoError(t, s.UpdateLicense(ctx, "ws-a", token))
	}

	// Finite Enterprise license: the seat claim wins.
	storeLicense(t, &LicenseParams{
		Plan: v1pb.PlanType_ENTERPRISE.String(), Seats: 100, WorkspaceID: "ws-a",
	})
	limit, err = licenseService.GetUserLimitUncached(ctx, "ws-a")
	require.NoError(t, err)
	require.Equal(t, 100, limit)

	// Legacy Enterprise license without a seat claim: unlimited.
	storeLicense(t, &LicenseParams{
		Plan: v1pb.PlanType_ENTERPRISE.String(), Seats: 0, WorkspaceID: "ws-a",
	})
	limit, err = licenseService.GetUserLimitUncached(ctx, "ws-a")
	require.NoError(t, err)
	require.Equal(t, math.MaxInt, limit)

	// Expired Enterprise license: falls back to the Free plan limit.
	storeLicense(t, &LicenseParams{
		Plan: v1pb.PlanType_ENTERPRISE.String(), Seats: 100, WorkspaceID: "ws-a",
		ExpiresAt: time.Now().Add(-time.Hour),
	})
	limit, err = licenseService.GetUserLimitUncached(ctx, "ws-a")
	require.NoError(t, err)
	require.Equal(t, userLimitValues[v1pb.PlanType_FREE], limit)
}
