<template>
  <slot />
</template>

<script lang="ts" setup>
import { uniqBy } from "lodash-es";
import { reactive, onMounted } from "vue";
import { useRoute } from "vue-router";
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
import { isReplaceableTab, mapConnectionAtom } from "@/utils";

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
const { setConnectionContextFromCurrentTab } = useSQLEditorConnection();

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
    const sheet = await sheetStore.fetchSheetById(sheetId);
    sqlEditorStore.isFetchingSheet = false;

    // Opening a stored tab.
    if (sheet.id !== UNKNOWN_ID) {
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
        // and it's newly open (isSaved === true && statement === "")
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
        isSaved: true,
      });
      setConnectionContextFromCurrentTab(sheet);
      useSQLEditorStore().setSQLEditorState({
        sharedSheet: sheet,
        shouldSetContent: true,
      });
    }
  }
};

onMounted(async () => {
  sqlEditorStore.setConnectionContext({ isLoadingTree: true });
  await prepareAccessibleConnectionByProject();
  await prepareSQLEditorContext();
  sqlEditorStore.setConnectionContext({ isLoadingTree: false });

  await prepareSheetFromQuery();
  await sqlEditorStore.fetchQueryHistoryList();
  await useDebugStore().fetchDebug();
});
</script>
