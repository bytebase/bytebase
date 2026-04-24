import {
  storageKeyIntroState,
  storageKeyRecentProjects,
  storageKeyRecentVisit,
} from "@/utils/storage-keys";
import type { AppSliceCreator, PreferencesSlice } from "./types";
import {
  getCurrentUserEmail,
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
  _set,
  get
) => ({
  setRecentProject: (name) => {
    const email = getCurrentUserEmail(get);
    if (!email || !name) return;
    const key = storageKeyRecentProjects(email);
    const previous = readJson<string[]>(key, []);
    writeJson(
      key,
      [name, ...previous.filter((projectName) => projectName !== name)].slice(
        0,
        MAX_RECENT_PROJECT
      )
    );
  },

  recordRecentVisit: (path) => {
    const email = getCurrentUserEmail(get);
    if (!email) return;
    const key = storageKeyRecentVisit(email);
    const previous = readJson<string[]>(key, []);
    const pathOnly = path.replace(/[?#].*$/, "");
    const next = [
      path,
      ...previous.filter((item) => item.replace(/[?#].*$/, "") !== pathOnly),
    ].slice(0, MAX_RECENT_VISIT + 1);
    writeJson(key, next);
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
    window.dispatchEvent(
      new CustomEvent("bb.react-quickstart-reset", {
        detail: { keys: QUICKSTART_RESET_KEYS },
      })
    );
  },
});
