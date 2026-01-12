# Unify Predefined Roles in Store Package

## Goal

Move predefined role definitions from IAM manager to store package so that:
1. `store.ListRoles()` returns all roles (custom + predefined)
2. IAM manager focuses purely on permission checking

## Dependency Graph (unchanged)

```
v1 service → store
v1 service → iam manager
iam manager → store
iam manager → common/permission
store → common/permission
```

## File Changes

### New Package: `backend/common/permission/`

| Action | From | To |
|--------|------|-----|
| Move | `backend/component/iam/permission.go` | `backend/common/permission/permission.go` |
| Move | `backend/component/iam/permission.yaml` | `backend/common/permission/permission.yaml` |

### Store Package

| Action | File | Notes |
|--------|------|-------|
| Create | `backend/store/predefined_roles.go` | Define 9 roles using permission constants |
| Update | `backend/store/role.go` | `ListRoles()` appends predefined roles |

### IAM Manager

| Action | File | Notes |
|--------|------|-------|
| Update | `backend/component/iam/manager.go` | Remove `PredefinedRoles` field, `loadPredefinedRoles()`, YAML embedding |
| Delete | `backend/component/iam/acl.yaml` | Replaced by Go code |

### API Service

| Action | File | Notes |
|--------|------|-------|
| Update | `backend/api/v1/role_service.go` | Remove manual merge with predefined roles |

### Frontend

| Action | File | Notes |
|--------|------|-------|
| Update | `frontend/scripts/copy_config_files.sh` | New path: `../backend/common/permission/permission.yaml` |

## Implementation Details

### `store/predefined_roles.go`

```go
package store

import "github.com/bytebase/bytebase/backend/common/permission"

var predefinedRoles = []*RoleMessage{
    {
        ResourceID: "workspaceAdmin",
        Name:       "Workspace admin",
        Permissions: permissionSet(
            permission.PermissionAuditLogsExport,
            permission.PermissionInstancesCreate,
            // ... all permissions
        ),
    },
    // ... 8 more roles
}

func permissionSet(perms ...permission.Permission) map[string]bool {
    m := make(map[string]bool, len(perms))
    for _, p := range perms {
        m[string(p)] = true
    }
    return m
}
```

### `store/role.go` - ListRoles

```go
func (s *Store) ListRoles(ctx context.Context, find *FindRoleMessage) ([]*RoleMessage, error) {
    // existing DB query for custom roles
    roles := // from DB

    // Append predefined roles
    roles = append(roles, predefinedRoles...)
    return roles, nil
}
```

### `iam/manager.go` - Changes

Remove:
- `PredefinedRoles` field from `Manager` struct
- `loadPredefinedRoles()` function
- `//go:embed acl.yaml` directive

Update `ReloadCache()` to use `ListRoles()` directly without appending predefined.

## Migration

- No database migration needed
- No breaking API changes
- Update tests that reference `loadPredefinedRoles()` or `PredefinedRoles` field
