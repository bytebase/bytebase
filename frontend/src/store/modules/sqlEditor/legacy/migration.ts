import { create } from "@bufbuild/protobuf";
import { isUndefined, omit, omitBy } from "lodash-es";
import { extractWorksheetConnection } from "@/react/lib/sqlEditorConnection";
import { useAppStore } from "@/react/stores/app";
import type { EditorPanelViewState, SQLEditorTab } from "@/types";
import { DEFAULT_SQL_EDITOR_TAB_MODE } from "@/types";
import {
  Worksheet_Visibility,
  WorksheetSchema,
} from "@/types/proto-es/v1/worksheet_service_pb";
import { defaultSQLEditorTab, WebStorageHelper } from "@/utils";
import { getCurrentUserV1 } from "../../migration-helpers";
import { extractUserEmail } from "../../v1";
import {
  deleteExtendedTab,
  EXTENDED_TAB_FIELDS,
  fetchExtendedTab,
} from "./extendedTab";

const LOCAL_STORAGE_KEY_PREFIX = "bb.sql-editor-tab";

// Plain localStorage access mirroring the serialization the legacy data was
// written with by `useDynamicLocalStorage` (vueuse `useStorage`): plain JSON
// for objects/arrays, and entries-array JSON for `Map`. `read` also persists
// the default when the key is absent — matching vueuse's `writeDefaults`, which
// `storage-migrate.ts` relies on running first.
const isMap = (v: unknown): v is Map<unknown, unknown> => v instanceof Map;

const serializeLegacy = (value: unknown): string =>
  isMap(value) ? JSON.stringify([...value.entries()]) : JSON.stringify(value);

const readLegacyStorage = <T>(key: string, defaults: T): T => {
  const raw = localStorage.getItem(key);
  if (raw == null) {
    if (defaults != null) {
      localStorage.setItem(key, serializeLegacy(defaults));
    }
    return defaults;
  }
  try {
    const parsed = JSON.parse(raw);
    return isMap(defaults) ? (new Map(parsed) as T) : (parsed as T);
  } catch {
    return defaults;
  }
};

const writeLegacyStorage = <T>(key: string, value: T): void => {
  if (value == null) {
    localStorage.removeItem(key);
    return;
  }
  localStorage.setItem(key, serializeLegacy(value));
};

type PersistentTab = Pick<
  SQLEditorTab,
  "id" | "title" | "connection" | "mode" | "worksheet" | "status"
>;

type LegacyStoredTab = PersistentTab & { statement?: string };

export const migrateLegacyCache = async () => {
  const userUID = extractUserEmail(getCurrentUserV1().name);

  const keyNamespace = `${LOCAL_STORAGE_KEY_PREFIX}.${userUID}`;
  const tabIdListKey = `${keyNamespace}.tab-id-list`;
  const tabIdListMapByProject = readLegacyStorage<Record<string, string[]>>(
    tabIdListKey,
    {}
  );

  const getStorage = () => {
    return new WebStorageHelper(keyNamespace);
  };

  const keyForTab = (id: string) => {
    return `tab.${id}`;
  };

  const loadStoredTab = async (id: string) => {
    const stored = getStorage().load<PersistentTab | undefined>(
      keyForTab(id),
      undefined
    );
    if (!stored) {
      return undefined;
    }
    const tab: SQLEditorTab = {
      ...defaultSQLEditorTab(),
      // Ignore extended fields stored in localStorage since they are migrated
      // to extendedTabStore.
      ...omit(stored, EXTENDED_TAB_FIELDS),
      id,
    };
    if (tab.mode !== DEFAULT_SQL_EDITOR_TAB_MODE) {
      // Do not enter ADMIN mode initially
      tab.mode = DEFAULT_SQL_EDITOR_TAB_MODE;
    }

    await fetchExtendedTab(tab, () => {
      // When the first time of migration, the extended doc in IndexedDB is not
      // found.
      // Fallback to the original PersistentTab in LocalStorage if possible.
      // This might happen only once to each user, since the second time when a
      // tab is saved, extended fields will be migrated, and won't be saved to
      // LocalStorage, so the fallback routine won't be hit.
      const { statement } = stored as LegacyStoredTab;
      if (statement) {
        tab.statement = statement;
      }
    });

    return tab;
  };

  const entries = [...Object.entries(tabIdListMapByProject)];
  for (const [project, tabIds] of entries) {
    const draftTabListKey = `${LOCAL_STORAGE_KEY_PREFIX}.${project}.${userUID}.draft-tab-list`;
    const draftTabList = readLegacyStorage<SQLEditorTab[]>(draftTabListKey, []);

    for (const tabId of tabIds) {
      const exist = draftTabList.find((draft) => draft.id === tabId);
      if (exist) {
        continue;
      }
      const tab = await loadStoredTab(tabId);
      await deleteExtendedTab(tabId);
      if (!tab) {
        continue;
      }
      draftTabList.push(tab);
    }
    writeLegacyStorage(draftTabListKey, draftTabList);

    delete tabIdListMapByProject[project];
    writeLegacyStorage(tabIdListKey, tabIdListMapByProject);
  }
};

export const migrateDraftsFromCache = async (project: string) => {
  const userUID = extractUserEmail(getCurrentUserV1().name);

  const viewStateByTab = readLegacyStorage<
    Map</* tab.id */ string, EditorPanelViewState>
  >(`bb.sql-editor-tab-state.${userUID}`, new Map());

  const keyNamespace = `${LOCAL_STORAGE_KEY_PREFIX}.${project}.${userUID}`;

  const draftTabListKey = `${keyNamespace}.draft-tab-list`;
  const draftTabList = readLegacyStorage<SQLEditorTab[]>(draftTabListKey, []);

  const drafts = [...draftTabList];
  for (const draft of drafts) {
    const tab = {
      ...defaultSQLEditorTab(),
      ...omitBy(draft, isUndefined),
    };
    const viewState = viewStateByTab.get(tab.id);
    if ((!viewState || viewState.view === "CODE") && tab.statement) {
      // only store the draft with content
      try {
        const connection = await extractWorksheetConnection({
          database: tab.connection.database,
        });
        await useAppStore.getState().createWorksheet(
          create(WorksheetSchema, {
            title: tab.title,
            database: connection.database,
            content: new TextEncoder().encode(tab.statement),
            project,
            visibility: Worksheet_Visibility.PRIVATE,
          })
        );
      } catch {}
    }
    const index = draftTabList.findIndex((d) => d.id === draft.id);
    if (index >= 0) {
      draftTabList.splice(index, 1);
      writeLegacyStorage(draftTabListKey, draftTabList);
    }
  }
};

export const migrateTabViewState = (project: string) => {
  const userUID = extractUserEmail(getCurrentUserV1().name);

  const keyNamespace = `${LOCAL_STORAGE_KEY_PREFIX}.${project}.${userUID}`;

  const viewStateKey = `bb.sql-editor-tab-state.${userUID}`;
  const viewStateByTab = readLegacyStorage<
    Map</* tab.id */ string, EditorPanelViewState>
  >(viewStateKey, new Map());

  const openTmpTabListKey = `${keyNamespace}.opening-tab-list`;
  const openTmpTabList = readLegacyStorage<PersistentTab[]>(
    openTmpTabListKey,
    []
  );

  for (const openedTab of openTmpTabList) {
    const viewState = viewStateByTab.get(openedTab.id);
    if (viewState) {
      Object.assign(openedTab, { viewState });
    }
    viewStateByTab.delete(openedTab.id);
  }
  writeLegacyStorage(openTmpTabListKey, openTmpTabList);
  writeLegacyStorage(viewStateKey, viewStateByTab);
};
