import { create } from "@bufbuild/protobuf";
import { isUndefined, omit, omitBy } from "lodash-es";
import { computed, reactive } from "vue";
import type { EditorPanelViewState, SQLEditorTab } from "@/types";
import { DEFAULT_SQL_EDITOR_TAB_MODE } from "@/types";
import {
  Worksheet_Visibility,
  WorksheetSchema,
} from "@/types/proto-es/v1/worksheet_service_pb";
import {
  defaultSQLEditorTab,
  extractWorksheetConnection,
  useDynamicLocalStorage,
  WebStorageHelper,
} from "@/utils";
import { extractUserId, useWorkSheetStore } from "../../v1";
import { useCurrentUserV1 } from "../../v1/auth";
import { EXTENDED_TAB_FIELDS, useExtendedTabStore } from "./extendedTab";

const LOCAL_STORAGE_KEY_PREFIX = "bb.sql-editor-tab";

type PersistentTab = Pick<
  SQLEditorTab,
  "id" | "title" | "connection" | "mode" | "worksheet" | "status"
>;

type LegacyStoredTab = PersistentTab & { statement?: string };

export const migrateLegacyCache = async () => {
  const me = useCurrentUserV1();
  const userUID = computed(() => extractUserId(me.value.name));

  const keyNamespace = computed(
    () => `${LOCAL_STORAGE_KEY_PREFIX}.${userUID.value}`
  );
  const extendedTabStore = useExtendedTabStore();
  const tabIdListKey = computed(() => `${keyNamespace.value}.tab-id-list`);
  const tabIdListMapByProject = useDynamicLocalStorage<
    Record<string, string[]>
  >(tabIdListKey, {});

  const getStorage = () => {
    return new WebStorageHelper(keyNamespace.value);
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
    const tab = reactive<SQLEditorTab>({
      ...defaultSQLEditorTab(),
      // Ignore extended fields stored in localStorage since they are migrated
      // to extendedTabStore.
      ...omit(stored, EXTENDED_TAB_FIELDS),
      id,
    });
    if (tab.mode !== DEFAULT_SQL_EDITOR_TAB_MODE) {
      // Do not enter ADMIN mode initially
      tab.mode = DEFAULT_SQL_EDITOR_TAB_MODE;
    }

    await extendedTabStore.fetchExtendedTab(tab, () => {
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

  const entries = [...Object.entries(tabIdListMapByProject.value)];
  for (const [project, tabIds] of entries) {
    const draftTabList = useDynamicLocalStorage<SQLEditorTab[]>(
      computed(
        () =>
          `${LOCAL_STORAGE_KEY_PREFIX}.${project}.${userUID.value}.draft-tab-list`
      ),
      [],
      localStorage
    );

    for (const tabId of tabIds) {
      const exist = draftTabList.value.find((draft) => draft.id === tabId);
      if (exist) {
        continue;
      }
      const tab = await loadStoredTab(tabId);
      await extendedTabStore.deleteExtendedTab(tabId);
      if (!tab) {
        continue;
      }
      draftTabList.value.push(tab);
    }

    delete tabIdListMapByProject.value[project];
  }
};

export const migrateDraftsFromCache = async (project: string) => {
  const me = useCurrentUserV1();
  const userUID = computed(() => extractUserId(me.value.name));
  const worksheetStore = useWorkSheetStore();

  const viewStateByTab = useDynamicLocalStorage<
    Map</* tab.id */ string, EditorPanelViewState>
  >(
    computed(() => `bb.sql-editor-tab-state.${userUID.value}`),
    new Map()
  );

  const keyNamespace = computed(
    () => `${LOCAL_STORAGE_KEY_PREFIX}.${project}.${userUID.value}`
  );

  const draftTabList = useDynamicLocalStorage<SQLEditorTab[]>(
    computed(() => `${keyNamespace.value}.draft-tab-list`),
    [],
    localStorage,
    {
      listenToStorageChanges: true,
    }
  );

  const drafts = [...draftTabList.value];
  for (const draft of drafts) {
    const tab = {
      ...defaultSQLEditorTab(),
      ...omitBy(draft, isUndefined),
    };
    const viewState = viewStateByTab.value.get(tab.id);
    if ((!viewState || viewState.view === "CODE") && tab.statement) {
      // only store the draft with content
      try {
        const connection = await extractWorksheetConnection({
          database: tab.connection.database,
        });
        await worksheetStore.createWorksheet(
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
    const index = draftTabList.value.findIndex((d) => d.id === draft.id);
    if (index >= 0) {
      draftTabList.value.splice(index, 1);
    }
  }
};

export const migrateTabViewState = (project: string) => {
  const me = useCurrentUserV1();
  const userUID = computed(() => extractUserId(me.value.name));

  const keyNamespace = computed(
    () => `${LOCAL_STORAGE_KEY_PREFIX}.${project}.${userUID.value}`
  );

  const viewStateByTab = useDynamicLocalStorage<
    Map</* tab.id */ string, EditorPanelViewState>
  >(
    computed(() => `bb.sql-editor-tab-state.${userUID.value}`),
    new Map()
  );

  const openTmpTabList = useDynamicLocalStorage<PersistentTab[]>(
    computed(() => `${keyNamespace.value}.opening-tab-list`),
    [],
    localStorage,
    {
      listenToStorageChanges: false,
    }
  );

  for (const openedTab of openTmpTabList.value) {
    const viewState = viewStateByTab.value.get(openedTab.id);
    if (viewState) {
      Object.assign(openedTab, { viewState });
    }
    viewStateByTab.value.delete(openedTab.id);
  }
};
