import { create } from "zustand";
import { createAccessGrantSlice } from "./accessGrant";
import { createAuthSlice } from "./auth";
import { createChangelogSlice } from "./changelog";
import { createDatabaseSlice } from "./database";
import { createDBGroupSlice } from "./dbGroup";
import { createGroupSlice } from "./group";
import { createIamSlice } from "./iam";
import { createIdentityProviderSlice } from "./identityProvider";
import { createInstanceSlice } from "./instance";
import { createInstanceRoleSlice } from "./instanceRole";
import { createNotificationSlice } from "./notification";
import { createPreferencesSlice } from "./preferences";
import { createProjectSlice } from "./project";
import { createProjectWebhookSlice } from "./projectWebhook";
import { createReleaseSlice } from "./release";
import { createRevisionSlice } from "./revision";
import { createRoleSlice } from "./role";
import { createServiceAccountSlice } from "./serviceAccount";
import { createSheetSlice } from "./sheet";
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
    ...createReleaseSlice(...args),
    ...createRevisionSlice(...args),
    ...createChangelogSlice(...args),
    ...createProjectWebhookSlice(...args),
    ...createNotificationSlice(...args),
    ...createPreferencesSlice(...args),
  }));

export const useAppStore = createAppStore();

export function isDefaultProjectName(name: string) {
  return name === useAppStore.getState().serverInfo?.defaultProject;
}
