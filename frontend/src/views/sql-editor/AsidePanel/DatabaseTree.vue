<template>
  <div
    v-if="treeStore.state === 'READY'"
    class="sql-editor-tree gap-y-1 h-full flex flex-col"
  >
    <div class="flex flex-row gap-x-0.5 px-1 items-center">
      <SearchBox v-model:searchPattern="searchPattern" class="flex-1" />
      <GroupingBar class="shrink-0" />
    </div>
    <div
      ref="treeContainerElRef"
      class="sql-editor-tree--tree flex-1 px-1 pb-1 text-sm overflow-hidden select-none"
      :data-height="treeContainerHeight"
    >
      <NTree
        ref="treeRef"
        v-model:expanded-keys="expandedKeys"
        :block-line="true"
        :data="treeStore.tree"
        :show-irrelevant-nodes="false"
        :selected-keys="selectedKeys"
        :pattern="mounted ? searchPattern : ''"
        :expand-on-click="true"
        :node-props="nodeProps"
        :virtual-scroll="true"
        :theme-overrides="{ nodeHeight: '21px' }"
        :render-label="renderLabel"
        @load="handleLoadSubTree"
      />
    </div>

    <NDropdown
      placement="bottom-start"
      trigger="manual"
      :x="dropdownPosition.x"
      :y="dropdownPosition.y"
      :options="dropdownOptions"
      :show="showDropdown"
      :on-clickoutside="handleClickoutside"
      @select="handleSelect"
    />

    <DatabaseHoverPanel
      :database="hoverNode?.meta.target as ComposedDatabase"
      :x="hoverPanelPosition.x"
      :y="hoverPanelPosition.y"
      class="ml-3"
    />
  </div>
  <div v-else class="flex justify-center items-center h-full space-x-2">
    <BBSpin />
    <span class="text-control text-sm">{{
      $t("sql-editor.loading-databases")
    }}</span>
  </div>
</template>

<script lang="ts" setup>
import { useElementSize, useMounted, useClipboard } from "@vueuse/core";
import { head } from "lodash-es";
import {
  NTree,
  NDropdown,
  type DropdownOption,
  type TreeOption,
} from "naive-ui";
import { storeToRefs } from "pinia";
import { ref, computed, nextTick, watch, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { SQL_EDITOR_DATABASE_MODULE } from "@/router/sqlEditor";
import {
  useActuatorV1Store,
  useCurrentUserV1,
  useDBSchemaV1Store,
  useDatabaseV1Store,
  useInstanceV1Store,
  useIsLoggedIn,
  pushNotification,
  useSQLEditorTabStore,
  resolveOpeningDatabaseListFromSQLEditorTabList,
  useSQLEditorTreeStore,
  idForSQLEditorTreeNodeTarget,
  useConnectionOfCurrentSQLEditorTab,
} from "@/store";
import type {
  ComposedDatabase,
  ComposedInstance,
  CoreSQLEditorTab,
  RichTableMetadata,
  SQLEditorTabMode,
  SQLEditorTreeNode,
  SQLEditorTreeNodeTarget,
  SQLEditorTreeNodeType,
} from "@/types";
import {
  ConnectableTreeNodeTypes,
  DEFAULT_SQL_EDITOR_TAB_MODE,
  ExpandableTreeNodeTypes,
  UNKNOWN_ID,
  instanceOfSQLEditorTreeNode,
  isConnectableSQLEditorTreeNode,
  languageOfEngineV1,
} from "@/types";
import {
  findAncestor,
  hasWorkspacePermissionV2,
  instanceV1AllowsCrossDatabaseQuery,
  instanceV1HasAlterSchema,
  instanceV1HasReadonlyMode,
  isDescendantOf,
  generateSimpleSelectAllStatement,
  tryConnectToCoreSQLEditorTab,
  emptySQLEditorConnection,
  suggestedTabTitleForSQLEditorConnection,
  extractProjectResourceName,
  extractInstanceResourceName,
} from "@/utils";
import { useSQLEditorContext } from "../context";
import DatabaseHoverPanel from "./DatabaseHoverPanel.vue";
import GroupingBar from "./GroupingBar";
import SearchBox from "./SearchBox/index.vue";
import useSearchHistory from "./SearchBox/useSearchHistory";
import { Label } from "./TreeNode";
import { fetchDatabaseSubTree } from "./common";
import { provideHoverStateContext } from "./hover-state";

type Position = {
  x: number;
  y: number;
};

type DropdownOptionWithTreeNode = DropdownOption & {
  onSelect: () => void;
};

const { t } = useI18n();
const router = useRouter();
const { pageMode } = storeToRefs(useActuatorV1Store());
const treeStore = useSQLEditorTreeStore();
const tabStore = useSQLEditorTabStore();
const databaseStore = useDatabaseV1Store();
const instanceStore = useInstanceV1Store();
const dbSchemaV1Store = useDBSchemaV1Store();
const isLoggedIn = useIsLoggedIn();
const me = useCurrentUserV1();
const { executeReadonly } = useExecuteSQL();
const { selectedDatabaseSchemaByDatabaseName, events: editorEvents } =
  useSQLEditorContext();
const searchHistory = useSearchHistory();
const { node: hoverNode, update: updateHoverNode } = provideHoverStateContext();
const hoverPanelPosition = ref<Position>({
  x: 0,
  y: 0,
});

const mounted = useMounted();
const treeContainerElRef = ref<HTMLElement>();
const { height: treeContainerHeight } = useElementSize(
  treeContainerElRef,
  undefined,
  {
    box: "content-box",
  }
);
const treeRef = ref<InstanceType<typeof NTree>>();
const searchPattern = ref("");
const showDropdown = ref(false);
const dropdownPosition = ref<Position>({
  x: 0,
  y: 0,
});
const dropdownContext = ref<SQLEditorTreeNode>();

const dropdownOptions = computed((): DropdownOptionWithTreeNode[] => {
  const node = dropdownContext.value;
  if (!node) {
    return [];
  }
  const { type, target } = node.meta;
  if (type === "project") {
    return [];
  } else {
    // Don't show any context menu actions for disabled
    // instances/databases
    if (node.disabled) {
      return [];
    }

    const items: DropdownOptionWithTreeNode[] = [];

    if (isConnectableSQLEditorTreeNode(node)) {
      const instance = instanceOfSQLEditorTreeNode(node);
      if (instance && instanceV1HasReadonlyMode(instance)) {
        items.push({
          key: "connect",
          label: t("sql-editor.connect"),
          onSelect: () => setConnection(node),
        });
      }
      if (allowAdmin.value) {
        items.push({
          key: "connect-in-admin-mode",
          label: t("sql-editor.connect-in-admin-mode"),
          onSelect: () => setConnection(node, { sheet: "", mode: "ADMIN" }),
        });
      }
    }
    if (pageMode.value === "BUNDLED") {
      if (type === "database") {
        const database = target as ComposedDatabase;
        if (instanceV1HasAlterSchema(database.instanceEntity)) {
          items.push({
            key: "alter-schema",
            label: t("database.edit-schema"),
            onSelect: () => {
              const db = node.meta.target as ComposedDatabase;
              editorEvents.emit("alter-schema", {
                databaseUID: db.uid,
                schema: "",
                table: "",
              });
            },
          });
        }
      } else if (type === "table") {
        const { database, schema, table } = target as RichTableMetadata;
        items.push({
          key: "copy-url",
          label: t("sql-editor.copy-url"),
          onSelect: () => {
            const { copy, copied } = useClipboard({
              source: computed(() => {
                const route = router.resolve({
                  name: SQL_EDITOR_DATABASE_MODULE,
                  params: {
                    project: extractProjectResourceName(database.project),
                    instance: extractInstanceResourceName(database.instance),
                    database: database.databaseName,
                  },
                  query: {
                    table: table.name,
                    schema: schema.name,
                  },
                });
                return new URL(route.href, window.location.origin).href;
              }),
              legacy: true,
            });
            copy().then(() => {
              if (copied.value) {
                pushNotification({
                  module: "bytebase",
                  style: "INFO",
                  title: t("common.copied"),
                });
              }
            });
          },
        });
      }
    }
    return items;
  }
});

// Highlight the current tab's connection node.
const selectedKeys = computed(() => {
  const connection = tabStore.currentTab?.connection;
  if (!connection) {
    return [];
  }

  if (connection.database) {
    const database = databaseStore.getDatabaseByName(connection.database);
    const node = head(treeStore.nodesByTarget("database", database));
    if (!node) return [];
    const selected = selectedDatabaseSchemaByDatabaseName.value.get(
      database.name
    );
    if (selected) {
      const { schema, table, externalTable } = selected;
      if (table) {
        const tableNode = head(
          treeStore.nodesByTarget("table", {
            database,
            schema,
            table,
          })
        );
        if (tableNode) {
          return [tableNode.key];
        }
      } else if (externalTable) {
        const externalTableNode = head(
          treeStore.nodesByTarget("external-table", {
            database,
            schema,
            externalTable,
          })
        );
        if (externalTableNode) {
          return [externalTableNode.key];
        }
      }
    }
    return [node.key];
  } else if (connection.instance) {
    const instance = instanceStore.getInstanceByName(connection.instance);
    const nodes = treeStore.nodesByTarget("instance", instance);
    return nodes.map((node) => node.key);
  }
  return [];
});
const expandedKeys = ref<string[]>([]);
const upsertExpandedKeys = (key: string) => {
  if (expandedKeys.value.includes(key)) {
    return;
  }
  expandedKeys.value.push(key);
};
const expandNode = (
  node: SQLEditorTreeNode | undefined,
  keys?: Set<string>
) => {
  if (!node) {
    return;
  }
  if (ExpandableTreeNodeTypes.includes(node.meta.type)) {
    if (keys) {
      keys.add(node.key);
    } else {
      upsertExpandedKeys(node.key);
    }
  }
};
const expandNodeRecursively = (
  node: SQLEditorTreeNode | undefined,
  keys?: Set<string>
) => {
  if (!node) {
    return;
  }
  expandNode(node, keys);

  if (node.parent) {
    expandNodeRecursively(node.parent, keys);
  }
};
const expandNodesByType = <T extends SQLEditorTreeNodeType>(
  type: T,
  target: SQLEditorTreeNodeTarget<T>
) => {
  const nodes = treeStore.nodesByTarget(type, target);

  nodes.forEach((node) => {
    expandNodeRecursively(node);
  });

  return nodes;
};
const allowAdmin = computed(() =>
  hasWorkspacePermissionV2(me.value, "bb.instances.adminExecute")
);

const setConnection = (
  node: SQLEditorTreeNode,
  extra: { sheet: string; mode: SQLEditorTabMode } = {
    sheet: "",
    mode: DEFAULT_SQL_EDITOR_TAB_MODE,
  }
) => {
  if (node) {
    const { type } = node.meta;
    if (!ConnectableTreeNodeTypes.includes(type)) {
      return;
    }
    if (type === "instance") {
      const instance = node.meta.target as ComposedInstance;
      if (!instanceV1AllowsCrossDatabaseQuery(instance)) {
        return;
      }
    }
    const coreTab: CoreSQLEditorTab = {
      connection: emptySQLEditorConnection(),
      ...extra,
    };
    const conn = coreTab.connection;
    // If selected item is instance node
    if (type === "instance") {
      const instance = node.meta.target as ComposedInstance;
      conn.instance = instance.name;
    }
    // If selected item is database node
    if (type === "database") {
      const database = node.meta.target as ComposedDatabase;
      conn.instance = database.instance;
      conn.database = database.name;
    }
    tryConnectToCoreSQLEditorTab(coreTab);
  }
};

// dynamic render the highlight keywords
const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as any as SQLEditorTreeNode;
  return h(Label, {
    node,
    factors: treeStore.filteredFactorList,
    keyword: searchPattern.value ?? "",
  });
};

const handleSelect = (key: string) => {
  const option = dropdownOptions.value.find((item) => item.key === key);
  if (!option) {
    return;
  }
  option.onSelect();
  showDropdown.value = false;
  console.log("handleSelect", key);
};

const handleClickoutside = () => {
  showDropdown.value = false;
  console.log("handleClickoutside");
};

const maybeSelectTable = async (node: SQLEditorTreeNode) => {
  const target = node.meta.target as SQLEditorTreeNodeTarget<"table">;
  const { database, schema, table } = target;
  if (
    !tabStore.currentTab ||
    tabStore.currentTab.connection.database !== database.name
  ) {
    const coreTab: CoreSQLEditorTab = {
      connection: {
        instance: database.instance,
        database: database.name,
        schema: schema.name,
        table: table.name,
      },
      mode: DEFAULT_SQL_EDITOR_TAB_MODE,
      sheet: "",
    };
    tryConnectToCoreSQLEditorTab(coreTab);
    await nextTick();
  }

  const tableMetadata = await dbSchemaV1Store.getOrFetchTableMetadata({
    database: database.name,
    schema: schema.name,
    table: table.name,
  });
  const databaseMetadata = dbSchemaV1Store.getDatabaseMetadata(database.name);

  selectedDatabaseSchemaByDatabaseName.value.set(database.name, {
    db: database,
    database: databaseMetadata,
    schema,
    table: tableMetadata,
  });

  if (tabStore.currentTab) {
    tabStore.updateCurrentTab({
      connection: {
        ...tabStore.currentTab.connection,
        schema: schema.name,
        table: table.name,
      },
    });
  }
};

const maybeSelectExternalTable = async (node: SQLEditorTreeNode) => {
  const target = node.meta.target as SQLEditorTreeNodeTarget<"external-table">;
  const { database, schema, externalTable } = target;

  if (
    !tabStore.currentTab ||
    tabStore.currentTab.connection.database !== database.name
  ) {
    const coreTab: CoreSQLEditorTab = {
      connection: {
        instance: database.instance,
        database: database.name,
      },
      mode: DEFAULT_SQL_EDITOR_TAB_MODE,
      sheet: "",
    };
    tryConnectToCoreSQLEditorTab(coreTab);
    await nextTick();
  }

  const databaseMetadata = dbSchemaV1Store.getDatabaseMetadata(database.name);
  selectedDatabaseSchemaByDatabaseName.value.set(database.name, {
    db: database,
    database: databaseMetadata,
    schema,
    externalTable,
  });
};

const selectAllFromTableOrView = async (node: SQLEditorTreeNode) => {
  const { type, target } = (node as SQLEditorTreeNode<"table" | "view">).meta;
  const { database } = target;
  const { engine } = database.instanceEntity;
  const LIMIT = 50; // default pagesize of SQL Editor

  const language = languageOfEngineV1(engine);
  if (language === "redis") {
    return; // not supported
  }
  const schema = target.schema.name;
  const tableOrViewName =
    type === "table"
      ? (target as SQLEditorTreeNodeTarget<"table">).table.name
      : type === "view"
        ? (target as SQLEditorTreeNodeTarget<"view">).view.name
        : "";
  if (!tableOrViewName) {
    return;
  }

  const runQuery = async (statement: string) => {
    const tab: CoreSQLEditorTab = {
      connection: {
        instance: database.instance,
        database: database.name,
        schema,
        table: tableOrViewName,
      },
      mode: DEFAULT_SQL_EDITOR_TAB_MODE,
      sheet: "",
    };
    if (
      tabStore.currentTab &&
      (tabStore.currentTab.status === "NEW" || !tabStore.currentTab.sheet)
    ) {
      // If the current tab is "fresh new" or unsaved, update its connection directly.
      tabStore.updateCurrentTab({
        ...tab,
        title: suggestedTabTitleForSQLEditorConnection(tab.connection),
        status: "DIRTY",
        statement,
      });
    } else {
      // Otherwise select or add a new tab and set its connection
      tabStore.addTab(
        {
          ...tab,
          title: suggestedTabTitleForSQLEditorConnection(tab.connection),
          statement,
          status: "DIRTY",
        },
        /* beside */ true
      );
    }
    await nextTick();
    executeReadonly({
      statement,
      connection: { ...tab.connection },
      explain: false,
      engine: database.instanceEntity.engine,
    });
  };

  if (language === "javascript" && type === "table") {
    // mongodb
    const query = `db["${tableOrViewName}"].find().limit(${LIMIT});`;
    runQuery(query);
    return;
  }

  if (type === "table") {
    maybeSelectTable(node);
  }

  const query = generateSimpleSelectAllStatement(
    engine,
    schema,
    tableOrViewName,
    LIMIT
  );
  runQuery(query);
};

const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as any as SQLEditorTreeNode;
  return {
    onClick(e: MouseEvent) {
      if (node.disabled) return;

      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        const { type } = node.meta;
        // Check if clicked on the content part.
        // And ignore the fold/unfold arrow.
        if (type === "instance" || type === "database") {
          setConnection(node);
        }
        if (type === "table" || type === "partition-table") {
          maybeSelectTable(node);
        }
        if (type === "external-table") {
          maybeSelectExternalTable(node);
        }

        // If the search pattern is not empty, append the selected database name to
        // the search history.
        if (searchPattern.value) {
          if (type === "database") {
            const database = node.meta.target as ComposedDatabase;
            searchHistory.appendSearchResult(database.name);
          }
        }
      }
    },
    onContextmenu(e: MouseEvent) {
      e.preventDefault();
      showDropdown.value = false;
      console.log("onContextmenu", false);
      if (node && node.key) {
        dropdownContext.value = node;
      }

      nextTick().then(() => {
        showDropdown.value = true;
        console.log("onContextmenu", true);
        dropdownPosition.value.x = e.clientX;
        dropdownPosition.value.y = e.clientY;
      });
    },
    onmouseenter(e: MouseEvent) {
      if (node.meta.type === "database") {
        if (hoverNode.value) {
          updateHoverNode(node, "before", 0 /* overrideDelay */);
        } else {
          updateHoverNode(node, "before");
        }
        nextTick().then(() => {
          // Find the node element and put the database panel to the right corner
          // of the node
          const wrapper = findAncestor(e.target as HTMLElement, ".n-tree-node");
          if (!wrapper) {
            updateHoverNode(undefined, "after", 0 /* overrideDelay */);
            return;
          }
          const bounding = wrapper.getBoundingClientRect();
          hoverPanelPosition.value.x = bounding.right;
          hoverPanelPosition.value.y = bounding.top;
        });
      }
    },
    onmouseleave() {
      updateHoverNode(undefined, "after");
    },
    ondblclick() {
      if (node.meta.type === "table" || node.meta.type === "view") {
        selectAllFromTableOrView(node);
      }
    },
    // attrs below for trouble-shooting
    "data-node-meta-type": node.meta.type,
    "data-node-meta-id": idForSQLEditorTreeNodeTarget(
      node.meta.type,
      node.meta.target
    ),
    "data-node-key": node.key,
  };
};

const handleLoadSubTree = (option: TreeOption) => {
  const node = option as any as SQLEditorTreeNode;
  const { type } = node.meta;
  if (type === "database") {
    const request = fetchDatabaseSubTree(node);
    request
      .then(() => nextTick())
      .then(() => {
        if ((node.children?.length ?? 0) > 0) {
          expandNode(node);
          if (expandSchemaAndTableNodes(node)) {
            // if a specified schema node is expanded we stop here
            return;
          }
          // we expand the first schema node otherwise
          expandFirstSchemaNode(node);
        }
      });
    return request;
  }
  return Promise.resolve();
};

/**
 * returns true if successfully expanded, false if not expanded
 */
const expandSchemaAndTableNodes = (
  databaseNode: SQLEditorTreeNode<"database">
): boolean => {
  const tab = tabStore.currentTab;
  if (!tab) {
    return false;
  }
  const { database, schema, table } = tab.connection;
  const db = databaseNode.meta.target;
  if (db.name !== database) {
    return false;
  }
  const schemaMetadata = dbSchemaV1Store.getSchemaByName(db.name, schema ?? "");
  if (!schemaMetadata) {
    return false;
  }

  const schemaNode = databaseNode.children?.find(
    (
      node: SQLEditorTreeNode
    ): node is
      | SQLEditorTreeNode<"schema">
      | SQLEditorTreeNode<"expandable-text"> => {
      if (node.meta.type === "schema") {
        return (
          (node.meta.target as SQLEditorTreeNodeTarget<"schema">).schema
            .name === schemaMetadata.name
        );
      }
      if (node.meta.type === "expandable-text") {
        return schemaMetadata.name === "";
      }
      return false;
    }
  );

  if (!schemaNode) {
    return false;
  }
  expandNode(schemaNode);
  if (schemaNode.meta.type === "schema") {
    // a schema node contains a "Tables" `expandable-text` node
    // so we should also expand the first child
    expandNode(head(schemaNode.children));
  }

  if (!table) {
    return true;
  }
  const tableMetadata = dbSchemaV1Store.getTableByName(db.name, table, schema);
  if (!tableMetadata) {
    return true;
  }

  const databaseMetadata = dbSchemaV1Store.getDatabaseMetadata(db.name);
  selectedDatabaseSchemaByDatabaseName.value.set(db.name, {
    db: db,
    database: databaseMetadata,
    schema: schemaMetadata,
    table: tableMetadata,
  });
  return true;
};

/**
 * returns true if successfully expanded, false if not expanded
 */
const expandFirstSchemaNode = (
  databaseNode: SQLEditorTreeNode<"database">,
  keys?: Set<string>
) => {
  const firstChild = head(databaseNode.children) as SQLEditorTreeNode;
  expandNode(firstChild);
  if (firstChild && firstChild.meta.type === "schema") {
    // a schema node contains a "Tables" `expandable-text` node
    // so we should also expand the first child
    expandNode(head(firstChild.children));
  }
};

// Open corresponding tree node when the connection changed.
const { instance, database } = useConnectionOfCurrentSQLEditorTab();
watch(
  [isLoggedIn, instance, database, () => treeStore.state],
  ([isLoggedIn, instance, database, treeState]) => {
    if (!isLoggedIn) {
      // Don't go further and cleanup the state if we signed out.
      // treeStore.expandedKeys = [];
      expandedKeys.value = [];
      return;
    }

    if (treeState !== "READY") {
      return;
    }
    if (instance.uid !== String(UNKNOWN_ID)) {
      expandNodesByType("instance", instance);
    }
    if (database.uid !== String(UNKNOWN_ID)) {
      expandNodesByType("project", database.projectEntity);
      expandNodesByType("database", database);
    }
  },
  { immediate: true }
);

watch(
  selectedKeys,
  (keys) => {
    if (keys.length !== 1) return;
    const key = keys[0];
    nextTick(() => {
      treeRef.value?.scrollTo({ key });
    });
  },
  { immediate: true }
);

useEmitteryEventListener(editorEvents, "tree-ready", async () => {
  await nextTick();
  const openingDatabaseList = resolveOpeningDatabaseListFromSQLEditorTabList();
  const keys = new Set<string>();
  // Recursively expand opening databases' parent nodes
  openingDatabaseList.forEach((meta) => {
    const db = meta.target;
    const nodes = treeStore.nodesByTarget("database", db);
    nodes.forEach((node) => expandNodeRecursively(node.parent, keys));
  });
  const tab = tabStore.currentTab;
  // Expand current tab's connected database node
  if (tab && tab.connection.database) {
    const db = useDatabaseV1Store().getDatabaseByName(tab.connection.database);
    const node = head(treeStore.nodesByTarget("database", db));
    if (node) {
      keys.add(node.key);
    }
  }
  expandedKeys.value = Array.from(keys);
});
</script>

<style lang="postcss" scoped>
.sql-editor-tree :deep(.n-tree .v-vl) {
  --n-node-content-height: 21px !important;
}
.sql-editor-tree :deep(.n-tree-node-content) {
  @apply !pl-0 text-sm;
}
.sql-editor-tree :deep(.n-tree-node-wrapper) {
  padding: 0;
}
.sql-editor-tree :deep(.n-tree-node-indent) {
  width: 0.25rem;
}
.sql-editor-tree :deep(.n-tree-node-content__prefix) {
  @apply shrink-0 !mr-1;
}
.sql-editor-tree.project
  :deep(.n-tree-node[data-node-type="project"] .n-tree-node-content__prefix) {
  @apply hidden;
}
.sql-editor-tree :deep(.n-tree-node-content__text) {
  @apply truncate mr-1;
}
.sql-editor-tree :deep(.n-tree-node--pending) {
  background-color: transparent !important;
}
.sql-editor-tree :deep(.n-tree-node--pending:hover) {
  background-color: var(--n-node-color-hover) !important;
}
.sql-editor-tree :deep(.n-tree-node--selected),
.sql-editor-tree :deep(.n-tree-node--selected:hover) {
  background-color: var(--n-node-color-active) !important;
  font-weight: 500;
}
</style>
