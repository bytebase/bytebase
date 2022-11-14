<template>
  <slot />
</template>

<script lang="ts" setup>
import { uniqBy } from "lodash-es";
import { reactive, onMounted, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  useCurrentUser,
  useDatabaseStore,
  useSQLEditorStore,
  useTabStore,
  useSheetStore,
  useDebugStore,
  useInstanceStore,
  pushNotification,
} from "@/store";
import type { Instance, Database, Connection, ConnectionAtom } from "@/types";
import { ConnectionTreeState, UNKNOWN_ID, DEFAULT_PROJECT_ID } from "@/types";
import {
  emptyConnection,
  idFromSlug,
  mapConnectionAtom,
  sheetSlug as makeSheetSlug,
  connectionSlug as makeConnectionSlug,
  isSheetReadable,
  isSameConnection,
  isTempTab,
} from "@/utils";
import { useI18n } from "vue-i18n";

type LocalState = {
  instanceList: Instance[];
  databaseList: Database[];
};

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const state = reactive<LocalState>({
  instanceList: [],
  databaseList: [],
});

const currentUser = useCurrentUser();
const instanceStore = useInstanceStore();
const databaseStore = useDatabaseStore();
const sqlEditorStore = useSQLEditorStore();
const tabStore = useTabStore();
const sheetStore = useSheetStore();

const prepareAccessibleConnectionByProject = async () => {
  // It will also be called when user logout
  if (currentUser.value.id === UNKNOWN_ID) {
    return;
  }
  instanceStore.fetchInstanceList();

  // `databaseList` is the database list accessible by current user.
  // Only accessible instances and databases will be listed in the tree.
  const databaseList = (
    await databaseStore.fetchDatabaseList({
      syncStatus: "OK",
    })
  ).filter((db) => db.project.id !== DEFAULT_PROJECT_ID);
  state.instanceList = uniqBy(
    databaseList.map((db) => db.instance),
    (instance) => instance.id
  );
  state.databaseList = databaseList;
};

const prepareSQLEditorContext = async () => {
  let connectionTree: ConnectionAtom[] = [];

  const { instanceList, databaseList } = state;
  connectionTree = instanceList.map(mapConnectionAtom("instance", 0));

  for (const instance of instanceList) {
    const instanceItem = connectionTree.find(
      (item: ConnectionAtom) => item.id === instance.id
    )!;

    instanceItem.children = databaseList
      .filter((db) => db.instance.id === instance.id)
      .map(mapConnectionAtom("database", instance.id));

    sqlEditorStore.connectionTree.data = connectionTree;
  }

  // Won't fetch tableList for every database here.
  // Will fetch them asynchronously only when a database node opens.
};

const prepareSheet = async () => {
  const sheetSlug = (route.params.sheetSlug as string) || "";
  const sheetId = idFromSlug(sheetSlug);
  if (Number.isNaN(sheetId)) {
    return false;
  }
  sqlEditorStore.isFetchingSheet = true;
  const sheet = await sheetStore.getOrFetchSheetById(sheetId);
  sqlEditorStore.isFetchingSheet = false;
  if (sheet.id === UNKNOWN_ID) {
    return false;
  }
  if (!isSheetReadable(sheet, currentUser.value)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.access-denied"),
    });
    return false;
  }

  const openingSheetTab = tabStore.tabList.find(
    (tab) => tab.sheetId === sheet.id
  );
  if (openingSheetTab) {
    // Switch to a sheet tab if it's open already.
    tabStore.setCurrentTabId(openingSheetTab.id);
  } else {
    // Open the sheet in a "temp" tab otherwise.
    tabStore.selectOrAddTempTab();
    tabStore.updateCurrentTab({
      sheetId: sheet.id,
      name: sheet.name,
      statement: sheet.statement,
      isSaved: true,
      connection: {
        ...emptyConnection(),
        instanceId: sheet.database?.instanceId || UNKNOWN_ID,
        databaseId: sheet.databaseId || UNKNOWN_ID,
      },
    });
  }

  return true;
};

const prepareConnectionSlug = async () => {
  const connectionSlug = (route.params.connectionSlug as string) || "";
  const [instanceSlug, databaseSlug = ""] = connectionSlug.split("_");
  const instanceId = idFromSlug(instanceSlug);
  const databaseId = idFromSlug(databaseSlug);

  if (Number.isNaN(instanceId) && Number.isNaN(databaseId)) {
    return false;
  }

  const maybeOpenNewTab = (connection: Connection) => {
    const tab = tabStore.currentTab;
    if (tab.sheetId) {
      // Don't touch a saved sheet.
      tabStore.selectOrAddTempTab();
      return;
    }
    if (isTempTab(tab)) {
      // Override current tab if it's a temp tab.
      return;
    }
    if (isSameConnection(tab.connection, connection)) {
      // Stay on current tab if its connection and target connection are equal.
      return;
    }
    // Select or add a temp tab otherwise.
    tabStore.selectOrAddTempTab();
  };

  if (Number.isNaN(databaseId)) {
    // connected to instance
    const connection = await sqlEditorStore.fetchConnectionByInstanceId(
      instanceId
    );
    maybeOpenNewTab(connection);
    tabStore.updateCurrentTab({ connection });
  } else {
    // connected to db
    const connection =
      await sqlEditorStore.fetchConnectionByInstanceIdAndDatabaseId(
        instanceId,
        databaseId
      );
    maybeOpenNewTab(connection);
    tabStore.updateCurrentTab({ connection });
  }
  return true;
};

// Get sheetId from query.
const setConnectionFromQuery = async () => {
  // Priority:
  // 1. idFromSlug in sheetSlug
  // 2. instanceId and databaseId in connectionSlug
  // 3. disconnected

  if (await prepareSheet()) {
    return;
  }

  if (await prepareConnectionSlug()) {
    return;
  }

  // Keep disconnected otherwise
  // We don't need to `selectOrAddTempTab` here since we already have the
  // default tab.
};

// Keep the URL synced with connection
// 1. /sql-editor/sheet/{sheet_slug}  - saved sheets
// 2. /sql-editor/{connection_slug}   - unsaved tabs
// 3. /sql-editor                     - clean tabs
const syncURLWithConnection = () => {
  const connection = computed(() => tabStore.currentTab.connection);
  watch(
    [
      () => connection.value.instanceId,
      () => connection.value.databaseId,
      () => tabStore.currentTab.sheetId,
    ],
    ([instanceId, databaseId, sheetId]) => {
      if (sheetId && sheetId !== UNKNOWN_ID) {
        const sheet = sheetStore.sheetById.get(sheetId);
        if (sheet) {
          router.replace({
            name: "sql-editor.share",
            params: {
              sheetSlug: makeSheetSlug(sheet),
            },
          });
          return;
        }
      }
      if (instanceId !== UNKNOWN_ID) {
        const instance = instanceStore.getInstanceById(instanceId);
        const database = databaseStore.getDatabaseById(databaseId); // might be <<Unknown database>> here
        router.replace({
          name: "sql-editor.detail",
          params: {
            connectionSlug: makeConnectionSlug(instance, database),
          },
        });
        return;
      }
      router.replace({
        name: "sql-editor.home",
      });
    },
    { immediate: true }
  );
};

onMounted(async () => {
  if (sqlEditorStore.connectionTree.state === ConnectionTreeState.UNSET) {
    sqlEditorStore.connectionTree.state = ConnectionTreeState.LOADING;
    await prepareAccessibleConnectionByProject();
    await prepareSQLEditorContext();
    sqlEditorStore.connectionTree.state = ConnectionTreeState.LOADED;
  }

  watch(currentUser, (user) => {
    if (user.id === UNKNOWN_ID) {
      // Cleanup when user signed out
      sqlEditorStore.connectionTree.data = [];
      sqlEditorStore.connectionTree.state = ConnectionTreeState.UNSET;

      tabStore.reset();
    }
  });

  await setConnectionFromQuery();
  await sqlEditorStore.fetchQueryHistoryList();
  await useDebugStore().fetchDebug();

  syncURLWithConnection();
});
</script>
