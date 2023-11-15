<template>
  <div
    v-if="treeStore.state === 'READY'"
    class="sql-editor-tree p-0.5 gap-y-1 h-full flex flex-col"
  >
    <div class="sql-editor-tree--input pt-1 px-2">
      <NInput
        size="small"
        :value="searchPattern"
        :placeholder="$t('sql-editor.search-databases')"
        :clearable="true"
        @update:value="$emit('update:search-pattern', $event)"
      >
        <template #prefix>
          <heroicons-outline:search class="h-5 w-5 text-gray-300" />
        </template>
      </NInput>
    </div>
    <div class="sql-editor-tree--tree flex-1 overflow-y-auto select-none">
      <NTree
        ref="treeRef"
        v-model:expanded-keys="treeStore.expandedKeys"
        :block-line="true"
        :data="treeStore.tree"
        :show-irrelevant-nodes="false"
        :selected-keys="selectedKeys"
        :pattern="mounted ? throttledSearchPattern : ''"
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
      :database="(hoverNode?.meta.target as ComposedDatabase|undefined)"
      :x="hoverPanelPosition.x"
      :y="hoverPanelPosition.y"
      class="ml-3"
    />
  </div>
  <div v-else class="flex justify-center items-center h-full">
    <BBSpin :title="$t('sql-editor.loading-databases')" />
  </div>
</template>

<script lang="ts" setup>
import { useMounted, useThrottleFn } from "@vueuse/core";
import { head } from "lodash-es";
import { NTree, NInput, NDropdown, DropdownOption, TreeOption } from "naive-ui";
import { storeToRefs } from "pinia";
import { ref, computed, nextTick, watch, h } from "vue";
import { useI18n } from "vue-i18n";
import {
  useActuatorV1Store,
  useCurrentUserV1,
  useDBSchemaV1Store,
  useDatabaseV1Store,
  useInstanceV1Store,
  useIsLoggedIn,
  useTabStore,
} from "@/store";
import { useSQLEditorTreeStore } from "@/store/modules/sqlEditorTree";
import type {
  ComposedDatabase,
  ComposedInstance,
  CoreTabInfo,
  SQLEditorTreeNode,
  SQLEditorTreeNodeTarget,
} from "@/types";
import {
  ConnectableTreeNodeTypes,
  TabMode,
  UNKNOWN_ID,
  instanceOfSQLEditorTreeNode,
  isConnectableSQLEditorTreeNode,
} from "@/types";
import {
  emptyConnection,
  findAncestor,
  getSuggestedTabNameFromConnection,
  hasWorkspacePermissionV1,
  instanceV1AllowsCrossDatabaseQuery,
  instanceV1HasAlterSchema,
  instanceV1HasReadonlyMode,
  isDescendantOf,
  isSimilarTab,
} from "@/utils";
import { useSQLEditorContext } from "../context";
import DatabaseHoverPanel from "./DatabaseHoverPanel.vue";
import { Label } from "./TreeNode";
import { fetchDatabaseSubTree } from "./common";
import { provideHoverStateContext } from "./hover-state";

type Position = {
  x: number;
  y: number;
};

type DropdownOptionWithTreeNode = DropdownOption & {
  node: SQLEditorTreeNode;
};

const props = defineProps<{
  searchPattern?: string;
}>();

defineEmits<{
  (event: "update:search-pattern", keyword: string): void;
}>();

const { t } = useI18n();
const { pageMode } = storeToRefs(useActuatorV1Store());
const treeStore = useSQLEditorTreeStore();
const tabStore = useTabStore();
const databaseStore = useDatabaseV1Store();
const instanceStore = useInstanceV1Store();
const dbSchemaV1Store = useDBSchemaV1Store();
const isLoggedIn = useIsLoggedIn();
const me = useCurrentUserV1();
const { selectedDatabaseSchemaByDatabaseName, events: editorEvents } =
  useSQLEditorContext();
const { node: hoverNode, update: updateHoverNode } = provideHoverStateContext();
const hoverPanelPosition = ref<Position>({
  x: 0,
  y: 0,
});

const mounted = useMounted();
const treeRef = ref<InstanceType<typeof NTree>>();
const throttledSearchPattern = ref(props.searchPattern);
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
          node,
        });
      }
      if (allowAdmin.value) {
        items.push({
          key: "connect-in-admin-mode",
          label: t("sql-editor.connect-in-admin-mode"),
          node,
        });
      }
    }
    if (type === "database" && pageMode.value === "BUNDLED") {
      const database = target as ComposedDatabase;
      if (instanceV1HasAlterSchema(database.instanceEntity)) {
        items.push({
          key: "alter-schema",
          label: t("database.edit-schema"),
          node,
        });
      }
    }
    return items;
  }
});

// Highlight the current tab's connection node.
const selectedKeys = computed(() => {
  const { instanceId, databaseId } = tabStore.currentTab.connection;

  if (databaseId !== String(UNKNOWN_ID)) {
    const database = databaseStore.getDatabaseByUID(databaseId);
    const node = head(treeStore.nodesByTarget("database", database));
    if (!node) return [];
    const selected = selectedDatabaseSchemaByDatabaseName.value.get(
      database.name
    );
    if (selected) {
      const { schema, table } = selected;
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
    }
    return [node.key];
  } else if (instanceId !== String(UNKNOWN_ID)) {
    const instance = instanceStore.getInstanceByUID(instanceId);
    const nodes = treeStore.nodesByTarget("instance", instance);
    return nodes.map((node) => node.key);
  }
  return [];
});

const allowAdmin = computed(() =>
  hasWorkspacePermissionV1(
    "bb.permission.workspace.admin-sql-editor",
    me.value.userRole
  )
);

const connect = (target: CoreTabInfo) => {
  if (isSimilarTab(target, tabStore.currentTab)) {
    // Don't go further if the connection doesn't change.
    return;
  }
  if (tabStore.currentTab.isFreshNew) {
    // If the current tab is "fresh new", update its connection directly.
    tabStore.updateCurrentTab(target);
  } else {
    // Otherwise select or add a new tab and set its connection
    const name = getSuggestedTabNameFromConnection(target.connection);
    tabStore.selectOrAddSimilarTab(
      target,
      /* beside */ false,
      /* defaultTabName */ name
    );
    tabStore.updateCurrentTab(target);
  }
};

const setConnection = (
  node: SQLEditorTreeNode,
  extra: { sheetName?: string; mode: TabMode } = {
    sheetName: undefined,
    mode: TabMode.ReadOnly,
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
    const target: CoreTabInfo = {
      connection: emptyConnection(),
      ...extra,
    };
    const conn = target.connection;
    // If selected item is instance node
    if (type === "instance") {
      const instance = node.meta.target as ComposedInstance;
      conn.instanceId = instance.uid;
    }
    // If selected item is database node
    if (type === "database") {
      const database = node.meta.target as ComposedDatabase;
      conn.instanceId = database.instanceEntity.uid;
      conn.databaseId = database.uid;
    }
    connect(target);
  }
};

// dynamic render the highlight keywords
const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as any as SQLEditorTreeNode;
  return h(Label, {
    node,
    factors: treeStore.filteredFactorList,
    keyword: props.searchPattern ?? "",
  });
};

const handleSelect = (key: string) => {
  const option = dropdownOptions.value.find((item) => item.key === key);
  if (!option) {
    return;
  }
  if (key === "alter-schema") {
    const db = option.node.meta.target as ComposedDatabase;
    editorEvents.emit("alter-schema", {
      databaseUID: db.uid,
      schema: "",
      table: "",
    });
  } else if (key === "connect") {
    setConnection(option.node);
  } else if (key === "connect-in-admin-mode") {
    setConnection(option.node, { mode: TabMode.Admin });
  }
  showDropdown.value = false;
};

const handleClickoutside = () => {
  showDropdown.value = false;
};

const maybeSelectTable = async (node: SQLEditorTreeNode) => {
  const target = node.meta.target as SQLEditorTreeNodeTarget<"table">;
  const { database, schema, table } = target;
  if (database.uid !== tabStore.currentTab.connection.databaseId) {
    const target: CoreTabInfo = {
      connection: {
        instanceId: database.instanceEntity.uid,
        databaseId: database.uid,
      },
      mode: TabMode.ReadOnly,
    };
    target.connection.instanceId = database.instanceEntity.uid;
    target.connection.databaseId = database.uid;
    connect(target);
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
        if (type === "table") {
          maybeSelectTable(node);
        }
      }
    },
    onContextmenu(e: MouseEvent) {
      e.preventDefault();
      showDropdown.value = false;
      if (node && node.key) {
        dropdownContext.value = node;
      }

      nextTick().then(() => {
        showDropdown.value = true;
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
    "data-node-type": node.type,
  };
};

const handleLoadSubTree = (option: TreeOption) => {
  const node = option as any as SQLEditorTreeNode;
  const { type, target } = node.meta;
  if (type === "database") {
    const request = fetchDatabaseSubTree(node);
    request
      .then(() => nextTick())
      .then(() => treeStore.expandNodes(type, target));
    return request;
  }
  return Promise.resolve();
};

// Open corresponding tree node when the connection changed.
watch(
  [
    isLoggedIn,
    () => tabStore.currentTab.connection.instanceId,
    () => tabStore.currentTab.connection.databaseId,
    () => treeStore.state,
  ],
  ([isLoggedIn, instanceId, databaseId, treeState]) => {
    if (!isLoggedIn) {
      // Don't go further and cleanup the state if we signed out.
      treeStore.expandedKeys = [];
      return;
    }
    if (treeState !== "READY") {
      return;
    }

    if (instanceId !== String(UNKNOWN_ID)) {
      const instance = instanceStore.getInstanceByUID(instanceId);
      treeStore.expandNodes("instance", instance);
    }
    if (databaseId !== String(UNKNOWN_ID)) {
      const db = databaseStore.getDatabaseByUID(databaseId);
      treeStore.expandNodes("database", db);
    }
  },
  { immediate: true }
);

watch(
  () => props.searchPattern,
  useThrottleFn(
    (searchPattern: string | undefined) => {
      throttledSearchPattern.value = searchPattern;
    },
    100,
    true /* trailing */,
    true /* leading */
  ),
  {
    immediate: true,
  }
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
