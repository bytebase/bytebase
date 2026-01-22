# API Separation: Service Registration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Complete the API layer separation by registering ServiceAccountService and WorkloadIdentityService in the backend server and adding frontend Connect clients.

**Architecture:** The services are already fully implemented (`service_account_service.go`, `workload_identity_service.go`) but not registered. We need to: (1) instantiate and register them in `grpc_routes.go`, (2) add frontend Connect clients in `frontend/src/connect/index.ts`.

**Tech Stack:** Go (Connect RPC), TypeScript (Connect-Web)

---

### Task 1: Register ServiceAccountService in Backend

**Files:**
- Modify: `backend/server/grpc_routes.go:114-116` (service instantiation)
- Modify: `backend/server/grpc_routes.go:212-219` (handler registration)
- Modify: `backend/server/grpc_routes.go:222-250` (reflection list)
- Modify: `backend/server/grpc_routes.go:342-350` (REST gateway)

**Step 1: Add ServiceAccountService instantiation**

After line 114 (after `userService := ...`), add:

```go
serviceAccountService := apiv1.NewServiceAccountService(stores, iamManager)
```

**Step 2: Register ServiceAccountService handler**

After line 213 (after `userPath, userHandler := ...`), add:

```go
serviceAccountPath, serviceAccountHandler := v1connect.NewServiceAccountServiceHandler(serviceAccountService, handlerOpts)
connectHandlers[serviceAccountPath] = serviceAccountHandler
```

**Step 3: Add to reflection list**

In the `grpcreflect.NewStaticReflector(...)` call (around line 222-250), add in alphabetical order after `RolloutServiceName`:

```go
v1connect.ServiceAccountServiceName,
```

**Step 4: Register REST gateway handler**

After the `v1pb.RegisterUserServiceHandler` call (around line 342), add:

```go
if err := v1pb.RegisterServiceAccountServiceHandler(ctx, mux, grpcConn); err != nil {
	return err
}
```

**Step 5: Build to verify**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds with no errors

**Step 6: Run linter**

Run: `golangci-lint run --allow-parallel-runners`
Expected: 0 issues

---

### Task 2: Register WorkloadIdentityService in Backend

**Files:**
- Modify: `backend/server/grpc_routes.go` (same sections as Task 1)

**Step 1: Add WorkloadIdentityService instantiation**

After the serviceAccountService line added in Task 1, add:

```go
workloadIdentityService := apiv1.NewWorkloadIdentityService(stores, iamManager)
```

**Step 2: Register WorkloadIdentityService handler**

After the serviceAccountHandler registration, add:

```go
workloadIdentityPath, workloadIdentityHandler := v1connect.NewWorkloadIdentityServiceHandler(workloadIdentityService, handlerOpts)
connectHandlers[workloadIdentityPath] = workloadIdentityHandler
```

**Step 3: Add to reflection list**

In the reflector list, add after `WorkspaceServiceName` (last item, alphabetically):

```go
v1connect.WorkloadIdentityServiceName,
```

**Step 4: Register REST gateway handler**

After the ServiceAccountServiceHandler registration, add:

```go
if err := v1pb.RegisterWorkloadIdentityServiceHandler(ctx, mux, grpcConn); err != nil {
	return err
}
```

**Step 5: Build to verify**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds with no errors

**Step 6: Run linter**

Run: `golangci-lint run --allow-parallel-runners`
Expected: 0 issues

---

### Task 3: Add Frontend Connect Clients

**Files:**
- Modify: `frontend/src/connect/index.ts`

**Step 1: Add ServiceAccountService import**

After line 34 (`import { UserService } ...`), add:

```typescript
import { ServiceAccountService } from "@/types/proto-es/v1/service_account_service_pb";
import { WorkloadIdentityService } from "@/types/proto-es/v1/workload_identity_service_pb";
```

**Step 2: Add ServiceAccountService client export**

After line 140 (`export const userServiceClientConnect = ...`), add:

```typescript
export const serviceAccountServiceClientConnect = createClient(
  ServiceAccountService,
  transport
);

export const workloadIdentityServiceClientConnect = createClient(
  WorkloadIdentityService,
  transport
);
```

**Step 3: Run frontend check**

Run: `pnpm --dir frontend check`
Expected: No errors

**Step 4: Run frontend type-check**

Run: `pnpm --dir frontend type-check`
Expected: No errors

---

### Task 4: Add Services to Frontend Audit Methods List

**Files:**
- Modify: `frontend/src/connect/methods.ts`

**Step 1: Add service imports**

After line 27 (`import { UserService } ...`), add:

```typescript
import { ServiceAccountService } from "@/types/proto-es/v1/service_account_service_pb";
import { WorkloadIdentityService } from "@/types/proto-es/v1/workload_identity_service_pb";
```

**Step 2: Add to ALL_METHODS_WITH_AUDIT array**

In the array starting at line 33, add after `UserService,` (maintaining alphabetical order by placement):

```typescript
  ServiceAccountService,
```

And add after `WorkspaceService,`:

```typescript
  WorkloadIdentityService,
```

**Step 3: Run frontend check**

Run: `pnpm --dir frontend check`
Expected: No errors

---

### Task 5: Final Verification

**Step 1: Build backend**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds

**Step 2: Run backend linter**

Run: `golangci-lint run --allow-parallel-runners`
Expected: 0 issues

**Step 3: Run frontend checks**

Run: `pnpm --dir frontend check && pnpm --dir frontend type-check`
Expected: No errors

**Step 4: Commit changes**

```bash
git add backend/server/grpc_routes.go frontend/src/connect/index.ts frontend/src/connect/methods.ts
git commit -m "$(cat <<'EOF'
feat(api): register ServiceAccountService and WorkloadIdentityService

- Register ServiceAccountService and WorkloadIdentityService in grpc_routes.go
- Add Connect handlers and REST gateway handlers
- Add services to gRPC reflection list
- Add frontend Connect clients for both services
- Add services to audit methods list

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Summary

| Task | Component | Files |
|------|-----------|-------|
| 1 | Backend: ServiceAccountService registration | `grpc_routes.go` |
| 2 | Backend: WorkloadIdentityService registration | `grpc_routes.go` |
| 3 | Frontend: Connect clients | `connect/index.ts` |
| 4 | Frontend: Audit methods | `connect/methods.ts` |
| 5 | Verification & Commit | All modified files |
