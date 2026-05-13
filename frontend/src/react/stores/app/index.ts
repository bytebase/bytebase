import { create } from "zustand";
import { createAuthSlice } from "./auth";
import { createDatabaseSlice } from "./database";
import { createDBGroupSlice } from "./dbGroup";
import { createIamSlice } from "./iam";
import { createInstanceSlice } from "./instance";
import { createInstanceRoleSlice } from "./instanceRole";
import { createNotificationSlice } from "./notification";
import { createPreferencesSlice } from "./preferences";
import { createProjectSlice } from "./project";
import { createSheetSlice } from "./sheet";
import { createWorkspaceSlice } from "./workspace";

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
    ...createInstanceRoleSlice(...args),
    ...createNotificationSlice(...args),
    ...createPreferencesSlice(...args),
  }));

export const useAppStore = createAppStore();

export function isDefaultProjectName(name: string) {
  return name === useAppStore.getState().serverInfo?.defaultProject;
}
