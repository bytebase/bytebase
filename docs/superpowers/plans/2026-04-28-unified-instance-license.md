# Unified Instance License Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make licenses whose effective registration cap is less than or equal to their effective activated cap behave and present as one-number instance licenses.

**Architecture:** Add one centralized backend license-mode helper, then use it to compute effective activation without mutating stored instance metadata. Mirror the same effective-limit rule in the frontend subscription store so feature guards and settings pages hide assignment-oriented UI in unified mode while legacy split-cap licenses keep current behavior.

**Tech Stack:** Go backend services and tests, Connect v1 API, Pinia/Vue subscription store, React settings/components, Vitest frontend tests.

---

## File Structure

- Modify `backend/enterprise/license.go`: add `IsUnifiedInstanceLicense` and make `IsFeatureEnabledForInstance` use effective activation.
- Add `backend/enterprise/license_test.go`: unit coverage for unified limit comparison, feature gating, and `CreateLicense` equal-count claims.
- Modify `backend/api/v1/instance_service.go`: skip activation quota checks in unified mode and return effective activation for instance service responses.
- Modify `backend/api/v1/instance_service_converter.go`: add a small helper to override response activation without changing store metadata.
- Modify `backend/api/v1/actuator_service.go`: report all registered instances as activated in unified mode.
- Modify `frontend/src/store/modules/v1/subscription.ts`: add `hasUnifiedInstanceLicense` and use it in instance feature/missing-license helpers.
- Modify `frontend/src/react/pages/settings/SubscriptionPage.tsx`: show one-number instance quota and hide assignment sheet trigger in unified mode.
- Modify `frontend/src/react/components/FeatureAttention.tsx`: prevent assignment sheet rendering/action in unified mode through store helpers.
- Modify `frontend/src/react/components/instance/InstanceFormBody.tsx`: hide the activation toggle in unified mode.
- Add focused frontend tests in `frontend/src/react/components/FeatureAttention.test.tsx` and a new `frontend/src/store/modules/v1/subscription.test.ts`.

## Task 1: Backend Unified License Helper

**Files:**
- Modify: `backend/enterprise/license.go`
- Add: `backend/enterprise/license_test.go`

- [ ] **Step 1: Write failing tests for the normalized limit rule**

Add `backend/enterprise/license_test.go` in package `enterprise` with table tests for a pure helper:

```go
package enterprise

import (
	"math"
	"testing"
)

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
```

- [ ] **Step 2: Run the failing backend unit test**

Run:

```bash
go test -v -count=1 ./backend/enterprise -run ^TestIsUnifiedInstanceLimit$
```

Expected: fail because `isUnifiedInstanceLimit` is undefined.

- [ ] **Step 3: Implement the helper**

In `backend/enterprise/license.go`, add:

```go
func isUnifiedInstanceLimit(instanceLimit, activatedInstanceLimit int) bool {
	return instanceLimit <= activatedInstanceLimit
}

// IsUnifiedInstanceLicense returns whether every registrable instance is effectively activated.
func (s *LicenseService) IsUnifiedInstanceLicense(ctx context.Context, workspaceID string) bool {
	return isUnifiedInstanceLimit(
		s.GetInstanceLimit(ctx, workspaceID),
		s.GetActivatedInstanceLimit(ctx, workspaceID),
	)
}
```

- [ ] **Step 4: Run the helper test**

Run:

```bash
go test -v -count=1 ./backend/enterprise -run ^TestIsUnifiedInstanceLimit$
```

Expected: pass.

- [ ] **Step 5: Commit Task 1**

```bash
gofmt -w backend/enterprise/license.go backend/enterprise/license_test.go
git add backend/enterprise/license.go backend/enterprise/license_test.go
git commit -m "feat: add unified instance license helper"
```

## Task 2: Backend Effective Feature Activation

**Files:**
- Modify: `backend/enterprise/license.go`
- Modify: `backend/enterprise/license_test.go`

- [ ] **Step 1: Write failing tests for instance feature gating**

Extend `backend/enterprise/license_test.go` with tests that construct a `LicenseService` with cached subscriptions. Use package `enterprise` so tests can seed the cache directly:

```go
import (
	"context"
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

func TestIsFeatureEnabledForInstanceUnifiedLicense(t *testing.T) {
	ctx := context.Background()
	instance := &store.InstanceMessage{
		ResourceID: "prod",
		Workspace:  "test-workspace",
		Metadata: &storepb.Instance{
			Activation: false,
		},
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
		Metadata: &storepb.Instance{
			Activation: false,
		},
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
```

- [ ] **Step 2: Run the failing feature-gate tests**

Run:

```bash
go test -v -count=1 ./backend/enterprise -run '^(TestIsFeatureEnabledForInstanceUnifiedLicense|TestIsFeatureEnabledForInstanceSplitLicense)$'
```

Expected: unified-license test fails because the implementation still requires stored activation.

- [ ] **Step 3: Implement effective activation in `IsFeatureEnabledForInstance`**

In `backend/enterprise/license.go`, change the final activation check:

```go
	if s.IsUnifiedInstanceLicense(ctx, workspaceID) {
		return nil
	}
	if !instance.Metadata.GetActivation() {
		return errors.Errorf(`feature "%s" is not available for instance %s, please assign license to the instance to enable it`, f.String(), instance.ResourceID)
	}
	return nil
```

- [ ] **Step 4: Add and run `CreateLicense` equal-claim regression test**

Extract the literal `Claims` construction in `CreateLicense` into:

```go
func newLicenseClaims(params *LicenseParams) *Claims {
	return &Claims{
		Plan:            params.Plan,
		Seats:           params.Seats,
		ActiveInstances: params.Instances,
		Instances:       params.Instances,
		WorkspaceID:     params.WorkspaceID,
	}
}
```

Then have `CreateLicense` call:

```go
c := newLicenseClaims(params)
```

Add this regression test:

```go
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
```

- [ ] **Step 5: Run backend enterprise tests**

Run:

```bash
go test -v -count=1 ./backend/enterprise
```

Expected: pass.

- [ ] **Step 6: Commit Task 2**

```bash
gofmt -w backend/enterprise/license.go backend/enterprise/license_test.go
git add backend/enterprise/license.go backend/enterprise/license_test.go
git commit -m "feat: treat unified instance licenses as activated"
```

## Task 3: Backend Instance And Actuator API Output

**Files:**
- Modify: `backend/api/v1/instance_service_converter.go`
- Modify: `backend/api/v1/instance_service.go`
- Modify: `backend/api/v1/actuator_service.go`

- [ ] **Step 1: Add an effective activation converter helper**

In `backend/api/v1/instance_service_converter.go`, add:

```go
func convertToV1InstanceWithEffectiveActivation(instance *store.InstanceMessage, effectiveActivation bool) *v1pb.Instance {
	result := convertToV1Instance(instance)
	result.Activation = effectiveActivation
	return result
}
```

- [ ] **Step 2: Add an `InstanceService` conversion method**

In `backend/api/v1/instance_service.go`, near helper methods, add:

```go
func (s *InstanceService) convertToV1Instance(ctx context.Context, instance *store.InstanceMessage) *v1pb.Instance {
	if s.licenseService.IsUnifiedInstanceLicense(ctx, common.GetWorkspaceIDFromContext(ctx)) {
		return convertToV1InstanceWithEffectiveActivation(instance, true)
	}
	return convertToV1Instance(instance)
}
```

- [ ] **Step 3: Replace instance service response conversions**

In `backend/api/v1/instance_service.go`, replace response call sites such as:

```go
result := convertToV1Instance(instance)
```

with:

```go
result := s.convertToV1Instance(ctx, instance)
```

Also replace list appends:

```go
ins := convertToV1Instance(instance)
```

with:

```go
ins := s.convertToV1Instance(ctx, instance)
```

Do not change `convertToStoreInstance`; create/update requests should still preserve incoming stored activation.

- [ ] **Step 4: Skip activation quota checks in unified mode**

In create/update activation quota checks in `backend/api/v1/instance_service.go`, guard the check:

```go
if instanceMessage.Metadata.GetActivation() && !s.licenseService.IsUnifiedInstanceLicense(ctx, workspaceID) {
	activatedInstanceLimit := s.licenseService.GetActivatedInstanceLimit(ctx, workspaceID)
	count, err := s.store.GetActivatedInstanceCount(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if count >= activatedInstanceLimit {
		return nil, connect.NewError(connect.CodeResourceExhausted, errors.Errorf(instanceExceededError, activatedInstanceLimit))
	}
}
```

For update:

```go
if updateActivation && !s.licenseService.IsUnifiedInstanceLicense(ctx, workspaceID) {
	activatedInstanceLimit := s.licenseService.GetActivatedInstanceLimit(ctx, workspaceID)
	count, err := s.store.GetActivatedInstanceCount(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if count >= activatedInstanceLimit {
		return nil, connect.NewError(connect.CodeResourceExhausted, errors.Errorf(instanceExceededError, activatedInstanceLimit))
	}
}
```

Remove the existing unconditional `activatedInstanceLimit := ...` lines near these blocks after moving the assignment inside the guarded branch.

- [ ] **Step 5: Update actuator stats**

In `backend/api/v1/actuator_service.go`, after computing `activeInstanceCount`, set activation count by mode:

```go
activeInstanceCount, err := s.store.CountActiveInstances(ctx, workspaceID)
if err != nil {
	return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to count total instance"))
}
serverInfo.TotalInstanceCount = int32(activeInstanceCount)

if s.licenseService.IsUnifiedInstanceLicense(ctx, workspaceID) {
	serverInfo.ActivatedInstanceCount = int32(activeInstanceCount)
} else {
	activatedInstanceCount, err := s.store.GetActivatedInstanceCount(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to count activated instance"))
	}
	serverInfo.ActivatedInstanceCount = int32(activatedInstanceCount)
}
```

Preserve nearby unrelated actuator fields.

- [ ] **Step 6: Run targeted compile tests**

Run:

```bash
go test -v -count=1 ./backend/api/v1 -run '^(TestNonExistent)$'
```

Expected: package compiles and reports no tests to run or passes existing package setup. If this package requires external services and cannot run cleanly, use:

```bash
go test -run '^$' ./backend/api/v1
```

Expected: compile succeeds.

- [ ] **Step 7: Commit Task 3**

```bash
gofmt -w backend/api/v1/instance_service_converter.go backend/api/v1/instance_service.go backend/api/v1/actuator_service.go
git add backend/api/v1/instance_service_converter.go backend/api/v1/instance_service.go backend/api/v1/actuator_service.go
git commit -m "feat: expose effective activation for unified licenses"
```

## Task 4: Frontend Store Unified Mode

**Files:**
- Modify: `frontend/src/store/modules/v1/subscription.ts`

- [ ] **Step 1: Add the computed mode**

In `frontend/src/store/modules/v1/subscription.ts`, after `instanceLicenseCount`, add:

```ts
  const hasUnifiedInstanceLicense = computed(() => {
    return instanceCountLimit.value <= instanceLicenseCount.value;
  });
```

- [ ] **Step 2: Use unified mode in feature helpers**

Update `hasInstanceFeature`:

```ts
    return checkInstanceFeature(
      currentPlan.value,
      feature,
      hasUnifiedInstanceLicense.value || instance.activation
    );
```

Update `instanceMissingLicense`:

```ts
    if (hasUnifiedInstanceLicense.value) {
      return false;
    }
    return hasFeature(feature) && !instance.activation;
```

- [ ] **Step 3: Return the computed value from the store**

Add `hasUnifiedInstanceLicense` to the returned getters:

```ts
    hasUnifiedInstanceLicense,
```

- [ ] **Step 4: Run frontend type check for the changed store**

Run:

```bash
pnpm --dir frontend type-check
```

Expected: pass. If unrelated type errors already exist, capture the first unrelated error and continue only after confirming it is not caused by this change.

- [ ] **Step 5: Commit Task 4**

```bash
git add frontend/src/store/modules/v1/subscription.ts
git commit -m "feat(frontend): derive unified instance license mode"
```

## Task 5: Frontend Presentation Updates

**Files:**
- Modify: `frontend/src/react/pages/settings/SubscriptionPage.tsx`
- Modify: `frontend/src/react/components/FeatureAttention.tsx`
- Modify: `frontend/src/react/components/instance/InstanceFormBody.tsx`
- Do not modify `frontend/src/components/WorkspaceInstanceLicenseStats.vue`; `rg "WorkspaceInstanceLicenseStats" frontend/src` shows no active references in this worktree.

- [ ] **Step 1: Update subscription page stats**

In `frontend/src/react/pages/settings/SubscriptionPage.tsx`, read the store mode:

```ts
  const hasUnifiedInstanceLicense = useVueState(
    () => subscriptionStore.hasUnifiedInstanceLicense
  );
```

Pass it into `InstanceLicenseStats`:

```tsx
            hasUnifiedInstanceLicense={hasUnifiedInstanceLicense}
```

Update props and render one-number stats when free or unified:

```tsx
function InstanceLicenseStats({
  planType,
  hasUnifiedInstanceLicense,
  instanceCountLimit,
  activatedCount,
  totalLicenseCount,
  onManageInstanceLicenses,
}: {
  planType: string;
  hasUnifiedInstanceLicense: boolean;
  instanceCountLimit: number;
  activatedCount: number;
  totalLicenseCount: string;
  onManageInstanceLicenses: () => void;
}) {
  const { t } = useTranslation();

  if (planType === "FREE" || hasUnifiedInstanceLicense) {
    return (
      <div className="flex flex-col text-left">
        <dt className="text-main">{t("subscription.max-instance-count")}</dt>
        <div className="mt-1 text-4xl">{instanceCountLimit}</div>
      </div>
    );
  }
```

Only render `InstanceAssignmentSheet` when not unified:

```tsx
      {!hasUnifiedInstanceLicense && (
        <InstanceAssignmentSheet
          open={showInstanceAssignmentSheet}
          onOpenChange={setShowInstanceAssignmentSheet}
        />
      )}
```

- [ ] **Step 2: Update feature attention assignment paths**

In `frontend/src/react/components/FeatureAttention.tsx`, read the mode:

```ts
  const hasUnifiedInstanceLicense = useVueState(
    () => subscriptionStore.hasUnifiedInstanceLicense
  );
```

Update missing-license existence:

```ts
  const existInstanceWithoutLicense = useVueState(
    () =>
      !subscriptionStore.hasUnifiedInstanceLicense &&
      actuatorStore.totalInstanceCount > actuatorStore.activatedInstanceCount &&
      instanceLimitFeature.has(feature)
  );
```

Only render the assignment sheet when not unified:

```tsx
      {!hasUnifiedInstanceLicense && (
        <InstanceAssignmentSheet
          open={showInstanceAssignment}
          selectedInstanceList={instance ? [instance.name] : []}
          onOpenChange={setShowInstanceAssignment}
        />
      )}
```

- [ ] **Step 3: Hide instance activation toggle in unified mode**

In `frontend/src/react/components/instance/InstanceFormBody.tsx`, add a store mode read near existing subscription values:

```ts
  const hasUnifiedInstanceLicense = subscriptionStore.hasUnifiedInstanceLicense;
```

Update the activation toggle condition:

```tsx
            {subscriptionStore.currentPlan !== PlanType.FREE &&
              !hasUnifiedInstanceLicense &&
              allowEdit && (
```

- [ ] **Step 4: Run frontend checks**

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend type-check
pnpm --dir frontend test -- FeatureAttention
```

Expected: formatting succeeds, type-check passes, and the targeted FeatureAttention tests pass.

- [ ] **Step 5: Commit Task 5**

```bash
git add frontend/src/store/modules/v1/subscription.ts frontend/src/react/pages/settings/SubscriptionPage.tsx frontend/src/react/components/FeatureAttention.tsx frontend/src/react/components/instance/InstanceFormBody.tsx
git commit -m "feat(frontend): hide license assignment in unified mode"
```

## Task 6: Focused Regression Tests

**Files:**
- Modify: `frontend/src/react/components/FeatureAttention.test.tsx`
- Add: `frontend/src/store/modules/v1/subscription.test.ts`

- [ ] **Step 1: Add FeatureAttention unified-mode test**

In `frontend/src/react/components/FeatureAttention.test.tsx`, extend the mocked subscription store with:

```ts
hasUnifiedInstanceLicense: false,
```

Add a test:

```tsx
test("does not show assignment attention in unified instance license mode", () => {
  mocks.hasFeature.mockReturnValue(true);
  mocks.instanceMissingLicense.mockReturnValue(false);
  mocks.hasUnifiedInstanceLicense = true;
  mocks.totalInstanceCount = 2;
  mocks.activatedInstanceCount = 2;

  render(<FeatureAttention feature={PlanFeature.FEATURE_DATA_MASKING} />);

  expect(screen.queryByText("subscription.instance-assignment.assign-license")).not.toBeInTheDocument();
});
```

Use the existing mock object in `FeatureAttention.test.tsx`; add the `hasUnifiedInstanceLicense` field beside the existing mocked subscription store methods and reset it to `false` in `beforeEach`.

- [ ] **Step 2: Add store helper tests**

Create `frontend/src/store/modules/v1/subscription.test.ts`:

```ts
import { create } from "@bufbuild/protobuf";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, test } from "vitest";
import { useSubscriptionV1Store } from "./subscription";
import { InstanceSchema } from "@/types/proto-es/v1/instance_service_pb";
import {
  PlanFeature,
  PlanType,
  SubscriptionSchema,
} from "@/types/proto-es/v1/subscription_service_pb";

describe("useSubscriptionV1Store unified instance license", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  test("computes unified mode from effective limits", () => {
    const store = useSubscriptionV1Store();
    store.setSubscription(
      create(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        instances: 10,
        activeInstances: 10,
      })
    );

    expect(store.hasUnifiedInstanceLicense).toBe(true);

    store.setSubscription(
      create(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        instances: 50,
        activeInstances: 20,
      })
    );

    expect(store.hasUnifiedInstanceLicense).toBe(false);
  });

  test("does not report missing instance license in unified mode", () => {
    const store = useSubscriptionV1Store();
    store.setSubscription(
      create(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        instances: 10,
        activeInstances: 10,
      })
    );

    const inactiveInstance = create(InstanceSchema, {
      name: "instances/prod",
      title: "prod",
      activation: false,
    });

    expect(
      store.instanceMissingLicense(
        PlanFeature.FEATURE_DATA_MASKING,
        inactiveInstance
      )
    ).toBe(false);
  });
});
```

- [ ] **Step 3: Run targeted frontend tests**

Run:

```bash
pnpm --dir frontend test -- FeatureAttention
pnpm --dir frontend test -- subscription.test
```

Expected: targeted tests pass.

- [ ] **Step 4: Commit Task 6**

```bash
git add frontend/src/react/components/FeatureAttention.test.tsx frontend/src/store/modules/v1/subscription.test.ts
git commit -m "test: cover unified instance license presentation"
```

## Task 7: Final Verification

**Files:**
- No source edits expected.

- [ ] **Step 1: Run Go formatting**

Run:

```bash
gofmt -w backend/enterprise/license.go backend/enterprise/license_test.go backend/api/v1/instance_service_converter.go backend/api/v1/instance_service.go backend/api/v1/actuator_service.go
```

Expected: no output.

- [ ] **Step 2: Run backend tests**

Run:

```bash
go test -v -count=1 ./backend/enterprise
go test -run '^$' ./backend/api/v1
```

Expected: pass or compile successfully for `backend/api/v1`.

- [ ] **Step 3: Run linter repeatedly until clean**

Run:

```bash
golangci-lint run --allow-parallel-runners
```

Expected: no issues. If issues are reported, run:

```bash
golangci-lint run --fix --allow-parallel-runners
golangci-lint run --allow-parallel-runners
```

Repeat until no issues remain.

- [ ] **Step 4: Run frontend validation**

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
```

Expected: pass.

- [ ] **Step 5: Run backend build**

Run:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: build succeeds.

- [ ] **Step 6: Final status check**

Run:

```bash
git status --short
git log --oneline -5
```

Expected: only intended source/test changes are present or committed. Unrelated pre-existing untracked files remain untouched.
