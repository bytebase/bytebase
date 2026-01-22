# Frontend API Separation Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor frontend store layer to route API calls to the correct backend service (UserService, ServiceAccountService, WorkloadIdentityService) based on user type.

**Architecture:** The backend has separated User, Service Account, and Workload Identity into distinct services. The frontend `user.ts` store currently calls `userServiceClientConnect` for all operations, but now it must route to the appropriate service based on `UserType`. This is a pure refactoring - no UI changes.

**Tech Stack:** TypeScript, Vue 3, Pinia, Connect-Web

---

## Background

**Backend API Changes:**
- `UserService.CreateUser` now only accepts `UserType.USER` (END_USER)
- `UserService.ListUsers` only returns END_USER type
- `ServiceAccountService` handles all SERVICE_ACCOUNT operations
- `WorkloadIdentityService` handles all WORKLOAD_IDENTITY operations

**Frontend Impact:**
- `CreateUserDrawer.vue` calls `userStore.createUser()` for all types
- `UserOperationsCell.vue` calls `userStore.archiveUser/restoreUser`
- `UserDataTable/index.vue` calls `userStore.updateUser` for service key reset

---

### Task 1: Add New Service Client Imports to User Store

**Files:**
- Modify: `frontend/src/store/modules/user.ts:1-35`

**Step 1: Add imports for new services**

At the top of the file, after existing imports, add:

```typescript
import {
  serviceAccountServiceClientConnect,
  workloadIdentityServiceClientConnect,
} from "@/connect";
import {
  CreateServiceAccountRequestSchema,
  DeleteServiceAccountRequestSchema,
  ServiceAccountSchema,
  UndeleteServiceAccountRequestSchema,
  UpdateServiceAccountRequestSchema,
} from "@/types/proto-es/v1/service_account_service_pb";
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import {
  CreateWorkloadIdentityRequestSchema,
  DeleteWorkloadIdentityRequestSchema,
  UndeleteWorkloadIdentityRequestSchema,
  UpdateWorkloadIdentityRequestSchema,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
```

**Step 2: Run frontend check**

Run: `pnpm --dir frontend check`
Expected: No errors (imports are valid)

---

### Task 2: Add Helper Functions to Convert Between Types

**Files:**
- Modify: `frontend/src/store/modules/user.ts` (add after line 63, before `useUserStore`)

**Step 1: Add conversion helpers**

Add these helper functions to convert ServiceAccount/WorkloadIdentity to User type:

```typescript
// Helper to convert ServiceAccount to User for internal store consistency
const serviceAccountToUser = (sa: ServiceAccount): User => {
  return create(UserSchema, {
    name: `users/${sa.email}`,
    email: sa.email,
    title: sa.title,
    state: sa.state,
    userType: UserType.SERVICE_ACCOUNT,
    serviceKey: sa.serviceKey,
  });
};

// Helper to convert WorkloadIdentity to User for internal store consistency
const workloadIdentityToUser = (wi: WorkloadIdentity): User => {
  return create(UserSchema, {
    name: `users/${wi.email}`,
    email: wi.email,
    title: wi.title,
    state: wi.state,
    userType: UserType.WORKLOAD_IDENTITY,
    workloadIdentityConfig: wi.workloadIdentityConfig,
  });
};

// Extract email prefix from full email for service account/workload identity creation
const extractEmailPrefix = (email: string, suffix: string): string => {
  if (email.endsWith(suffix)) {
    return email.slice(0, -suffix.length);
  }
  return email.split("@")[0];
};
```

**Step 2: Add UserSchema import**

Update the import from user_service_pb to include UserSchema:

```typescript
import {
  BatchGetUsersRequestSchema,
  CreateUserRequestSchema,
  DeleteUserRequestSchema,
  GetUserRequestSchema,
  ListUsersRequestSchema,
  UndeleteUserRequestSchema,
  UpdateUserRequestSchema,
  UserSchema,
  UserType,
} from "@/types/proto-es/v1/user_service_pb";
```

**Step 3: Run frontend check**

Run: `pnpm --dir frontend check`
Expected: No errors

---

### Task 3: Refactor createUser to Route by Type

**Files:**
- Modify: `frontend/src/store/modules/user.ts:126-133`

**Step 1: Replace createUser function**

Replace the existing `createUser` function with:

```typescript
  const createUser = async (user: User) => {
    let createdUser: User;

    if (user.userType === UserType.SERVICE_ACCOUNT) {
      // Route to ServiceAccountService
      const emailPrefix = extractEmailPrefix(user.email, "@service.bytebase.com");
      const request = create(CreateServiceAccountRequestSchema, {
        parent: "", // workspace-level
        serviceAccountId: emailPrefix,
        serviceAccount: create(ServiceAccountSchema, {
          title: user.title || emailPrefix,
        }),
      });
      const response = await serviceAccountServiceClientConnect.createServiceAccount(request);
      createdUser = serviceAccountToUser(response);
    } else if (user.userType === UserType.WORKLOAD_IDENTITY) {
      // Route to WorkloadIdentityService
      const emailPrefix = extractEmailPrefix(user.email, "@workload.bytebase.com");
      const request = create(CreateWorkloadIdentityRequestSchema, {
        parent: "", // workspace-level
        workloadIdentityId: emailPrefix,
        workloadIdentity: create(WorkloadIdentitySchema, {
          title: user.title || emailPrefix,
          workloadIdentityConfig: user.workloadIdentityConfig,
        }),
      });
      const response = await workloadIdentityServiceClientConnect.createWorkloadIdentity(request);
      createdUser = workloadIdentityToUser(response);
    } else {
      // END_USER - use UserService
      const request = create(CreateUserRequestSchema, {
        user: user,
      });
      const response = await userServiceClientConnect.createUser(request);
      createdUser = response;
    }

    await actuatorStore.fetchServerInfo();
    return setUser(createdUser);
  };
```

**Step 2: Run frontend check**

Run: `pnpm --dir frontend check`
Expected: No errors

---

### Task 4: Refactor archiveUser to Route by Type

**Files:**
- Modify: `frontend/src/store/modules/user.ts:164-172`

**Step 1: Replace archiveUser function**

Replace the existing `archiveUser` function with:

```typescript
  const archiveUser = async (user: User) => {
    if (user.userType === UserType.SERVICE_ACCOUNT) {
      const request = create(DeleteServiceAccountRequestSchema, {
        name: `serviceAccounts/${user.email}`,
      });
      await serviceAccountServiceClientConnect.deleteServiceAccount(request);
    } else if (user.userType === UserType.WORKLOAD_IDENTITY) {
      const request = create(DeleteWorkloadIdentityRequestSchema, {
        name: `workloadIdentities/${user.email}`,
      });
      await workloadIdentityServiceClientConnect.deleteWorkloadIdentity(request);
    } else {
      const request = create(DeleteUserRequestSchema, {
        name: user.name,
      });
      await userServiceClientConnect.deleteUser(request);
    }
    user.state = State.DELETED;
    await actuatorStore.fetchServerInfo();
    return user;
  };
```

**Step 2: Run frontend check**

Run: `pnpm --dir frontend check`
Expected: No errors

---

### Task 5: Refactor restoreUser to Route by Type

**Files:**
- Modify: `frontend/src/store/modules/user.ts:174-181`

**Step 1: Replace restoreUser function**

Replace the existing `restoreUser` function with:

```typescript
  const restoreUser = async (user: User) => {
    let restoredUser: User;

    if (user.userType === UserType.SERVICE_ACCOUNT) {
      const request = create(UndeleteServiceAccountRequestSchema, {
        name: `serviceAccounts/${user.email}`,
      });
      const response = await serviceAccountServiceClientConnect.undeleteServiceAccount(request);
      restoredUser = serviceAccountToUser(response);
    } else if (user.userType === UserType.WORKLOAD_IDENTITY) {
      const request = create(UndeleteWorkloadIdentityRequestSchema, {
        name: `workloadIdentities/${user.email}`,
      });
      const response = await workloadIdentityServiceClientConnect.undeleteWorkloadIdentity(request);
      restoredUser = workloadIdentityToUser(response);
    } else {
      const request = create(UndeleteUserRequestSchema, {
        name: user.name,
      });
      const response = await userServiceClientConnect.undeleteUser(request);
      restoredUser = response;
    }

    await actuatorStore.fetchServerInfo();
    return setUser(restoredUser);
  };
```

**Step 2: Run frontend check**

Run: `pnpm --dir frontend check`
Expected: No errors

---

### Task 6: Refactor updateUser to Route by Type for Service Key Reset

**Files:**
- Modify: `frontend/src/store/modules/user.ts:135-150`

**Step 1: Replace updateUser function**

Replace the existing `updateUser` function with:

```typescript
  const updateUser = async (updateUserRequest: UpdateUserRequest) => {
    const name = updateUserRequest.user?.name || "";
    const originData = await getOrFetchUserByIdentifier(name);
    if (!originData) {
      throw new Error(`user with name ${name} not found`);
    }

    // Check if this is a service account update (service_key rotation)
    if (originData.userType === UserType.SERVICE_ACCOUNT && updateUserRequest.updateMask?.paths.includes("service_key")) {
      const request = create(UpdateServiceAccountRequestSchema, {
        serviceAccount: create(ServiceAccountSchema, {
          name: `serviceAccounts/${originData.email}`,
          title: updateUserRequest.user?.title || originData.title,
        }),
        updateMask: updateUserRequest.updateMask,
      });
      const response = await serviceAccountServiceClientConnect.updateServiceAccount(request);
      return setUser(serviceAccountToUser(response));
    }

    // Check if this is a workload identity update
    if (originData.userType === UserType.WORKLOAD_IDENTITY) {
      const request = create(UpdateWorkloadIdentityRequestSchema, {
        workloadIdentity: create(WorkloadIdentitySchema, {
          name: `workloadIdentities/${originData.email}`,
          title: updateUserRequest.user?.title || originData.title,
          workloadIdentityConfig: updateUserRequest.user?.workloadIdentityConfig,
        }),
        updateMask: updateUserRequest.updateMask,
      });
      const response = await workloadIdentityServiceClientConnect.updateWorkloadIdentity(request);
      return setUser(workloadIdentityToUser(response));
    }

    // END_USER - use UserService
    const request = create(UpdateUserRequestSchema, {
      user: updateUserRequest.user,
      updateMask: updateUserRequest.updateMask,
      otpCode: updateUserRequest.otpCode,
      regenerateTempMfaSecret: updateUserRequest.regenerateTempMfaSecret,
      regenerateRecoveryCodes: updateUserRequest.regenerateRecoveryCodes,
    });
    const response = await userServiceClientConnect.updateUser(request);
    return setUser(response);
  };
```

**Step 2: Run frontend check**

Run: `pnpm --dir frontend check`
Expected: No errors

---

### Task 7: Final Verification and Commit

**Step 1: Run frontend type-check**

Run: `pnpm --dir frontend type-check`
Expected: No errors

**Step 2: Run frontend check**

Run: `pnpm --dir frontend check`
Expected: No errors

**Step 3: Test backend build**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds

**Step 4: Commit changes**

```bash
git add frontend/src/store/modules/user.ts
git commit -m "$(cat <<'EOF'
refactor(frontend): route API calls by user type to correct backend service

- Route createUser to ServiceAccountService/WorkloadIdentityService based on UserType
- Route archiveUser/restoreUser to correct service based on UserType
- Route updateUser (service_key reset) to ServiceAccountService
- Add conversion helpers between ServiceAccount/WorkloadIdentity and User types
- Add imports for new service clients and request schemas

This completes the frontend API separation refactor to align with backend changes.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Add service client imports | `user.ts` |
| 2 | Add type conversion helpers | `user.ts` |
| 3 | Refactor createUser | `user.ts` |
| 4 | Refactor archiveUser | `user.ts` |
| 5 | Refactor restoreUser | `user.ts` |
| 6 | Refactor updateUser | `user.ts` |
| 7 | Final verification & commit | `user.ts` |

**Key Points:**
- Only `frontend/src/store/modules/user.ts` needs modification
- No UI changes required
- The store acts as a router, directing API calls to the correct backend service
- Conversion helpers ensure internal consistency by converting responses to User type
