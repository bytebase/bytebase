# Unified Instance License Design

## Goal

Move new and equal-cap licenses to a one-number product experience while preserving the existing two-field license schema and legacy split-cap behavior.

The immediate product contract is: when every registrable instance can receive paid instance-scoped features, Bytebase should not expose an assignment step and should treat every registered instance as activated. This delivers the sales and UX simplification without a database migration, proto change, or JWT schema change.

## Context

Bytebase licenses currently carry two instance fields:

- `Claims.Instances` / JWT `instance`: registration cap.
- `Claims.ActiveInstances` / JWT `instanceCount`: activated or paid-feature instance cap.

Runtime code also stores per-instance activation in instance metadata. Several paid instance-scoped features, including data masking, read-only connections, custom sync time, and external secret manager, check this activation state.

The long-term goal is a single instance number. The first phase should not collapse the schema or migrate customers. It should recognize licenses where the effective registration cap is less than or equal to the effective activated cap, then make those licenses behave like one-number licenses.

## Product Contract

Define unified instance license mode as:

```text
effective registration cap <= effective activated cap
```

The comparison must use existing effective limit semantics, not raw JWT values:

- Backend registration cap: `LicenseService.GetInstanceLimit(ctx, workspaceID)`.
- Backend activated cap: `LicenseService.GetActivatedInstanceLimit(ctx, workspaceID)`.
- Frontend registration cap: `subscriptionStore.instanceCountLimit`.
- Frontend activated cap: `subscriptionStore.instanceLicenseCount`.

This handles zero and unlimited values consistently with existing plan and license fallback rules. For example, an unlimited registration cap and unlimited activated cap is unified; an unlimited registration cap with only 20 activated instances is legacy split-cap.

In unified mode:

- Every registered instance is effectively activated.
- Stored instance metadata is not mutated.
- Users should not see assignment-oriented entry points.
- If a workspace later receives a legacy split-cap license, the stored activation state becomes effective again.

In legacy split-cap mode:

- Existing runtime behavior remains.
- Existing UI and wording remain for this phase.
- Customers can still choose which instances receive paid instance-scoped features.

## Backend Behavior

Add a centralized helper on `LicenseService`:

```go
func (s *LicenseService) IsUnifiedInstanceLicense(ctx context.Context, workspaceID string) bool
```

It should return:

```text
GetInstanceLimit(ctx, workspaceID) <= GetActivatedInstanceLimit(ctx, workspaceID)
```

Use this helper in activation-sensitive paths.

Feature gates:

- `IsFeatureEnabledForInstance` still checks plan entitlement first.
- In unified mode, skip the stored `instance.Metadata.GetActivation()` check.
- In legacy split-cap mode, keep the existing activation check and error behavior.

Instance API responses:

- Public v1 instance responses should return `activation=true` in unified mode.
- The stored metadata remains unchanged.
- In legacy split-cap mode, return the stored activation state.

Actuator stats:

- In unified mode, return `activatedInstanceCount = totalInstanceCount`.
- In legacy split-cap mode, keep counting stored activated instances.

Activation quota checks:

- In unified mode, skip quota checks that specifically guard turning `activation` on.
- Instance creation remains protected by the registration cap through `GetInstanceLimit`.
- In legacy split-cap mode, keep current activation quota checks.

This phase should not change database schema, migration files, proto definitions, JWT parsing, or uploaded-license compatibility.

## Frontend Behavior

Frontend should compute presentation mode from effective store values:

```ts
const hasUnifiedInstanceLicense = computed(() => {
  return subscriptionStore.instanceCountLimit <= subscriptionStore.instanceLicenseCount;
});
```

Do not compare raw `subscription.instances` and `subscription.activeInstances` for presentation mode.

In unified mode:

- Hide assignment-oriented entry points such as "Assign License" buttons, assignment sheet triggers, and missing-license CTAs.
- Do not show "Assigned / Total Instance License" as the main subscription metric.
- Show a single instance quota based on `instanceCountLimit`.
- Omit or disable the activation toggle in instance forms because activation is effectively true.
- Ensure local feature guard helpers do not show missing-license states when the plan has the feature and the license is unified.

In legacy split-cap mode:

- Keep existing UI and copy unchanged.
- Assignment sheet, activation toggle, missing-license warnings, and activated/total metric remain available.
- Do not rename or redesign legacy assignment workflows in this phase.

Backend behavior remains authoritative. Frontend mode detection is for presentation and avoiding misleading assignment UI.

## License Issuance And Legacy Handling

Keep the license schema unchanged:

```go
ActiveInstances int `json:"instanceCount"`
Instances       int `json:"instance"`
```

New-license policy:

- New SaaS licenses should issue equal caps.
- `CreateLicense` should preserve the current behavior of setting both claims from the same `LicenseParams.Instances`.
- Add regression coverage so SaaS license generation does not accidentally diverge the two claims later.
- Manual and self-host license-generation tooling or process should issue equal caps for new licenses.

Legacy handling:

- Existing and uploaded unequal licenses remain valid.
- If effective registration cap is greater than effective activated cap, Bytebase remains in legacy split-cap behavior.
- Do not automatically mutate instance activation.
- Do not automatically shrink customer entitlement.
- Do not automatically expand customer entitlement.
- Renewal or migration to equal-count licenses happens through normal commercial or account workflow.

## Phases

Phase 1: effective unified mode.

- Add the normalized backend helper.
- Apply effective activation behavior in feature gates, instance API responses, actuator stats, and activation quota checks.
- Update frontend presentation to avoid assignment-oriented UI in unified mode.
- Keep schema, stored metadata, uploaded licenses, and split-cap behavior compatible.

Phase 2: issuance policy hardening.

- Ensure every supported new-license issuing path produces equal registration and activated caps.
- Document or update manual license-generation process for self-host licenses.
- Keep legacy unequal licenses accepted.

Phase 3: legacy drain.

- Identify workspaces or accounts still using split-cap licenses.
- Migrate them at renewal or through explicit account action.
- Avoid silent entitlement shrink or silent free expansion.

Phase 4: one-number cleanup.

- After split-cap licenses are gone, collapse product and code concepts to one instance count.
- Remove assignment UI and activation-based feature gating.
- Deprecate or ignore the old JWT field after a compatibility window.

This design covers Phase 1 and records the later direction. Phases 2 through 4 should be planned separately.

## Testing

Backend tests should cover:

- `IsUnifiedInstanceLicense` returns true for equal caps, more activated than registered, and unlimited/unlimited.
- `IsUnifiedInstanceLicense` returns false for split-cap licenses.
- `IsFeatureEnabledForInstance` allows instance-scoped features for stored `activation=false` instances in unified mode.
- The same feature check still fails in legacy split-cap mode when stored `activation=false`.
- Instance API responses return `activation=true` in unified mode without mutating stored metadata.
- Actuator info returns `activatedInstanceCount = totalInstanceCount` in unified mode.
- Activation quota checks are skipped only for unified mode.
- SaaS `CreateLicense` emits equal `instance` and `instanceCount` claims.

Frontend tests should cover:

- Unified mode is computed from effective values, not raw subscription fields.
- Missing-license warnings do not appear in unified mode.
- Assignment entry points are hidden in unified mode.
- Instance forms do not expose an actionable activation toggle in unified mode.
- Legacy split-cap mode keeps existing assignment UI.

Final implementation verification should follow `AGENTS.md`: run relevant Go formatting, linting, tests, build, and frontend fix/check/type-check/test commands for modified files.
