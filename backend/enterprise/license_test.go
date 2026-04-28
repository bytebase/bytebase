package enterprise

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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
