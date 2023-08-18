<template>
  <slot />
</template>

<script lang="ts" setup>
import { uniqBy } from "lodash-es";
import { onMounted, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import {
  useSQLEditorStore,
  useTabStore,
  pushNotification,
  useConnectionTreeStore,
  useProjectV1Store,
  useCurrentUserV1,
  useSheetV1Store,
  useInstanceV1Store,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useUserStore,
} from "@/store";
import { getInstanceAndDatabaseId } from "@/store/modules/v1/common";
import { usePolicyV1Store } from "@/store/modules/v1/policy";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import {
  Connection,
  ConnectionAtom,
  ConnectionTreeMode,
  CoreTabInfo,
  DEFAULT_PROJECT_V1_NAME,
  TabMode,
  UNKNOWN_USER_NAME,
} from "@/types";
import { ConnectionTreeState, UNKNOWN_ID } from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import {
  emptyConnection,
  idFromSlug,
  sheetNameFromSlug,
  sheetSlugV1,
  connectionV1Slug as makeConnectionV1Slug,
  isSheetReadableV1,
  isDatabaseV1Queryable,
  getSuggestedTabNameFromConnection,
  isSimilarTab,
  hasWorkspacePermissionV1,
} from "@/utils";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const currentUserV1 = useCurrentUserV1();
const instanceStore = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const policyV1Store = usePolicyV1Store();
const sqlEditorStore = useSQLEditorStore();
const connectionTreeStore = useConnectionTreeStore();
const tabStore = useTabStore();
const sheetV1Store = useSheetV1Store();

const prepareAccessControlPolicy = async () => {
  connectionTreeStore.accessControlPolicyList =
    await policyV1Store.fetchPolicies({
      policyType: PolicyType.WORKSPACE_IAM,
      resourceType: PolicyResourceType.WORKSPACE,
    });
  await policyV1Store.fetchPolicies({
    resourceType: PolicyResourceType.ENVIRONMENT,
    policyType: PolicyType.DISABLE_COPY_DATA,
  });
};

const prepareAccessibleDatabaseList = async () => {
  // It will also be called when user logout
  if (currentUserV1.value.name === UNKNOWN_USER_NAME) {
    return;
  }
  instanceStore.fetchInstanceList();

  // `databaseList` is the database list accessible by current user.
  // Only accessible instances and databases will be listed in the tree.
  const databaseList = (
    await databaseStore.searchDatabaseList({
      parent: "instances/-",
    })
  ).filter(
    (db) =>
      db.syncState === State.ACTIVE &&
      isDatabaseV1Queryable(db, currentUserV1.value)
  );
  connectionTreeStore.tree.databaseList = databaseList;
};

const prepareConnectionTree = async () => {
  if (connectionTreeStore.tree.mode === ConnectionTreeMode.INSTANCE) {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-database",
        currentUserV1.value.userRole
      )
    ) {
      connectionTreeStore.tree.mode = ConnectionTreeMode.PROJECT;
      return;
    }
    const { databaseList } = connectionTreeStore.tree;
    const instanceList = uniqBy(
      databaseList.map((db) => db.instanceEntity),
      (instance) => instance.uid
    );
    const connectionTree = instanceList.map((instance) => {
      const node = connectionTreeStore.mapAtom(instance, "instance", "0");
      return node;
    });

    for (const instance of instanceList) {
      const instanceItem = connectionTree.find(
        (item: ConnectionAtom) => item.id === instance.uid
      )!;

      instanceItem.children = databaseList
        .filter((db) => db.instanceEntity.uid === instance.uid)
        .map((db) => {
          const node = connectionTreeStore.mapAtom(
            db,
            "database",
            instance.uid
          );
          node.disabled = !isDatabaseV1Queryable(db, currentUserV1.value);
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
    const databaseList = connectionTreeStore.tree.databaseList.filter((db) => {
      return db.project !== DEFAULT_PROJECT_V1_NAME;
    });
    const projectList = uniqBy(
      databaseList.map((db) => db.projectEntity),
      (project) => project.uid
    );

    const projectAtomList = projectList.map((project) => {
      const node = connectionTreeStore.mapAtom(project, "project", "0");
      return node;
    });

    projectAtomList.forEach((projectAtom) => {
      projectAtom.children = databaseList
        .filter((db) => db.projectEntity.uid === projectAtom.id)
        .map((db) => {
          const node = connectionTreeStore.mapAtom(
            db,
            "database",
            projectAtom.id
          );
          node.disabled = !isDatabaseV1Queryable(db, currentUserV1.value);
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
  if (!sheetSlug) {
    return false;
  }

  const sheetName = sheetNameFromSlug(sheetSlug);
  const openingSheetTab = tabStore.tabList.find(
    (tab) => tab.sheetName == sheetName
  );

  sqlEditorStore.isFetchingSheet = true;
  const sheet = await sheetV1Store.getOrFetchSheetByName(sheetName);
  sqlEditorStore.isFetchingSheet = false;

  if (!sheet) {
    if (openingSheetTab) {
      // If a sheet is open in a tab but it returns 404 NOT_FOUND
      // that means the sheet has been deleted somewhere else.
      // We need to turn the sheet to an unsaved tab.
      openingSheetTab.sheetName = undefined;
      openingSheetTab.isSaved = false;
    }
    return false;
  }
  if (!isSheetReadableV1(sheet)) {
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
    tabStore.addTab();
  }

  let insId = String(UNKNOWN_ID);
  let dbId = String(UNKNOWN_ID);
  if (sheet.database) {
    const [instanceName, databaseId] = getInstanceAndDatabaseId(sheet.database);
    const ins = await useInstanceV1Store().getOrFetchInstanceByName(
      `instances/${instanceName}`
    );
    insId = ins.uid;
    dbId = databaseId;
  }

  tabStore.updateCurrentTab({
    sheetName,
    name: sheet.title,
    statement: new TextDecoder().decode(sheet.content),
    isSaved: true,
    connection: {
      ...emptyConnection(),
      // TODO: legacy instance id.
      instanceId: insId,
      databaseId: dbId,
    },
  });

  return true;
};

const prepareConnectionSlug = async () => {
  const connectionSlug = (route.params.connectionSlug as string) || "";
  const [instanceSlug, databaseSlug = ""] = connectionSlug.split("_");
  const instanceId = Number(idFromSlug(instanceSlug));
  const databaseId = Number(idFromSlug(databaseSlug));

  if (Number.isNaN(instanceId) && Number.isNaN(databaseId)) {
    return false;
  }

  const connect = (connection: Connection) => {
    const tab = tabStore.currentTab;
    if (tab.sheetName) {
      // Don't touch a saved sheet.
      tabStore.addTab();
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
    const name = getSuggestedTabNameFromConnection(target.connection);
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
      String(instanceId)
    );
    connect(connection);
  } else {
    // connected to db
    const connection =
      await connectionTreeStore.fetchConnectionByInstanceIdAndDatabaseId(
        String(instanceId),
        String(databaseId)
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
      () => tabStore.currentTab.sheetName,
    ],
    ([instanceId, databaseId, sheetName]) => {
      if (sheetName) {
        const sheet = sheetV1Store.getSheetByName(sheetName);
        if (sheet) {
          router.replace({
            name: "sql-editor.share",
            params: {
              sheetSlug: sheetSlugV1(sheet),
            },
          });
        } else {
          // A sheet is not found, fallback to an unsaved tab.
          tabStore.updateCurrentTab({
            sheetName: undefined,
            isSaved: false,
          });
        }
        return;
      }
      if (instanceId !== String(UNKNOWN_ID)) {
        const instance = instanceStore.getInstanceByUID(instanceId);
        const database = databaseStore.getDatabaseByUID(databaseId); // might be <<Unknown database>> here
        // Sometimes the instance and/or the database might be <<Unknown>> since
        // they might be deleted somewhere else during the life of the page.
        // So we need to sync the connection values for cleaning up to prevent
        // exceptions.
        tabStore.updateCurrentTab({
          connection: {
            instanceId: instance.uid,
            databaseId: database.uid,
          },
        });
        router.replace({
          name: "sql-editor.detail",
          params: {
            connectionSlug: makeConnectionV1Slug(instance, database),
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
  await useUserStore().fetchUserList();

  if (connectionTreeStore.tree.state === ConnectionTreeState.UNSET) {
    connectionTreeStore.tree.state = ConnectionTreeState.LOADING;
    // Initialize project list state for iam policy.
    await useProjectV1Store().fetchProjectList(true /* include archived */);
    // Initialize environment list for composing.
    await useEnvironmentV1Store().fetchEnvironments(
      true /* include archived */
    );
    await usePolicyV1Store().getOrFetchPolicyByName("policies/WORKSPACE_IAM");
    await prepareAccessControlPolicy();
    await prepareAccessibleDatabaseList();
    connectionTreeStore.tree.state = ConnectionTreeState.LOADED;
  }

  watch(() => connectionTreeStore.tree.mode, prepareConnectionTree, {
    immediate: true,
  });

  watch(
    () => currentUserV1.value.name,
    (name) => {
      if (name === UNKNOWN_USER_NAME) {
        // Cleanup when user signed out
        connectionTreeStore.tree.data = [];
        connectionTreeStore.tree.state = ConnectionTreeState.UNSET;

        tabStore.reset();
      }
    }
  );

  await setConnectionFromQuery();
  await sqlEditorStore.fetchQueryHistoryList();
  await useSettingV1Store().fetchSettingList();

  syncURLWithConnection();
});
</script>
