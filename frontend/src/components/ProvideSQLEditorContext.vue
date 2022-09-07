<template>
  <slot />
</template>

<script lang="ts" setup>
import { uniqBy } from "lodash-es";
import { reactive, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useSQLEditorConnection } from "@/composables/useSQLEditorConnection";
import {
  useCurrentUser,
  useDatabaseStore,
  useSQLEditorStore,
  useTabStore,
  useSheetStore,
  useDebugStore,
  useInstanceStore,
} from "@/store";
import { Instance, Database, ConnectionAtom, UNKNOWN_ID } from "@/types";
import { connectionSlug, isReplaceableTab, mapConnectionAtom } from "@/utils";
import { useMounted } from "@/composables/useMounted";

type LocalState = {
  instanceList: Instance[];
  databaseList: Database[];
};

const route = useRoute();
const router = useRouter();

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
const { setConnectionContextFromCurrentTab } = useSQLEditorConnection();
const mounted = useMounted();

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

// Get sheetId from query.
const prepareSheetFromQuery = async () => {
  const sheetId = Number(route.query.sheetId);
  if (!Number.isNaN(sheetId)) {
    sqlEditorStore.isFetchingSheet = true;
    const sheet = await sheetStore.getOrFetchSheetById(sheetId);
    sqlEditorStore.isFetchingSheet = false;

    // Check if the tab is already open.
    const openSheetTab = tabStore.tabList.find(
      (tab) => tab.sheetId === sheet.id
    );
    if (openSheetTab) {
      // The sheet is already open in a tab. Switch to it.
      tabStore.setCurrentTabId(openSheetTab.id);
    } else {
      // Find a replaceable, and 'replaceable' means
      // It's not related to any sheets (sheetId === undefined)
      // and it's newly open (isModified === false && statement === "")
      const replaceableTab = tabStore.tabList.find(isReplaceableTab);
      if (replaceableTab) {
        // Open the sheet in the replaceable tab if we found one.
        tabStore.setCurrentTabId(replaceableTab.id);
      } else {
        // Open the sheet in a new tab otherwise.
        tabStore.addTab();
      }
    }

    tabStore.updateCurrentTab({
      sheetId: sheet.id,
      name: sheet.name,
      statement: sheet.statement,
      isModified: !!openSheetTab,
    });
    setConnectionContextFromCurrentTab();
    useSQLEditorStore().setSQLEditorState({
      sharedSheet: sheet,
      shouldSetContent: true,
    });
  }
};

onMounted(async () => {
  if (sqlEditorStore.connectionTree.length === 0) {
    // Won't rebuild the tree if already done.
    sqlEditorStore.isFetchingTree = true;
    await prepareAccessibleConnectionByProject();
    await prepareSQLEditorContext();
    sqlEditorStore.isFetchingTree = false;
  }
  await prepareSheetFromQuery();
  await sqlEditorStore.fetchQueryHistoryList();
  await useDebugStore().fetchDebug();
});

watch(currentUser, (user) => {
  if (user.id === UNKNOWN_ID) {
    // Clear SQL editor state when signed out.
    sqlEditorStore.$reset();
    tabStore.$reset();
    sheetStore.$reset();
  }
});

// Keep the URL synced with current connection context.
watch(
  [
    () => sheetStore.currentSheet,
    () => sqlEditorStore.connectionContext.instanceId,
    () => sqlEditorStore.connectionContext.databaseId,
  ],
  ([sheet, instanceId, databaseId]) => {
    if (!mounted.value) return;
    const routeArgs: any = {
      name: "sql-editor.home",
      params: {},
      query: {},
    };
    const database = useDatabaseStore().getDatabaseById(databaseId);
    if (sheet.id !== UNKNOWN_ID) {
      routeArgs.query.sheetId = sheet.id;
    } else if (database.id !== UNKNOWN_ID) {
      routeArgs.name = "sql-editor.detail";
      routeArgs.params.connectionSlug = connectionSlug(database);
    }
    router.replace(routeArgs);
  }
);
</script>
