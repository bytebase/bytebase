<template>
  <slot />
</template>

<script lang="ts" setup>
import { uniqBy } from "lodash-es";
import { reactive, onMounted } from "vue";
import { useRoute } from "vue-router";
import {
  useCurrentUser,
  useDatabaseStore,
  useSQLEditorStore,
  useTabStore,
  useSheetStore,
  useDebugStore,
  useInstanceStore,
} from "@/store";
import {
  Instance,
  Database,
  ConnectionAtom,
  UNKNOWN_ID,
  DEFAULT_PROJECT_ID,
} from "@/types";
import { emptyConnection, mapConnectionAtom } from "@/utils";

type LocalState = {
  instanceList: Instance[];
  databaseList: Database[];
};

const route = useRoute();

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
  const databaseList = await databaseStore.fetchDatabaseList({
    syncStatus: "OK",
  });
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

    sqlEditorStore.setConnectionTree(connectionTree);
  }

  // Won't fetch tableList for every database here.
  // Will fetch them asynchronously only when a database node opens.
};

const prepareSheet = async () => {
  // TODO(Jim): use standard slug format instead of "_" joint format.

  let sheetId = parseInt(route.query.sheetId as string, 10);
  if (Number.isNaN(sheetId)) {
    const sheetSlug = (route.params.sheetSlug as string) || "";
    sheetId = parseInt(sheetSlug.split("_").pop()!, 10);
  }
  if (Number.isNaN(sheetId)) {
    return false;
  }
  sqlEditorStore.isFetchingSheet = true;
  const sheet = await sheetStore.getOrFetchSheetById(sheetId);
  sqlEditorStore.isFetchingSheet = false;
  if (sheet.id === UNKNOWN_ID) {
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
  }
  tabStore.updateCurrentTab({
    sheetId: sheet.id,
    name: sheet.name,
    statement: sheet.statement,
    isSaved: true,
    connection: {
      ...emptyConnection(),
      projectId: sheet.database?.projectId || DEFAULT_PROJECT_ID,
      instanceId: sheet.database?.instanceId || UNKNOWN_ID,
      databaseId: sheet.databaseId || UNKNOWN_ID,
    },
  });

  // TODO(Jim): remove this after refactor `sharedSheet`
  useSQLEditorStore().setSQLEditorState({
    sharedSheet: sheet,
  });

  return true;
};

const prepareConnectionSlug = async () => {
  // TODO(Jim): use standard slug format instead of "_" joint format.

  const connectionSlug = (route.params.connectionSlug as string) || "";
  // [instanceName, instanceId, databaseName, databaseId]
  const parts = connectionSlug.split("_");
  const instanceId = parseInt(parts[1], 10);
  const databaseId = parseInt(parts[3], 10);
  if (Number.isNaN(instanceId) || Number.isNaN(databaseId)) {
    return false;
  }

  tabStore.selectOrAddTempTab();
  const connection =
    await sqlEditorStore.fetchConnectionByInstanceIdAndDatabaseId({
      instanceId,
      databaseId,
    });
  tabStore.updateCurrentTab({
    connection,
  });
  return true;
};

// Get sheetId from query.
const setConnectionFromQuery = async () => {
  // Priority:
  // 1. sheetId in query
  // 2. idFromSlug in sheetSlug
  // 3. instanceId and databaseId in connectionSlug
  // 4. disconnected

  if (await prepareSheet()) {
    return;
  }

  if (await prepareConnectionSlug()) {
    return;
  }

  tabStore.selectOrAddTempTab();
  // Keep disconnected otherwise
};

onMounted(async () => {
  sqlEditorStore.setConnectionContext({ isLoadingTree: true });
  await prepareAccessibleConnectionByProject();
  await prepareSQLEditorContext();
  sqlEditorStore.setConnectionContext({ isLoadingTree: false });

  await setConnectionFromQuery();
  await sqlEditorStore.fetchQueryHistoryList();
  await useDebugStore().fetchDebug();
});
</script>
