import type { Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

// Permission checks resolve against the React app store (the single source of
// truth for the current user, workspace IAM policy, and roles). The legacy
// Pinia `usePermissionStore` is no longer populated and must not be used.
//
// The app store *registers* its checkers here at init rather than this module
// importing the store directly: `@/utils` sits in the early barrel-import
// graph (`@/types` → `@/utils` → this file), so a static import of the store
// would force `createAppStore()` to run before `@/types` finishes
// initializing, crashing on partially-initialized exports.
type WorkspaceChecker = (permission: Permission) => boolean;
type ProjectChecker = (project: Project, permission: Permission) => boolean;

// Default to "deny" until the app store registers (it does so at module init,
// well before any render-time permission check).
let workspaceChecker: WorkspaceChecker = () => false;
let projectChecker: ProjectChecker = () => false;
let projectWideChecker: ProjectChecker = () => false;

export const registerPermissionCheckers = (checkers: {
  hasWorkspacePermission: WorkspaceChecker;
  hasProjectPermission: ProjectChecker;
  hasProjectWidePermission: ProjectChecker;
}): void => {
  workspaceChecker = checkers.hasWorkspacePermission;
  projectChecker = checkers.hasProjectPermission;
  projectWideChecker = checkers.hasProjectWidePermission;
};

export const hasWorkspacePermissionV2 = (permission: Permission): boolean =>
  workspaceChecker(permission);

// hasProjectPermissionV2 checks if the user has the given permission on the
// project. The app store's hasProjectPermission already falls back to
// workspace-level permissions first.
export const hasProjectPermissionV2 = (
  project: Project,
  permission: Permission
): boolean => projectChecker(project, permission);

// hasProjectWidePermissionV2 mirrors the server's CheckProjectWidePermission:
// a binding whose condition scopes resources confers nothing. Surfaces the
// server gates project-wide (the saved-query permissions) use this so the UI
// never offers what the RPC will deny.
export const hasProjectWidePermissionV2 = (
  project: Project,
  permission: Permission
): boolean => projectWideChecker(project, permission);
