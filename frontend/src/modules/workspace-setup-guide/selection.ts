import { useAppStore } from "@/stores/app";
import {
  getCurrentUserEmail,
  getWorkspaceResourceScope,
  readJson,
  writeJson,
} from "@/stores/app/utils";
import {
  storageKeyWorkspaceSetupGuideScenario,
  storageKeyWorkspaceSetupGuideWorkspaceUsage,
} from "@/utils/storage-keys";
import type { GuideScenarioId, GuideWorkspaceUsage } from "./types";

export const isGuideScenarioId = (value: unknown): value is GuideScenarioId =>
  value === "query-data" || value === "create-database-change";

export const isGuideWorkspaceUsage = (
  value: unknown
): value is GuideWorkspaceUsage => value === "team" || value === "solo";

const selectionStorageKey = () => {
  const get = useAppStore.getState;
  const email = getCurrentUserEmail(get);
  const scope = getWorkspaceResourceScope(get);
  if (!email || !scope) return undefined;
  return storageKeyWorkspaceSetupGuideScenario(scope, email);
};

const workspaceUsageStorageKey = () => {
  const get = useAppStore.getState;
  const email = getCurrentUserEmail(get);
  const scope = getWorkspaceResourceScope(get);
  if (!email || !scope) return undefined;
  return storageKeyWorkspaceSetupGuideWorkspaceUsage(scope, email);
};

export const readSelectedGuideScenarioId = (): GuideScenarioId | undefined => {
  const key = selectionStorageKey();
  if (!key) return undefined;
  const value = readJson<unknown>(key, undefined);
  return isGuideScenarioId(value) ? value : undefined;
};

export const saveSelectedGuideScenarioId = (id: GuideScenarioId): boolean => {
  const key = selectionStorageKey();
  if (!key) return false;
  try {
    writeJson(key, id);
    return true;
  } catch {
    return false;
  }
};

export const clearSelectedGuideScenarioId = (): boolean => {
  const key = selectionStorageKey();
  if (!key) return false;
  try {
    localStorage.removeItem(key);
    return true;
  } catch {
    return false;
  }
};

export const readGuideWorkspaceUsage = (): GuideWorkspaceUsage | undefined => {
  const key = workspaceUsageStorageKey();
  if (!key) return undefined;
  const value = readJson<unknown>(key, undefined);
  return isGuideWorkspaceUsage(value) ? value : undefined;
};

export const saveGuideWorkspaceUsage = (
  value: GuideWorkspaceUsage
): boolean => {
  const key = workspaceUsageStorageKey();
  if (!key) return false;
  try {
    writeJson(key, value);
    return true;
  } catch {
    return false;
  }
};

export const clearGuideWorkspaceUsage = (): boolean => {
  const key = workspaceUsageStorageKey();
  if (!key) return false;
  try {
    localStorage.removeItem(key);
    return true;
  } catch {
    return false;
  }
};
