import { create } from "zustand";
import { createAuthSlice } from "./auth";
import { createIamSlice } from "./iam";
import { createNotificationSlice } from "./notification";
import { createPreferencesSlice } from "./preferences";
import { createProjectSlice } from "./project";
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
    ...createNotificationSlice(...args),
    ...createPreferencesSlice(...args),
  }));

export const useAppStore = createAppStore();

export function isDefaultProjectName(name: string) {
  return name === useAppStore.getState().serverInfo?.defaultProject;
}
