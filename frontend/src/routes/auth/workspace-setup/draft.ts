import { isGuideScenarioId } from "@/modules/workspace-setup-guide/scenarios";
import {
  isGuideTeamSize,
  saveSetupAnswer,
  setupStorageKey,
} from "@/modules/workspace-setup-guide/selection";
import type {
  GuideScenarioId,
  GuideTeamSize,
} from "@/modules/workspace-setup-guide/types";
import { readJson } from "@/stores/app/utils";
import { storageKeyWorkspaceSetupDraft } from "@/utils/storage-keys";
import type { ProjectGrants, ProjectIdRefusal } from "./finish";

const SETUP_SCREENS = ["workspace", "goal", "project", "team"] as const;

export type SetupScreen = (typeof SETUP_SCREENS)[number];

// The team step is Cloud's: on self-hosted its answer reaches nothing.
export const setupScreens = (isSaaS: boolean): readonly SetupScreen[] =>
  isSaaS ? SETUP_SCREENS : SETUP_SCREENS.filter((screen) => screen !== "team");

export type SetupGoal = Exclude<GuideScenarioId, "mark-sensitive-data">;

export type SetupSource = "sample" | "own";

export type SetupDraft = {
  screen: SetupScreen;
  // Undefined until edited, so an untouched field keeps showing its prefill.
  name?: string;
  workspaceTitle?: string;
  goal?: SetupGoal;
  projectTitle: string;
  projectId: string;
  source: SetupSource;
  teamSize?: GuideTeamSize;
  emails: string[];
  // The ID a Finish attempt has sent a create for. Until `createdProject` is
  // set, that create's outcome is unknown; a refusal, or a project found
  // holding the ID, settles it.
  attemptedProjectId?: string;
  // The ID the server refused at Finish, which blocks until the ID changes.
  refusedProjectId?: ProjectIdRefusal & { id: string };
  // What a Finish attempt has already made on the server; from then on the
  // project's name, ID and prepared sample are fixed.
  createdProject?: string;
  samplePrepared?: boolean;
  sampleInstanceName?: string;
  projectGrants?: ProjectGrants;
};

const emptySetupDraft = (): SetupDraft => ({
  screen: "workspace",
  projectTitle: "",
  projectId: "",
  source: "sample",
  emails: [],
});

export const isSetupGoal = (value: unknown): value is SetupGoal =>
  isGuideScenarioId(value) && value !== "mark-sensitive-data";

// Kept as the guide's stored answers are. It holds no credentials, because the
// wizard collects none.
const draftStorageKey = () => setupStorageKey(storageKeyWorkspaceSetupDraft);

const optionalString = (value: unknown) =>
  typeof value === "string" ? value : undefined;

const strings = (value: unknown): string[] =>
  Array.isArray(value) ? value.filter((item) => typeof item === "string") : [];

const grants = (value: unknown): ProjectGrants =>
  value && typeof value === "object"
    ? Object.fromEntries(
        Object.entries(value).map(([role, members]) => [role, strings(members)])
      )
    : {};

const refusedId = (value: unknown): SetupDraft["refusedProjectId"] => {
  if (!value || typeof value !== "object") return undefined;
  const { id, reason } = value as Record<string, unknown>;
  return typeof id === "string"
    ? { id, reason: optionalString(reason) }
    : undefined;
};

// Self-hosted's ways out of setup without a project, offered until a Finish
// attempt has created the project or sent a create whose outcome is unknown.
export const waysOutOffered = (draft: SetupDraft, isSaaS: boolean) =>
  !isSaaS && !draft.createdProject && !draft.attemptedProjectId;

// Storage is outside the type system, so every field is read defensively and
// the screen falls back to the first one whose answer is missing.
export const readSetupDraft = (isSaaS: boolean): SetupDraft => {
  const key = draftStorageKey();
  if (!key) return emptySetupDraft();
  const stored = readJson<Record<string, unknown> | undefined>(key, undefined);
  if (!stored || typeof stored !== "object") return emptySetupDraft();

  const draft: SetupDraft = {
    ...emptySetupDraft(),
    name: optionalString(stored.name),
    workspaceTitle: optionalString(stored.workspaceTitle),
    goal: isSetupGoal(stored.goal) ? stored.goal : undefined,
    projectTitle: optionalString(stored.projectTitle) ?? "",
    projectId: optionalString(stored.projectId) ?? "",
    source: stored.source === "own" ? "own" : "sample",
    teamSize: isGuideTeamSize(stored.teamSize) ? stored.teamSize : undefined,
    emails: strings(stored.emails),
    attemptedProjectId: optionalString(stored.attemptedProjectId),
    refusedProjectId: refusedId(stored.refusedProjectId),
    createdProject: optionalString(stored.createdProject),
    samplePrepared: stored.samplePrepared === true,
    sampleInstanceName: optionalString(stored.sampleInstanceName),
    projectGrants: grants(stored.projectGrants),
  };
  const projectAnswered = !!draft.projectTitle.trim() && !!draft.projectId;
  const screens = setupScreens(isSaaS);
  const reachable: SetupScreen = !draft.goal
    ? "goal"
    : !projectAnswered || !isSaaS
      ? "project"
      : "team";
  // A self-hosted draft saved on the team step before it was removed opens on
  // the last step there is.
  const storedScreen = screens.find((screen) => screen === stored.screen);
  draft.screen =
    storedScreen && screens.indexOf(storedScreen) <= screens.indexOf(reachable)
      ? storedScreen
      : reachable;
  return draft;
};

// Each save is stored with a revision of its own, so it changes what is stored
// even when no answer changed, and the other tabs hear of it.
export const saveSetupDraft = (draft: SetupDraft) =>
  saveSetupAnswer(storageKeyWorkspaceSetupDraft, {
    ...draft,
    revision: crypto.getRandomValues(new Uint32Array(2)).join("-"),
  });

// Calls `onChange` when another tab saves the draft, or removes it because
// setup ended there.
export const watchSetupDraft = (onChange: (ended: boolean) => void) => {
  const listener = (event: StorageEvent) => {
    if (event.key === draftStorageKey()) onChange(event.newValue === null);
  };
  window.addEventListener("storage", listener);
  return () => window.removeEventListener("storage", listener);
};

export const clearSetupDraft = () => {
  const key = draftStorageKey();
  if (!key) return;
  try {
    localStorage.removeItem(key);
  } catch {
    // Nothing to remove from storage that cannot be reached.
  }
};
