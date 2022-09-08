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
    const sheet = await sheetStore.fetchSheetById(sheetId);
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
    setConnectionContextFromCurrentTab();
    useSQLEditorStore().setSQLEditorState({
      sharedSheet: sheet,
    });
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
