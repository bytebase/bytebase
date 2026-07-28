import { create } from "zustand";
import { createAccessGrantSlice } from "./accessGrant";
import { createAuthSlice } from "./auth";
import { createChangelogSlice } from "./changelog";
import { createDatabaseSlice } from "./database";
import { createDatabaseCatalogSlice } from "./databaseCatalog";
import { createDBGroupSlice } from "./dbGroup";
import { createDBSchemaSlice } from "./dbSchema";
import { createGroupSlice } from "./group";
import { createIamSlice } from "./iam";
import { createIdentityProviderSlice } from "./identityProvider";
import { createIssueCommentSlice } from "./issueComment";
import { createInstanceSlice } from "./instance";
import { createInstanceRoleSlice } from "./instanceRole";
import { createIssueSlice } from "./issue";
import { createPolicySlice } from "./policy";
import { createNotificationSlice } from "./notification";
import { createPreferencesSlice } from "./preferences";
import { createProjectSlice } from "./project";
import { createProjectWebhookSlice } from "./projectWebhook";
import { createReleaseSlice } from "./release";
import { createRevisionSlice } from "./revision";
import { createPlanSlice } from "./plan";
import { createRoleSlice } from "./role";
import { createRolloutSlice } from "./rollout";
import { createServiceAccountSlice } from "./serviceAccount";
import { createSheetSlice } from "./sheet";
import { createSQLSlice } from "./sql";
import { createUserSlice } from "./user";
import { createWorksheetSlice } from "./worksheet";
import { createWorkspaceSlice } from "./workspace";
import { createWorkloadIdentitySlice } from "./workloadIdentity";

export type { AppStoreState } from "./types";
export {
  getProjectResourceId,
  isConnectAlreadyExists,
  projectResourceNameFromId,
} from "./utils";
export type { ProjectListParams } from "./types";
import { registerAppStoreUtilBridge } from "@/utils/app-store-bridge";
import { registerPermissionCheckers } from "@/utils/iam/permission";
import type { AppStoreState } from "./types";

export const createAppStore = () =>
  create<AppStoreState>()((...args) => ({
    ...createAuthSlice(...args),
    ...createWorkspaceSlice(...args),
    ...createIamSlice(...args),
    ...createProjectSlice(...args),
    ...createInstanceSlice(...args),
    ...createDatabaseSlice(...args),
    ...createDBGroupSlice(...args),
    ...createDBSchemaSlice(...args),
    ...createSheetSlice(...args),
    ...createWorksheetSlice(...args),
    ...createInstanceRoleSlice(...args),
    ...createGroupSlice(...args),
    ...createServiceAccountSlice(...args),
    ...createWorkloadIdentitySlice(...args),
    ...createIdentityProviderSlice(...args),
    ...createAccessGrantSlice(...args),
    ...createUserSlice(...args),
    ...createRoleSlice(...args),
    ...createRolloutSlice(...args),
    ...createPlanSlice(...args),
    ...createIssueCommentSlice(...args),
    ...createDatabaseCatalogSlice(...args),
    ...createReleaseSlice(...args),
    ...createRevisionSlice(...args),
    ...createChangelogSlice(...args),
    ...createProjectWebhookSlice(...args),
    ...createNotificationSlice(...args),
    ...createPreferencesSlice(...args),
    ...createSQLSlice(...args),
    ...createIssueSlice(...args),
    ...createPolicySlice(...args),
  }));

export const useAppStore = createAppStore();

// Back the legacy `hasWorkspacePermissionV2` / `hasProjectPermissionV2` shared
// utils with the app store, so all callers resolve permissions from this store
// without importing it (which would create an init cycle through `@/utils`).
registerPermissionCheckers({
  hasWorkspacePermission: (permission) =>
    useAppStore.getState().hasWorkspacePermission(permission),
  hasProjectPermission: (project, permission) =>
    useAppStore.getState().hasProjectPermission(project, permission),
});

// Back the subscription / environment reads in shared `@/utils` helpers with
// this store, replacing the legacy Pinia stores those helpers used to read.
registerAppStoreUtilBridge({
  currentUser: () => useAppStore.getState().currentUser,
  isLoggedIn: () => useAppStore.getState().isLoggedIn(),
  setUnauthenticatedOccurred: (value) =>
    useAppStore.getState().setUnauthenticatedOccurred(value),
  defaultProjectName: () =>
    useAppStore.getState().serverInfo?.defaultProject ?? "",
  currentPlan: () => useAppStore.getState().currentPlan(),
  hasFeature: (feature) => useAppStore.getState().hasFeature(feature),
  environmentList: () => useAppStore.getState().environmentList,
  getEnvironmentByName: (name, fallback) =>
    useAppStore.getState().getEnvironmentByName(name, fallback),
  roleList: () => useAppStore.getState().roleList,
  getRoleByName: (name) => useAppStore.getState().getRoleByName(name),
  getGroupByIdentifier: (identifier) =>
    useAppStore.getState().getGroupByIdentifier(identifier),
  workspaceRoleMapToUsers: () =>
    useAppStore.getState().workspaceRoleMapToUsers(),
  getProjectIamPolicy: (project) =>
    useAppStore.getState().getProjectIamPolicy(project),
});

export function isDefaultProjectName(name: string) {
  return name === useAppStore.getState().serverInfo?.defaultProject;
}
