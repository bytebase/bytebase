# Token Duration Settings Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add separate configurable settings for access token and refresh token durations.

**Architecture:** Rename legacy `token_duration` to `refresh_token_duration` (since old "access tokens" behaved as refresh tokens), add new `access_token_duration` field. Both gated by existing `FEATURE_SIGN_IN_FREQUENCY_CONTROL` feature flag.

**Tech Stack:** Go, Protocol Buffers, Vue 3, TypeScript, PostgreSQL

---

## Task 1: Update Store Proto

**Files:**
- Modify: `proto/store/store/setting.proto:72-73`

**Step 1: Rename token_duration and add access_token_duration**

In `WorkspaceProfileSetting` message, change:

```protobuf
  // The duration for token.
  google.protobuf.Duration token_duration = 4;
```

To:

```protobuf
  // The duration for refresh token. Default is 7 days.
  google.protobuf.Duration refresh_token_duration = 4;

  // The duration for access token. Default is 1 hour.
  google.protobuf.Duration access_token_duration = 18;
```

---

## Task 2: Update V1 Proto

**Files:**
- Modify: `proto/v1/v1/setting_service.proto:182-183`

**Step 1: Rename token_duration and add access_token_duration**

In `WorkspaceProfileSetting` message, change:

```protobuf
  // The duration for token.
  google.protobuf.Duration token_duration = 4;
```

To:

```protobuf
  // The duration for refresh token. Default is 7 days.
  google.protobuf.Duration refresh_token_duration = 4;

  // The duration for access token. Default is 1 hour.
  google.protobuf.Duration access_token_duration = 18;
```

---

## Task 3: Format and Generate Protos

**Step 1: Format protos**

Run: `buf format -w proto`

**Step 2: Generate code**

Run: `cd proto && buf generate`

---

## Task 4: Add Database Migration

**Files:**
- Create: `backend/migrator/migration/3.14/0012##rename_token_duration_setting.sql`

**Step 1: Create migration file**

```sql
-- Rename tokenDuration to refreshTokenDuration in WORKSPACE_PROFILE setting
UPDATE setting
SET value = jsonb_set(
  value - 'tokenDuration',
  '{refreshTokenDuration}',
  value->'tokenDuration'
)
WHERE name = 'WORKSPACE_PROFILE'
  AND value ? 'tokenDuration';
```

---

## Task 5: Update Backend Auth Constants

**Files:**
- Modify: `backend/api/auth/auth.go:43-44`

**Step 1: Rename constant**

Change:

```go
	// DefaultTokenDuration is the default token expiration duration.
	DefaultTokenDuration = 7 * 24 * time.Hour
```

To:

```go
	// DefaultAccessTokenDuration is the default access token expiration duration.
	DefaultAccessTokenDuration = 1 * time.Hour
	// DefaultRefreshTokenDuration is the default refresh token expiration duration.
	DefaultRefreshTokenDuration = 7 * 24 * time.Hour
```

---

## Task 6: Update Backend Header Functions

**Files:**
- Modify: `backend/api/auth/header.go`

**Step 1: Rename GetTokenDuration to GetAccessTokenDuration**

Replace the entire `GetTokenDuration` function (lines 44-74) with:

```go
func GetAccessTokenDuration(ctx context.Context, store *store.Store, licenseService *enterprise.LicenseService) time.Duration {
	accessTokenDuration := DefaultAccessTokenDuration

	// If the sign-in frequency control feature is not enabled, return default duration
	if err := licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_SIGN_IN_FREQUENCY_CONTROL); err != nil {
		return accessTokenDuration
	}

	workspaceProfile, err := store.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		return accessTokenDuration
	}

	if workspaceProfile.GetAccessTokenDuration().GetSeconds() > 0 {
		accessTokenDuration = workspaceProfile.GetAccessTokenDuration().AsDuration()
	}

	return accessTokenDuration
}

func GetRefreshTokenDuration(ctx context.Context, store *store.Store, licenseService *enterprise.LicenseService) time.Duration {
	refreshTokenDuration := DefaultRefreshTokenDuration

	// If the sign-in frequency control feature is not enabled, return default duration
	if err := licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_SIGN_IN_FREQUENCY_CONTROL); err != nil {
		return refreshTokenDuration
	}

	workspaceProfile, err := store.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		return refreshTokenDuration
	}

	if workspaceProfile.GetRefreshTokenDuration().GetSeconds() > 0 {
		refreshTokenDuration = workspaceProfile.GetRefreshTokenDuration().AsDuration()
	}
	// Currently we implement the password rotation restriction in a simple way:
	// 1. Only check if users need to reset their password during login.
	// 2. For the 1st time login, if `RequireResetPasswordForFirstLogin` is true, `require_reset_password` in the response will be true
	// 3. Otherwise if the `PasswordRotation` exists, check the password last updated time to decide if the `require_reset_password` is true.
	// So we will use the minimum value between (`refreshTokenDuration`, `passwordRestriction.PasswordRotation`) to force to expire the token.
	passwordRestriction := workspaceProfile.GetPasswordRestriction()
	if passwordRestriction.GetPasswordRotation().GetSeconds() > 0 {
		passwordRotation := passwordRestriction.GetPasswordRotation().AsDuration()
		if passwordRotation.Seconds() < refreshTokenDuration.Seconds() {
			refreshTokenDuration = passwordRotation
		}
	}

	return refreshTokenDuration
}
```

**Step 2: Update GetTokenCookie to use new function**

In `GetTokenCookie` function (line 25), change:

```go
	tokenDuration := GetTokenDuration(ctx, stores, licenseService)
```

To:

```go
	tokenDuration := GetAccessTokenDuration(ctx, stores, licenseService)
```

---

## Task 7: Update Auth Service

**Files:**
- Modify: `backend/api/v1/auth_service.go`

**Step 1: Remove placeholder GetRefreshTokenDuration**

Delete lines 90-96 (the placeholder `GetRefreshTokenDuration` function):

```go
// GetRefreshTokenDuration returns the configured refresh token duration or default.
func GetRefreshTokenDuration(_ context.Context, _ *store.Store, _ *enterprise.LicenseService) time.Duration {
	// TODO: Add refresh_token_duration field to WorkspaceProfileSetting proto
	// and implement workspace setting-based configuration similar to GetTokenDuration.
	// For now, use a fixed 30-day duration.
	return 30 * 24 * time.Hour
}
```

**Step 2: Update generateLoginToken to use new function name**

In `generateLoginToken` function (line 781), change:

```go
	tokenDuration := auth.GetTokenDuration(ctx, s.store, s.licenseService)
```

To:

```go
	tokenDuration := auth.GetAccessTokenDuration(ctx, s.store, s.licenseService)
```

**Step 3: Update Refresh method to use auth.GetRefreshTokenDuration**

In `Refresh` method (line 265), change:

```go
	refreshTokenDuration := GetRefreshTokenDuration(ctx, s.store, s.licenseService)
```

To:

```go
	refreshTokenDuration := auth.GetRefreshTokenDuration(ctx, s.store, s.licenseService)
```

**Step 4: Update finalizeLogin to use auth.GetRefreshTokenDuration**

In `finalizeLogin` method (line 813), change:

```go
		refreshTokenDuration := GetRefreshTokenDuration(ctx, s.store, s.licenseService)
```

To:

```go
		refreshTokenDuration := auth.GetRefreshTokenDuration(ctx, s.store, s.licenseService)
```

---

## Task 8: Update Frontend Types

**Files:**
- Modify: `frontend/src/types/setting.ts`

**Step 1: Add access token duration constant**

Change:

```typescript
export const defaultTokenDurationInHours = 7 * 24;
```

To:

```typescript
export const defaultAccessTokenDurationInHours = 1;
export const defaultRefreshTokenDurationInHours = 7 * 24;
```

---

## Task 9: Update Frontend Component

**Files:**
- Modify: `frontend/src/components/GeneralSetting/SignInFrequencySetting.vue`

**Step 1: Update import**

Change:

```typescript
import { defaultTokenDurationInHours } from "@/types";
```

To:

```typescript
import { defaultAccessTokenDurationInHours, defaultRefreshTokenDurationInHours } from "@/types";
```

**Step 2: Update LocalState interface**

Change:

```typescript
interface LocalState {
  tokenDuration: number;
  inactiveTimeout: number;
  timeFormat: "HOURS" | "DAYS";
  showFeatureModal: boolean;
}
```

To:

```typescript
interface LocalState {
  accessTokenDuration: number;
  accessTokenTimeFormat: "MINUTES" | "HOURS";
  refreshTokenDuration: number;
  refreshTokenTimeFormat: "HOURS" | "DAYS";
  inactiveTimeout: number;
  showFeatureModal: boolean;
}
```

**Step 3: Update getInitialState function**

Replace the `getInitialState` function with:

```typescript
const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    accessTokenDuration: defaultAccessTokenDurationInHours * 60,
    accessTokenTimeFormat: "MINUTES",
    refreshTokenDuration: defaultRefreshTokenDurationInHours / 24,
    refreshTokenTimeFormat: "DAYS",
    inactiveTimeout: -1,
    showFeatureModal: false,
  };

  // Access token duration
  const accessTokenSeconds = settingV1Store.workspaceProfileSetting
    ?.accessTokenDuration?.seconds
    ? Number(settingV1Store.workspaceProfileSetting.accessTokenDuration.seconds)
    : undefined;
  if (accessTokenSeconds && accessTokenSeconds > 0) {
    if (accessTokenSeconds < 60 * 60) {
      defaultState.accessTokenDuration = Math.floor(accessTokenSeconds / 60) || 1;
      defaultState.accessTokenTimeFormat = "MINUTES";
    } else {
      defaultState.accessTokenDuration = Math.floor(accessTokenSeconds / (60 * 60)) || 1;
      defaultState.accessTokenTimeFormat = "HOURS";
    }
  }

  // Refresh token duration
  const refreshTokenSeconds = settingV1Store.workspaceProfileSetting
    ?.refreshTokenDuration?.seconds
    ? Number(settingV1Store.workspaceProfileSetting.refreshTokenDuration.seconds)
    : undefined;
  if (refreshTokenSeconds && refreshTokenSeconds > 0) {
    if (refreshTokenSeconds < 60 * 60 * 24) {
      defaultState.refreshTokenDuration = Math.floor(refreshTokenSeconds / (60 * 60)) || 1;
      defaultState.refreshTokenTimeFormat = "HOURS";
    } else {
      defaultState.refreshTokenDuration = Math.floor(refreshTokenSeconds / (60 * 60 * 24)) || 1;
      defaultState.refreshTokenTimeFormat = "DAYS";
    }
  }

  // Inactive timeout
  const inactiveTimeoutSeconds = Number(
    settingV1Store.workspaceProfileSetting?.inactiveSessionTimeout?.seconds ?? 0
  );
  if (inactiveTimeoutSeconds) {
    defaultState.inactiveTimeout = Math.floor(inactiveTimeoutSeconds / (60 * 60)) || 0;
  }

  return defaultState;
};
```

**Step 4: Add handlers for new settings**

Add after `handleInactivityTimeoutSettingChange`:

```typescript
const handleAccessTokenDurationChange = async () => {
  const seconds =
    state.accessTokenTimeFormat === "MINUTES"
      ? state.accessTokenDuration * 60
      : state.accessTokenDuration * 60 * 60;
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      accessTokenDuration: create(DurationSchema, {
        seconds: BigInt(seconds),
        nanos: 0,
      }),
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile.access_token_duration"],
    }),
  });
};

const handleRefreshTokenDurationChange = async () => {
  const seconds =
    state.refreshTokenTimeFormat === "HOURS"
      ? state.refreshTokenDuration * 60 * 60
      : state.refreshTokenDuration * 24 * 60 * 60;
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      refreshTokenDuration: create(DurationSchema, {
        seconds: BigInt(seconds),
        nanos: 0,
      }),
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile.refresh_token_duration"],
    }),
  });
};
```

**Step 5: Update handleUpdate function**

Replace `handleUpdate` with:

```typescript
const handleUpdate = async () => {
  const initState = getInitialState();

  if (initState.inactiveTimeout !== state.inactiveTimeout) {
    await handleInactivityTimeoutSettingChange();
  }

  if (
    initState.accessTokenDuration !== state.accessTokenDuration ||
    initState.accessTokenTimeFormat !== state.accessTokenTimeFormat
  ) {
    await handleAccessTokenDurationChange();
  }

  if (
    initState.refreshTokenDuration !== state.refreshTokenDuration ||
    initState.refreshTokenTimeFormat !== state.refreshTokenTimeFormat
  ) {
    await handleRefreshTokenDurationChange();
  }
};
```

**Step 6: Update watch for time format constraints**

Replace the watch with:

```typescript
watch(
  () => [state.accessTokenTimeFormat],
  () => {
    if (state.accessTokenTimeFormat === "MINUTES" && state.accessTokenDuration > 59) {
      state.accessTokenDuration = 59;
    }
  }
);

watch(
  () => [state.refreshTokenTimeFormat],
  () => {
    if (state.refreshTokenTimeFormat === "HOURS" && state.refreshTokenDuration > 23) {
      state.refreshTokenDuration = 23;
    }
  }
);
```

**Step 7: Delete handleFrequencySettingChange function**

Remove the old `handleFrequencySettingChange` function (no longer needed).

**Step 8: Update template**

Replace the first `<div class="mb-7 mt-4 lg:mt-0">` block (lines 2-42) with:

```vue
  <!-- Access Token Duration -->
  <div class="mb-7 mt-4 lg:mt-0" @click="handleValueFieldClick">
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">{{
        $t("settings.general.workspace.access-token-duration.self")
      }}</span>
      <FeatureBadge :feature="PlanFeature.FEATURE_SIGN_IN_FREQUENCY_CONTROL" />
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.access-token-duration.description") }}
    </p>
    <NTooltip placement="top-start" :disabled="allowChangeSetting">
      <template #trigger>
        <div class="mt-3 w-full flex flex-row justify-start items-center">
          <NInputNumber
            v-model:value="state.accessTokenDuration"
            class="w-24 mr-4"
            :disabled="!allowChangeSetting"
            :min="1"
            :max="state.accessTokenTimeFormat === 'MINUTES' ? 59 : 23"
            :precision="0"
          />
          <NRadioGroup
            v-model:value="state.accessTokenTimeFormat"
            :disabled="!allowChangeSetting"
          >
            <NRadio
              :value="'MINUTES'"
              :label="$t('settings.general.workspace.access-token-duration.minutes')"
            />
            <NRadio
              :value="'HOURS'"
              :label="$t('settings.general.workspace.access-token-duration.hours')"
            />
          </NRadioGroup>
        </div>
      </template>
      <span class="text-sm text-gray-400 -translate-y-2">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </NTooltip>
  </div>

  <!-- Refresh Token Duration -->
  <div class="mb-7 mt-4 lg:mt-0" @click="handleValueFieldClick">
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">{{
        $t("settings.general.workspace.refresh-token-duration.self")
      }}</span>
      <FeatureBadge :feature="PlanFeature.FEATURE_SIGN_IN_FREQUENCY_CONTROL" />
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.refresh-token-duration.description") }}
    </p>
    <NTooltip placement="top-start" :disabled="allowChangeSetting">
      <template #trigger>
        <div class="mt-3 w-full flex flex-row justify-start items-center">
          <NInputNumber
            v-model:value="state.refreshTokenDuration"
            class="w-24 mr-4"
            :disabled="!allowChangeSetting"
            :min="1"
            :max="state.refreshTokenTimeFormat === 'HOURS' ? 23 : undefined"
            :precision="0"
          />
          <NRadioGroup
            v-model:value="state.refreshTokenTimeFormat"
            :disabled="!allowChangeSetting"
          >
            <NRadio
              :value="'HOURS'"
              :label="$t('settings.general.workspace.refresh-token-duration.hours')"
            />
            <NRadio
              :value="'DAYS'"
              :label="$t('settings.general.workspace.refresh-token-duration.days')"
            />
          </NRadioGroup>
        </div>
      </template>
      <span class="text-sm text-gray-400 -translate-y-2">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </NTooltip>
  </div>
```

---

## Task 10: Update Locale Files

**Files:**
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/zh-CN.json`
- Modify: `frontend/src/locales/ja-JP.json`
- Modify: `frontend/src/locales/es-ES.json`
- Modify: `frontend/src/locales/vi-VN.json`

**Step 1: Update en-US.json**

Replace the `"sign-in-frequency"` section (around line 704) with:

```json
        "access-token-duration": {
          "self": "Access token duration",
          "description": "How long access tokens remain valid before requiring refresh. Shorter durations improve security.",
          "minutes": "Minute(s)",
          "hours": "Hour(s)"
        },
        "refresh-token-duration": {
          "self": "Refresh token duration",
          "description": "How often users must fully reauthenticate. Configuration updates require users to sign in again for changes to take effect.",
          "hours": "Hour(s)",
          "days": "Day(s)"
        },
```

**Step 2: Update zh-CN.json**

Replace the `"sign-in-frequency"` section with:

```json
        "access-token-duration": {
          "self": "访问令牌有效期",
          "description": "访问令牌在需要刷新前保持有效的时间。较短的有效期可提高安全性。",
          "minutes": "分钟",
          "hours": "小时"
        },
        "refresh-token-duration": {
          "self": "刷新令牌有效期",
          "description": "用户需要完全重新认证的频率。配置更新后需要用户重新登录才能生效。",
          "hours": "小时",
          "days": "天"
        },
```

**Step 3: Update other locale files similarly**

For `ja-JP.json`, `es-ES.json`, `vi-VN.json` - replace the `"sign-in-frequency"` section with equivalent `"access-token-duration"` and `"refresh-token-duration"` sections (can use English as placeholder if translations not immediately available).

---

## Task 11: Build and Lint

**Step 1: Format Go code**

Run: `gofmt -w backend/api/auth/auth.go backend/api/auth/header.go backend/api/v1/auth_service.go`

**Step 2: Run Go linter**

Run: `golangci-lint run --allow-parallel-runners`

Fix any issues reported.

**Step 3: Run frontend checks**

Run: `pnpm --dir frontend biome:check`
Run: `pnpm --dir frontend type-check`

---

## Task 12: Commit Changes

**Step 1: Check status**

Run: `but status`

**Step 2: Commit**

Run: `but commit token-duration-settings -m "feat(auth): add separate access and refresh token duration settings"`
