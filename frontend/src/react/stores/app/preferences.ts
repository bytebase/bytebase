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
      hidden: false,
      "issue.visit": false,
      "project.visit": false,
      "environment.visit": false,
      "instance.visit": false,
      "database.visit": false,
      "member.visit": false,
      "data.query": false,
    };
    writeJson(key, next);
  },
});
