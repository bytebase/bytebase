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
  usePolicyStore,
  useConnectionTreeStore,
} from "@/store";
import {
  Database,
  Connection,
  ConnectionAtom,
  ConnectionTreeMode,
  CoreTabInfo,
  TabMode,
} from "@/types";
import { ConnectionTreeState, UNKNOWN_ID, DEFAULT_PROJECT_ID } from "@/types";
import {
  emptyConnection,
  idFromSlug,
  sheetSlug as makeSheetSlug,
  connectionSlug as makeConnectionSlug,
  isSheetReadable,
  isDatabaseAccessible,
  getDefaultTabNameFromConnection,
  isSimilarTab,
} from "@/utils";
import { useI18n } from "vue-i18n";

type LocalState = {
  databaseList: Database[];
};

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const state = reactive<LocalState>({
  databaseList: [],
});

const currentUser = useCurrentUser();
const instanceStore = useInstanceStore();
const databaseStore = useDatabaseStore();
const policyStore = usePolicyStore();
const sqlEditorStore = useSQLEditorStore();
const connectionTreeStore = useConnectionTreeStore();
const tabStore = useTabStore();
const sheetStore = useSheetStore();

const prepareAccessControlPolicy = async () => {
  connectionTreeStore.accessControlPolicyList =
    await policyStore.fetchPolicyListByResourceTypeAndPolicyType(
      "database",
      "bb.policy.access-control"
    );
};

const prepareAccessibleDatabaseList = async () => {
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
  ).filter((db) =>
    isDatabaseAccessible(
      db,
      connectionTreeStore.accessControlPolicyList,
      currentUser.value
    )
  );
  state.databaseList = databaseList;
};

const prepareConnectionTree = async () => {
  if (connectionTreeStore.tree.mode === ConnectionTreeMode.INSTANCE) {
    const { databaseList } = state;
    const instanceList = uniqBy(
      databaseList.map((db) => db.instance),
      (instance) => instance.id
    );
    const connectionTree = instanceList.map((instance) => {
      const node = connectionTreeStore.mapAtom(instance, "instance", 0);
      return node;
    });

    for (const instance of instanceList) {
      const instanceItem = connectionTree.find(
        (item: ConnectionAtom) => item.id === instance.id
      )!;

      instanceItem.children = databaseList
        .filter((db) => db.instance.id === instance.id)
        .map((db) => {
          const node = connectionTreeStore.mapAtom(db, "database", instance.id);
          node.disabled = !isDatabaseAccessible(
            db,
            connectionTreeStore.accessControlPolicyList,
            currentUser.value
          );
          if (node.disabled) {
            // If a database node is not accessible
            // it's not expandable either.
            node.isLeaf = true;
          }
          return node;
        });
    }
    connectionTreeStore.tree.data = connectionTree;
  } else {
    const databaseList = state.databaseList.filter((db) => {
      return db.project.id !== DEFAULT_PROJECT_ID;
    });
    const projectList = uniqBy(
      databaseList.map((db) => db.project),
      (project) => project.id
    );

    const projectAtomList = projectList.map((project) => {
      const node = connectionTreeStore.mapAtom(project, "project", 0);
      return node;
    });

    projectAtomList.forEach((projectAtom) => {
      projectAtom.children = databaseList
        .filter((db) => db.project.id === projectAtom.id)
        .map((db) => {
          const node = connectionTreeStore.mapAtom(
            db,
            "database",
            projectAtom.id
          );
          node.disabled = !isDatabaseAccessible(
            db,
            connectionTreeStore.accessControlPolicyList,
            currentUser.value
          );
          if (node.disabled) {
            // If a database node is not accessible
            // it's not expandable either.
            node.isLeaf = true;
          }
          return node;
        });
    });

    connectionTreeStore.tree.data = projectAtomList;
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
  const openingSheetTab = tabStore.tabList.find(
    (tab) => tab.sheetId === sheetId
  );

  sqlEditorStore.isFetchingSheet = true;
  const sheet = await sheetStore.getOrFetchSheetById(sheetId);
  sqlEditorStore.isFetchingSheet = false;

  if (sheet.id === UNKNOWN_ID) {
    if (openingSheetTab) {
      // If a sheet is open in a tab but it returns 404 NOT_FOUND
      // that means the sheet has been deleted somewhere else.
      // We need to turn the sheet to an unsaved tab.
      openingSheetTab.sheetId = undefined;
      openingSheetTab.isSaved = false;
    }
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

  const connect = (connection: Connection) => {
    const tab = tabStore.currentTab;
    if (tab.sheetId) {
      // Don't touch a saved sheet.
      tabStore.selectOrAddTempTab();
      return;
    }
    const target: CoreTabInfo = {
      connection,
      mode: TabMode.ReadOnly,
    };

    if (isSimilarTab(target, tabStore.currentTab)) {
      // Don't go further if the connection doesn't change.
      return;
    }
    const name = getDefaultTabNameFromConnection(target.connection);
    tabStore.selectOrAddSimilarTab(
      target,
      /* beside */ false,
      /* defaultTabName */ name
    );
    tabStore.updateCurrentTab(target);
  };

  if (Number.isNaN(databaseId)) {
    // connected to instance
    const connection = await connectionTreeStore.fetchConnectionByInstanceId(
      instanceId
    );
    connect(connection);
  } else {
    // connected to db
    const connection =
      await connectionTreeStore.fetchConnectionByInstanceIdAndDatabaseId(
        instanceId,
        databaseId
      );
    connect(connection);
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
        } else {
          // A sheet is not found, fallback to an unsaved tab.
          tabStore.updateCurrentTab({
            sheetId: undefined,
            isSaved: false,
          });
        }
        return;
      }
      if (instanceId !== UNKNOWN_ID) {
        const instance = instanceStore.getInstanceById(instanceId);
        const database = databaseStore.getDatabaseById(databaseId); // might be <<Unknown database>> here
        // Sometimes the instance and/or the database might be <<Unknown>> since
        // they might be deleted somewhere else during the life of the page.
        // So we need to sync the connection values for cleaning up to prevent
        // exceptions.
        tabStore.updateCurrentTab({
          connection: {
            instanceId: instance.id,
            databaseId: database.id,
          },
        });
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
  if (connectionTreeStore.tree.state === ConnectionTreeState.UNSET) {
    connectionTreeStore.tree.state = ConnectionTreeState.LOADING;
    await prepareAccessControlPolicy();
    await prepareAccessibleDatabaseList();
    await prepareConnectionTree();
    connectionTreeStore.tree.state = ConnectionTreeState.LOADED;
  }

  watch(() => connectionTreeStore.tree.mode, prepareConnectionTree);

  watch(currentUser, (user) => {
    if (user.id === UNKNOWN_ID) {
      // Cleanup when user signed out
      connectionTreeStore.tree.data = [];
      connectionTreeStore.tree.state = ConnectionTreeState.UNSET;

      tabStore.reset();
    }
  });

  await setConnectionFromQuery();
  await sqlEditorStore.fetchQueryHistoryList();
  await useDebugStore().fetchDebug();

  syncURLWithConnection();
});
</script>
