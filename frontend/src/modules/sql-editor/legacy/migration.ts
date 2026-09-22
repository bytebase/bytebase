import { create } from "@bufbuild/protobuf";
import { isUndefined, omitBy } from "lodash-es";
import { extractSavedQueryConnection } from "@/lib/sqlEditorConnection";
import { useAppStore } from "@/stores/app";
import { getCurrentUserV1 } from "@/stores/modules/migration-helpers";
import { extractUserEmail } from "@/stores/modules/v1";
import type { EditorPanelViewState, SQLEditorTab } from "@/types";
import { SavedQuerySchema } from "@/types/proto-es/v1/saved_query_service_pb";
import { canCreateSavedQueryInProject, defaultSQLEditorTab } from "@/utils";

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

// Legacy tabs were stored before the worksheet → saved query rename, so
// the persisted shape keeps the historical `worksheet` field name.
type PersistentTab = Pick<
  SQLEditorTab,
  "id" | "title" | "connection" | "mode" | "status"
> & { worksheet?: string };

const LEGACY_POUCH_DATABASES = [
  "_pouch_bb.plugin.ai.conversations",
  "_pouch_bb.plugin.ai.messages",
  "_pouch_bb.sql-editor.extended-tab",
  "_pouch_bb.storage",
];

export const cleanupLegacyPouchDatabases = async () => {
  if (!globalThis.indexedDB) return;
  await Promise.all(
    LEGACY_POUCH_DATABASES.map(
      (name) =>
        new Promise<void>((resolve) => {
          let request: IDBOpenDBRequest;
          try {
            request = indexedDB.deleteDatabase(name);
          } catch {
            resolve();
            return;
          }
          request.onsuccess = () => resolve();
          request.onerror = () => resolve();
          request.onblocked = () => resolve();
        })
    )
  );
};

export const migrateDraftsFromCache = async (project: string) => {
  const userUID = extractUserEmail(getCurrentUserV1().name);

  const viewStateByTab = readLegacyStorage<
    Map</* tab.id */ string, EditorPanelViewState>
  >(`bb.sql-editor-tab-state.${userUID}`, new Map());

  const keyNamespace = `${LOCAL_STORAGE_KEY_PREFIX}.${project}.${userUID}`;

  const draftTabListKey = `${keyNamespace}.draft-tab-list`;
  const draftTabList = readLegacyStorage<SQLEditorTab[]>(draftTabListKey, []);

  // Migrating a draft creates a saved query. This check is the cheap half of
  // the answer -- false means the server will refuse too, so skip the doomed
  // requests entirely. True is not a promise: it does not evaluate binding
  // conditions, and the server rejects a resource-scoped grant for this
  // permission. The loop below has to survive a refusal either way.
  if (!canCreateSavedQueryInProject(project)) {
    return;
  }

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
        const connection = await extractSavedQueryConnection({
          database: tab.connection.database,
        });
        await useAppStore.getState().createSavedQuery(
          create(SavedQuerySchema, {
            title: tab.title,
            database: connection.database,
            content: new TextEncoder().encode(tab.statement),
            project,
          })
        );
      } catch {
        // The draft in legacy storage is the only copy, so it survives a
        // failed migration: a denied create, a network blip, a server error.
        // It is retried the next time this runs.
        continue;
      }
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
