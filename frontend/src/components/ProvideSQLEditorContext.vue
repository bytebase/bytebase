<template>
  <slot />
</template>

<script lang="ts" setup>
import { reactive, onMounted } from "vue";
import { useRoute } from "vue-router";
import { useSQLEditorConnection } from "@/composables/useSQLEditorConnection";
import {
  useCurrentUser,
  useDatabaseStore,
  useInstanceStore,
  useProjectStore,
  useTableStore,
  useSQLEditorStore,
  useTabStore,
  useSheetStore,
} from "@/store";
import {
  Instance,
  Database,
  Table,
  ConnectionAtom,
  ConnectionAtomType,
  UNKNOWN_ID,
  InstanceId,
  DatabaseId,
  Project,
} from "@/types";

const route = useRoute();

const state = reactive<{
  projectList: Project[];
  instanceIdList: Map<InstanceId, Instance["name"]>;
  databaseIdList: Map<DatabaseId, Database["name"]>;
}>({
  projectList: [],
  instanceIdList: new Map(),
  databaseIdList: new Map(),
});

const currentUser = useCurrentUser();
const projectStore = useProjectStore();
const databaseStore = useDatabaseStore();
const tableStore = useTableStore();
const sqlEditorStore = useSQLEditorStore();
const tabStore = useTabStore();
const sheetStore = useSheetStore();
const { setConnectionContextFromCurrentTab } = useSQLEditorConnection();

const prepareAccessibleConnectionByProject = async () => {
  // It will also be called when user logout
  if (currentUser.value.id != UNKNOWN_ID) {
    state.projectList = await projectStore.fetchProjectListByUser({
      userId: currentUser.value.id,
    });
  }

  const promises = state.projectList.map(async (project) => {
    const databaseList = await databaseStore.fetchDatabaseListByProjectId(
      project.id
    );
    if (databaseList.length >= 0) {
      databaseList.forEach((database: Database) => {
        state.databaseIdList.set(database.id, database.name);
        state.instanceIdList.set(database.instance.id, database.instance.name);
      });
    }
  });

  await Promise.all(promises);
};

const prepareSQLEditorContext = async () => {
  sqlEditorStore.setConnectionContext({ isLoadingTree: true });
  let connectionTree = [];

  const mapConnectionAtom =
    (type: ConnectionAtomType, parentId: number) =>
    (item: Instance | Database | Table) => {
      const connectionAtom: ConnectionAtom = {
        parentId,
        id: item.id,
        key: `${type}-${item.id}`,
        label: item.name,
        type,
      };

      return connectionAtom;
    };

  const instanceList = await useInstanceStore().fetchInstanceList();
  const filteredInstanceList = instanceList.filter((instance: Instance) =>
    state.instanceIdList.has(instance.id)
  );
  connectionTree = filteredInstanceList.map(mapConnectionAtom("instance", 0));

  for (const instance of filteredInstanceList) {
    const databaseList = await databaseStore.fetchDatabaseListByInstanceId(
      instance.id
    );

    const instanceItem = connectionTree.find(
      (item: ConnectionAtom) => item.id === instance.id
    )!;
    const filteredDatabaseList = databaseList.filter((database: Database) =>
      state.databaseIdList.has(database.id)
    );

    instanceItem.children = filteredDatabaseList.map(
      mapConnectionAtom("database", instance.id)
    );

    for (const db of filteredDatabaseList) {
      const tableList = await tableStore.fetchTableListByDatabaseId(db.id);

      const databaseItem = instanceItem.children!.find(
        (item: ConnectionAtom) => item.id === db.id
      )!;

      databaseItem.children = tableList.map(mapConnectionAtom("table", db.id));
    }
  }

  sqlEditorStore.setConnectionTree(connectionTree);
  sqlEditorStore.setConnectionContext({ isLoadingTree: false });
  sqlEditorStore.fetchQueryHistoryList();
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
    });
    setConnectionContextFromCurrentTab();
    useSQLEditorStore().setSQLEditorState({
      sharedSheet: sheet,
      shouldSetContent: true,
    });
  }
};

onMounted(async () => {
  await prepareAccessibleConnectionByProject();
  await prepareSQLEditorContext();
  await prepareSheetFromQuery();
});
</script>
