import { useAppStore } from "@/stores/app";
import {
  getCurrentUserEmail,
  getWorkspaceResourceScope,
  readJson,
  writeJson,
} from "@/stores/app/utils";
import {
  storageKeyWorkspaceSetupDraft,
  storageKeyWorkspaceSetupGuideScenario,
  storageKeyWorkspaceSetupGuideTeamSize,
  storageKeyWorkspaceSetupGuideWorkspaceUsage,
} from "@/utils/storage-keys";
import { isGuideScenarioId } from "./scenarios";
import {
  GUIDE_TEAM_SIZES,
  type GuideScenarioId,
  type GuideTeamSize,
  type GuideWorkspaceUsage,
} from "./types";

export const isGuideWorkspaceUsage = (
  value: unknown
): value is GuideWorkspaceUsage => value === "team" || value === "solo";

export const isGuideTeamSize = (value: unknown): value is GuideTeamSize =>
  (GUIDE_TEAM_SIZES as readonly unknown[]).includes(value);

// Setup's answers are kept per workspace and per user, in this browser only.
// Undefined until both are known.
export const setupStorageKey = (
  build: (scope: string, email: string) => string
): string | undefined => {
  const get = useAppStore.getState;
  const email = getCurrentUserEmail(get);
  const scope = getWorkspaceResourceScope(get);
  if (!email || !scope) return undefined;
  return build(scope, email);
};

const readSetupAnswer = <T>(
  build: (scope: string, email: string) => string,
  isValid: (value: unknown) => value is T
): T | undefined => {
  const key = setupStorageKey(build);
  if (!key) return undefined;
  const value = readJson<unknown>(key, undefined);
  return isValid(value) ? value : undefined;
};

// False when the answer was not stored: storage that is full or blocked must
// not stop setup.
export const saveSetupAnswer = (
  build: (scope: string, email: string) => string,
  value: unknown
): boolean => {
  const key = setupStorageKey(build);
  if (!key) return false;
  try {
    writeJson(key, value);
    return true;
  } catch {
    return false;
  }
};

// True while setup this account began in this browser has neither finished
// nor been skipped.
export const hasWorkspaceSetupDraft = () => {
  const key = setupStorageKey(storageKeyWorkspaceSetupDraft);
  if (!key) return false;
  try {
    return localStorage.getItem(key) !== null;
  } catch {
    return false;
  }
};

export const readSelectedGuideScenarioId = () =>
  readSetupAnswer(storageKeyWorkspaceSetupGuideScenario, isGuideScenarioId);

export const saveSelectedGuideScenarioId = (id: GuideScenarioId) =>
  saveSetupAnswer(storageKeyWorkspaceSetupGuideScenario, id);

export const readGuideTeamSize = () =>
  readSetupAnswer(storageKeyWorkspaceSetupGuideTeamSize, isGuideTeamSize);

export const saveGuideTeamSize = (value: GuideTeamSize) =>
  saveSetupAnswer(storageKeyWorkspaceSetupGuideTeamSize, value);

// The team size answers this for accounts set up since the wizard asked for
// it; the stored usage is what earlier accounts answered.
export const readGuideWorkspaceUsage = (): GuideWorkspaceUsage | undefined => {
  const teamSize = readGuideTeamSize();
  if (teamSize) return teamSize === "just-me" ? "solo" : "team";
  return readSetupAnswer(
    storageKeyWorkspaceSetupGuideWorkspaceUsage,
    isGuideWorkspaceUsage
  );
};
