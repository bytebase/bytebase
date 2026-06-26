import { emitReactQuickstartReset } from "@/react/shell-bridge";
import {
  storageKeyIntroState,
  storageKeyRecentProjects,
  storageKeyRecentVisit,
} from "@/utils/storage-keys";
import type { AppSliceCreator, PreferencesSlice } from "./types";
import {
  getCurrentUserEmail,
  getWorkspaceCacheScope,
  MAX_RECENT_PROJECT,
  MAX_RECENT_VISIT,
  readJson,
  writeJson,
} from "./utils";

const QUICKSTART_RESET_KEYS = [
  "hidden",
  "issue.visit",
  "project.visit",
  "environment.visit",
  "instance.visit",
  "database.visit",
  "member.visit",
  "data.query",
];

export const createPreferencesSlice: AppSliceCreator<PreferencesSlice> = (
  set,
  get
) => ({
  introStateVersion: 0,
  setRecentProject: (name) => {
    const email = getCurrentUserEmail(get);
    if (!email || !name) return;
    const key = storageKeyRecentProjects(getWorkspaceCacheScope(get), email);
    const previous = readJson<string[]>(key, []);
    writeJson(
      key,
      [name, ...previous.filter((projectName) => projectName !== name)].slice(
        0,
        MAX_RECENT_PROJECT
      )
    );
  },

  recordRecentVisit: (path, workspaceName) => {
    const email = getCurrentUserEmail(get);
    if (!email) return;
    const key = storageKeyRecentVisit(
      getWorkspaceCacheScope(get, workspaceName),
      email
    );
    const previous = readJson<string[]>(key, []);
    const pathOnly = path.replace(/[?#].*$/, "");
    const next = [
      path,
      ...previous.filter((item) => item.replace(/[?#].*$/, "") !== pathOnly),
    ].slice(0, MAX_RECENT_VISIT + 1);
    writeJson(key, next);
  },

  removeRecentVisit: (path) => {
    const email = getCurrentUserEmail(get);
    if (!email) return;
    const key = storageKeyRecentVisit(getWorkspaceCacheScope(get), email);
    const previous = readJson<string[]>(key, []);
    writeJson(
      key,
      previous.filter((item) => item !== path)
    );
  },

  resetQuickstartProgress: () => {
    const email = getCurrentUserEmail(get);
    if (!email) return;
    const key = storageKeyIntroState(email);
    const previous = readJson<Record<string, boolean>>(key, {});
    const next = {
      ...previous,
      ...Object.fromEntries(QUICKSTART_RESET_KEYS.map((key) => [key, false])),
    };
    writeJson(key, next);
    set((state) => ({ introStateVersion: state.introStateVersion + 1 }));
    emitReactQuickstartReset({ keys: QUICKSTART_RESET_KEYS });
  },

  // Mirrors the Pinia `useUIStateStore.getIntroStateByKey`. Reads the per-user
  // localStorage map; `introStateVersion` (read by callers' selectors) is what
  // makes a subsequent `saveIntroStateByKey` re-trigger the read.
  getIntroStateByKey: (key) => {
    const email = getCurrentUserEmail(get);
    if (!email) return false;
    const map = readJson<Record<string, boolean>>(
      storageKeyIntroState(email),
      {}
    );
    return map[key] ?? false;
  },

  // Mirrors the Pinia `useUIStateStore.saveIntroStateByKey`: persists a
  // single intro flag (e.g. `data.query`) to the per-user localStorage map.
  saveIntroStateByKey: ({ key, newState }) => {
    const email = getCurrentUserEmail(get);
    if (!email) return;
    const storageKey = storageKeyIntroState(email);
    const previous = readJson<Record<string, boolean>>(storageKey, {});
    writeJson(storageKey, { ...previous, [key]: newState });
    set((state) => ({ introStateVersion: state.introStateVersion + 1 }));
  },
});
