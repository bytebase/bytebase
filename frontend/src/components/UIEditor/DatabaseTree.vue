<template>
  <div class="w-full h-full pr-2 relative overflow-y-hidden">
    <div class="w-full sticky top-0 pt-2 h-12 bg-white z-10">
      <NInput
        v-model:value="searchPattern"
        :placeholder="$t('sql-editor.search-databases')"
      >
        <template #prefix>
          <heroicons-outline:search class="w-4 h-auto text-gray-300" />
        </template>
      </NInput>
    </div>
    <div
      ref="treeRef"
      class="ui-editor-database-tree pb-2 h-auto"
      v-bind="$attrs"
    >
      <n-tree
        block-line
        :data="treeDataRef"
        :pattern="searchPattern"
        :render-prefix="renderPrefix"
        :render-label="renderLabel"
        :render-suffix="renderSuffix"
        :node-props="nodeProps"
        :on-load="loadSubTree"
        :expanded-keys="expandedKeysRef"
        :selected-keys="selectedKeysRef"
        :on-update:expanded-keys="handleExpandedKeysChange"
        :on-update:selected-keys="handleSelectedKeysChange"
      />
      <n-dropdown
        trigger="manual"
        placement="bottom-start"
        :show="contextMenu.showDropdown"
        :options="contextMenuOptions"
        :x="contextMenu.clientX"
        :y="contextMenu.clientY"
        @select="handleContextMenuDropdownSelect"
        @clickoutside="handleDropdownClickoutside"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, escape, isEqual, isUndefined } from "lodash-es";
import { TreeOption, NEllipsis, NInput } from "naive-ui";
import { computed, onMounted, watch, ref, h, reactive, nextTick } from "vue";
import {
  DatabaseId,
  InstanceId,
  Table,
  TableId,
  UIEditorTabType,
  UNKNOWN_ID,
} from "@/types";
import {
  useUIEditorStore,
  generateUniqueTabId,
  useInstanceStore,
  useTableStore,
} from "@/store";
import { getHighlightHTMLByKeyWords, isDescendantOf } from "@/utils";
import InstanceEngineIcon from "@/components/InstanceEngineIcon.vue";
import DatabaseIcon from "~icons/heroicons-outline/database";
import TableIcon from "~icons/heroicons-outline/table";
import EllipsisIcon from "~icons/heroicons-solid/ellipsis-horizontal";

interface BaseTreeNode extends TreeOption {
  key: string;
  label: string;
  isLeaf: boolean;
  children?: TreeNode[];
}

interface TreeNodeForInstance extends BaseTreeNode {
  type: "instance";
  instanceId: InstanceId;
}

interface TreeNodeForDatabase extends BaseTreeNode {
  type: "database";
  instanceId: InstanceId;
  databaseId: DatabaseId;
}

interface TreeNodeForTable extends BaseTreeNode {
  type: "table";
  instanceId: InstanceId;
  databaseId: DatabaseId;
  tableId: TableId;
  table: Table;
}

type TreeNode = TreeNodeForInstance | TreeNodeForDatabase | TreeNodeForTable;

interface TreeContextMenu {
  showDropdown: boolean;
  clientX: number;
  clientY: number;
  treeNode?: TreeNode;
}

interface LocalState {
  shouldRelocateTreeNode: boolean;
}

const editorStore = useUIEditorStore();
const instanceStore = useInstanceStore();
const tableStore = useTableStore();
const state = reactive<LocalState>({
  shouldRelocateTreeNode: false,
});
const contextMenu = reactive<TreeContextMenu>({
  showDropdown: false,
  clientX: 0,
  clientY: 0,
  treeNode: undefined,
});
const treeRef = ref();
const searchPattern = ref("");
const expandedKeysRef = ref<string[]>([]);
const selectedKeysRef = ref<string[]>([]);
const treeDataRef = ref<TreeNode[]>([]);

const databaseList = computed(() => editorStore.databaseList);
const tableList = computed(() => editorStore.tableList);
const contextMenuOptions = computed(() => {
  const treeNode = contextMenu.treeNode;
  if (isUndefined(treeNode)) {
    return [];
  }

  if (treeNode?.type === "database") {
    const options = [];
    options.push({
      key: "new-table",
      label: "New Table",
    });
    return options;
  } else if (treeNode?.type === "table") {
    const isDropped = editorStore.droppedTableList.includes(treeNode.table);
    const options = [];
    if (isDropped) {
      options.push({
        key: "restore",
        label: "Restore",
      });
    } else {
      options.push({
        key: "drop",
        label: "Drop",
      });
    }
    return options;
  }

  return [];
});

onMounted(() => {
  const treeNodeList: TreeNode[] = [];
  const instanceTreeNodeMap: Map<InstanceId, TreeNodeForInstance> = new Map();
  for (const database of databaseList.value) {
    const instance = instanceStore.getInstanceById(database.instanceId);
    let instanceTreeNode: TreeNodeForInstance;
    if (instanceTreeNodeMap.has(instance.id)) {
      instanceTreeNode = instanceTreeNodeMap.get(
        instance.id
      ) as TreeNodeForInstance;
    } else {
      instanceTreeNode = {
        type: "instance",
        key: `i-${instance.id}`,
        label: instance.name,
        isLeaf: false,
        instanceId: database.instance.id,
        children: [],
      };
      instanceTreeNodeMap.set(instance.id, instanceTreeNode);
      treeNodeList.push(instanceTreeNode);
      // Make the instance tree node expanded as default.
      expandedKeysRef.value.push(instanceTreeNode.key);
    }

    const databaseTreeNode: TreeNodeForDatabase = {
      type: "database",
      key: `d-${database.id}`,
      label: database.name,
      isLeaf: false,
      instanceId: database.instance.id,
      databaseId: database.id,
    };
    instanceTreeNode.children?.push(databaseTreeNode);
  }
  treeDataRef.value = treeNodeList;
});

watch(tableList.value, () => {
  const databaseTreeNodeList: TreeNodeForDatabase[] = [];
  for (const treeNode of treeDataRef.value) {
    if (treeNode.type === "instance") {
      databaseTreeNodeList.push(
        ...(treeNode.children as TreeNodeForDatabase[])
      );
    }
  }

  for (const database of databaseList.value) {
    const databaseTreeNode = databaseTreeNodeList.find(
      (treeNode) => treeNode.databaseId === database.id
    );
    if (isUndefined(databaseTreeNode)) {
      continue;
    }

    const databaseTableList = [];
    for (const table of tableList.value) {
      if (table.database.id === database.id) {
        databaseTableList.push(table);
      }
    }
    if (databaseTableList.length > 0) {
      databaseTreeNode.children = databaseTableList.map((table) => {
        return {
          type: "table",
          key: `t-${table.id}-${table.name}`,
          label: table.name,
          children: [],
          isLeaf: true,
          instanceId: database.instance.id,
          databaseId: database.id,
          tableId: table.id,
          table: table,
        };
      });
    }
  }
});

watch(
  () => editorStore.currentTab,
  () => {
    const currentTab = editorStore.currentTab;
    if (!currentTab) {
      selectedKeysRef.value = [];
      return;
    }

    if (currentTab.type === UIEditorTabType.TabForDatabase) {
      const key = `d-${currentTab.databaseId}`;
      selectedKeysRef.value = [key];
    } else if (currentTab.type === UIEditorTabType.TabForTable) {
      const databaseTreeNodeKey = `d-${currentTab.databaseId}`;
      if (!expandedKeysRef.value.includes(databaseTreeNodeKey)) {
        expandedKeysRef.value.push(databaseTreeNodeKey);
      }
      const tableTreeNodeKey = `t-${currentTab.tableId}-${currentTab.tableCache.name}`;
      selectedKeysRef.value = [tableTreeNodeKey];
    }

    if (state.shouldRelocateTreeNode) {
      nextTick(() => {
        const element = treeRef.value?.querySelector(".n-tree-node--selected");
        element?.scrollIntoView();
      });
    }
  }
);

// Render prefix icons before label text.
const renderPrefix = ({ option: treeNode }: { option: TreeNode }) => {
  if (treeNode.type === "instance") {
    const instance = instanceStore.getInstanceById(treeNode.instanceId);
    const children = [
      h(InstanceEngineIcon, {
        instance,
      }),
      h(
        "span",
        {
          class: "text-gray-500 text-sm",
        },
        `(${instance.environment.name})`
      ),
    ];

    return h("span", { class: "flex items-center gap-x-1" }, children);
  } else if (treeNode.type === "database") {
    return h(DatabaseIcon, {
      class: "w-4 h-auto text-gray-400",
    });
  } else if (treeNode.type === "table") {
    return h(TableIcon, {
      class: "w-4 h-auto text-gray-400",
    });
  }
  return null;
};

// Render label text.
const renderLabel = ({ option: treeNode }: { option: TreeNode }) => {
  const additionalClassList: string[] = ["select-none"];

  if (treeNode.type === "table") {
    if (treeNode.tableId === UNKNOWN_ID) {
      additionalClassList.push("text-green-700");
    } else {
      const originTable = tableStore.getTableByDatabaseIdAndTableId(
        treeNode.databaseId,
        treeNode.tableId
      );

      const isDropped = editorStore.droppedTableList.includes(treeNode.table);
      if (isDropped) {
        additionalClassList.push("text-red-700 line-through");
      } else if (!isEqual(originTable, treeNode.table)) {
        additionalClassList.push("text-yellow-700");
      }
    }
  }

  return h(
    NEllipsis,
    {
      class: additionalClassList.join(" "),
    },
    [
      h("span", {
        innerHTML: getHighlightHTMLByKeyWords(
          escape(treeNode.label),
          escape(searchPattern.value)
        ),
      }),
    ]
  );
};

// Render a 'menu' icon in the right of the node
const renderSuffix = ({ option: treeNode }: { option: TreeNode }) => {
  if (treeNode.type === "database") {
    return h(EllipsisIcon, {
      class: "w-4 h-auto text-gray-600",
      onClick: (e) => {
        handleShowDropdown(e, treeNode);
      },
    });
  } else if (treeNode.type === "table") {
    return h(EllipsisIcon, {
      class: "w-4 h-auto text-gray-600",
      onClick: (e) => {
        handleShowDropdown(e, treeNode);
      },
    });
  }
  return null;
};

const handleShowDropdown = (e: MouseEvent, treeNode: TreeNode) => {
  e.preventDefault();
  e.stopPropagation();
  contextMenu.treeNode = treeNode;
  contextMenu.showDropdown = true;
  contextMenu.clientX = e.clientX;
  contextMenu.clientY = e.clientY;
  selectedKeysRef.value = [treeNode.key];
};

// Dynamic fetching table list when database tree node clicking.
const loadSubTree = async (treeNode: TreeNode) => {
  if (treeNode.type === "database") {
    const instanceId = treeNode.instanceId;
    const databaseId = treeNode.databaseId;
    const tableList = await editorStore.getOrFetchTableListByDatabaseId(
      databaseId
    );

    const knownTableList = tableList.filter((table) => table.id !== UNKNOWN_ID);
    if (knownTableList.length === 0) {
      treeNode.isLeaf = true;
    } else {
      treeNode.isLeaf = false;
      treeNode.children = knownTableList.map((table) => {
        return {
          type: "table",
          key: `t-${table.id}-${table.name}`,
          label: table.name,
          children: [],
          isLeaf: true,
          instanceId: instanceId,
          databaseId: databaseId,
          tableId: table.id,
          table: table,
        };
      });
    }
  }
};

// Set event handler to tree nodes.
const nodeProps = ({ option: treeNode }: { option: TreeNode }) => {
  return {
    async onClick(e: MouseEvent) {
      // Check if clicked on the content part.
      // And ignore the fold/unfold arrow.
      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        state.shouldRelocateTreeNode = false;

        if (treeNode.type === "instance") {
          // Toggle instance tree node expanded status.
          const index = expandedKeysRef.value.findIndex(
            (key) => key === treeNode.key
          );
          if (index >= 0) {
            expandedKeysRef.value.splice(index, 1);
          } else {
            expandedKeysRef.value.push(treeNode.key);
          }
        } else if (treeNode.type === "database") {
          editorStore.addTab({
            id: generateUniqueTabId(),
            type: UIEditorTabType.TabForDatabase,
            databaseId: treeNode.databaseId as any,
          });
        } else if (treeNode.type === "table") {
          const databaseId = treeNode.databaseId as any;
          const tableId = treeNode.tableId as any;

          editorStore.addTab({
            id: generateUniqueTabId(),
            type: UIEditorTabType.TabForTable,
            databaseId: databaseId,
            tableId: tableId,
            table: treeNode.table,
            tableCache: cloneDeep(treeNode.table),
          });
        }

        nextTick(() => {
          if (treeNode.type === "database") {
            selectedKeysRef.value = [`d-${treeNode.databaseId}`];
          } else if (treeNode.type === "table") {
            selectedKeysRef.value = [`t-${treeNode.tableId}-${treeNode.label}`];
          }
          state.shouldRelocateTreeNode = true;
        });
      } else {
        nextTick(() => {
          selectedKeysRef.value = [];
        });
      }
    },
    onContextmenu(e: MouseEvent) {
      handleShowDropdown(e, treeNode);
    },
  };
};

const handleContextMenuDropdownSelect = async (key: string) => {
  const treeNode = contextMenu.treeNode;
  if (treeNode?.type === "database") {
    if (key === "new-table") {
      await editorStore.getOrFetchTableListByDatabaseId(treeNode.databaseId);
      const table = editorStore.createNewTable(treeNode.databaseId);
      editorStore.addTab({
        id: generateUniqueTabId(),
        type: UIEditorTabType.TabForTable,
        databaseId: treeNode.databaseId,
        tableId: table.id,
        table: table,
        tableCache: cloneDeep(table),
      });
    }
  } else if (treeNode?.type === "table") {
    if (key === "drop") {
      editorStore.dropTable(treeNode.table);
    } else if (key === "restore") {
      editorStore.restoreTable(
        treeNode.databaseId,
        treeNode.tableId,
        treeNode.label
      );
    }
  }
  contextMenu.showDropdown = false;
};

const handleDropdownClickoutside = (e: MouseEvent) => {
  if (
    !isDescendantOf(e.target as Element, ".n-tree-node-wrapper") ||
    e.button !== 2
  ) {
    selectedKeysRef.value = [];
    contextMenu.showDropdown = false;
  }
};

const handleExpandedKeysChange = (expandedKeys: string[]) => {
  expandedKeysRef.value = expandedKeys;
};

const handleSelectedKeysChange = (selectedKeys: string[]) => {
  selectedKeysRef.value = selectedKeys;
};
</script>

<style>
.ui-editor-database-tree .n-tree-node-wrapper {
  @apply !py-px;
}
.ui-editor-database-tree .n-tree-node-content__prefix {
  @apply shrink-0 !mr-1;
}
.ui-editor-database-tree .n-tree-node-content__text {
  @apply truncate mr-1;
}
.ui-editor-database-tree .n-tree-node-content__suffix {
  @apply rounded-sm !hidden hover:opacity-80;
}
.ui-editor-database-tree
  .n-tree-node-wrapper:hover
  .n-tree-node-content__suffix {
  @apply !flex;
}
.ui-editor-database-tree
  .n-tree-node-wrapper
  .n-tree-node--selected
  .n-tree-node-content__suffix {
  @apply !flex;
}
.ui-editor-database-tree .n-tree-node-switcher {
  @apply px-0 !w-4 !h-7;
}
</style>

<style scoped>
.ui-editor-database-tree {
  @apply overflow-y-auto;
  max-height: calc(100% - 48px);
}
</style>
