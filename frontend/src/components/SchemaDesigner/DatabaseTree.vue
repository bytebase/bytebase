<template>
  <div class="w-full h-full px-2 relative overflow-y-hidden">
    <div class="w-full flex flex-col sticky top-0 pt-2 h-20 bg-white z-10">
      <p class="w-full flex flex-row justify-between items-center h-8 px-1">
        <span class="text-sm">{{ $t("schema-designer.tables") }}</span>
        <button
          v-if="!readonly"
          class="text-gray-400 hover:text-gray-500 disabled:cursor-not-allowed"
          @click="handleCreateTable"
        >
          <heroicons-outline:plus class="w-4 h-auto" />
        </button>
      </p>
      <NInput
        v-model:value="searchPattern"
        :placeholder="$t('schema-designer.search-tables')"
      >
        <template #prefix>
          <heroicons-outline:search class="w-4 h-auto text-gray-300" />
        </template>
      </NInput>
    </div>
    <div
      class="schema-designer-database-tree pb-2 overflow-y-auto h-full text-sm"
    >
      <NTree
        ref="treeRef"
        :key="treeKeyRef"
        virtual-scroll
        style="height: 100%"
        :block-line="true"
        :data="treeData"
        :pattern="searchPattern"
        :render-prefix="renderPrefix"
        :render-label="renderLabel"
        :render-suffix="renderSuffix"
        :node-props="nodeProps"
        :expanded-keys="expandedKeysRef"
        :selected-keys="selectedKeysRef"
      />
      <NDropdown
        trigger="manual"
        placement="bottom-start"
        :show="contextMenu.showDropdown"
        :options="contextMenuOptions"
        :x="contextMenu.clientX"
        :y="contextMenu.clientY"
        to="body"
        @select="handleContextMenuDropdownSelect"
        @clickoutside="handleDropdownClickoutside"
      />
    </div>
  </div>

  <SchemaNameModal
    v-if="state.schemaNameModalContext !== undefined"
    :database-id="state.schemaNameModalContext.databaseId"
    @close="state.schemaNameModalContext = undefined"
  />

  <TableNameModal
    v-if="state.tableNameModalContext !== undefined"
    :schema-id="state.tableNameModalContext.schemaId"
    @close="state.tableNameModalContext = undefined"
  />
</template>

<script lang="ts" setup>
import { escape, isUndefined } from "lodash-es";
import { TreeOption, NEllipsis, NInput, NDropdown, NTree } from "naive-ui";
import { v1 as uuidv1 } from "uuid";
import { computed, watch, ref, h, reactive, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import DuplicateIcon from "~icons/heroicons-outline/document-duplicate";
import TableIcon from "~icons/heroicons-outline/table-cells";
import SchemaIcon from "~icons/heroicons-outline/view-columns";
import EllipsisIcon from "~icons/heroicons-solid/ellipsis-horizontal";
import { generateUniqueTabId } from "@/store";
import { Table } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { getHighlightHTMLByKeyWords, isDescendantOf } from "@/utils";
import SchemaNameModal from "./Modals/SchemaNameModal.vue";
import TableNameModal from "./Modals/TableNameModal.vue";
import { useSchemaDesignerContext, SchemaDesignerTabType } from "./common";
import { isTableChanged } from "./utils/table";

interface BaseTreeNode extends TreeOption {
  key: string;
  label: string;
  isLeaf: boolean;
  children?: TreeNode[];
}

interface TreeNodeForSchema extends BaseTreeNode {
  type: "schema";
  schemaId: string;
}

interface TreeNodeForTable extends BaseTreeNode {
  type: "table";
  schemaId: string;
  tableId: string;
}

type TreeNode = TreeNodeForSchema | TreeNodeForTable;

interface TreeContextMenu {
  showDropdown: boolean;
  clientX: number;
  clientY: number;
  treeNode?: TreeNode;
}

interface LocalState {
  shouldRelocateTreeNode: boolean;
  schemaNameModalContext?: {
    databaseId: string;
  };
  tableNameModalContext?: {
    schemaId: string;
  };
}

const { t } = useI18n();
const {
  readonly,
  engine,
  editableSchemas,
  tabState,
  addTab,
  getTable,
  getCurrentTab,
  dropTable,
} = useSchemaDesignerContext();
const state = reactive<LocalState>({
  shouldRelocateTreeNode: false,
});
const contextMenu = reactive<TreeContextMenu>({
  showDropdown: false,
  clientX: 0,
  clientY: 0,
  treeNode: undefined,
});
const treeRef = ref<InstanceType<typeof NTree>>();
const searchPattern = ref("");
const expandedKeysRef = ref<string[]>([]);
const selectedKeysRef = ref<string[]>([]);
// Trigger re-render when the tree data is changed.
const treeKeyRef = ref<string>("");

const treeData = computed(() => {
  const treeNodeList: TreeNode[] = [];
  if (engine.value === Engine.MYSQL) {
    const schema = schemaList.value[0];
    if (!schema) {
      return;
    }
    for (const table of tableList.value) {
      const tableTreeNode: TreeNodeForTable = {
        type: "table",
        key: `t-${schema.id}-${table.id}`,
        label: table.name,
        isLeaf: true,
        schemaId: schema.id,
        tableId: table.id,
      };
      treeNodeList.push(tableTreeNode);
    }
  } else {
    for (const schema of schemaList.value) {
      const schemaTreeNode: TreeNodeForSchema = {
        type: "schema",
        key: `s-${schema.id}`,
        label: schema.name,
        isLeaf: false,
        schemaId: schema.id,
      };
      treeNodeList.push(schemaTreeNode);
      for (const table of schema.tableList) {
        const tableTreeNode: TreeNodeForTable = {
          type: "table",
          key: `t-${schema.id}-${table.id}`,
          label: table.name,
          isLeaf: true,
          schemaId: schema.id,
          tableId: table.id,
        };
        schemaTreeNode.children?.push(tableTreeNode);
      }
    }
  }

  return treeNodeList;
});
const schemaList = computed(() => editableSchemas.value);
const tableList = computed(() =>
  schemaList.value.map((schema) => schema.tableList).flat()
);
const contextMenuOptions = computed(() => {
  const treeNode = contextMenu.treeNode;
  if (isUndefined(treeNode)) {
    return [];
  }

  if (treeNode.type === "schema") {
    const options = [];
    if (engine.value === Engine.POSTGRES) {
      const schema = schemaList.value.find(
        (schema) => schema.id === treeNode.schemaId
      );
      if (!schema) {
        return [];
      }

      options.push({
        key: "create-table",
        label: t("schema-editor.actions.create-table"),
        disabled: readonly.value,
      });
      options.push({
        key: "drop-schema",
        label: t("schema-editor.actions.drop-schema"),
        disabled: readonly.value,
      });
    }
    return options;
  } else if (treeNode.type === "table") {
    const schema = schemaList.value.find(
      (schema) => schema.id === treeNode.schemaId
    );
    if (!schema) {
      return [];
    }

    const table = schema.tableList.find(
      (table) => table.id === treeNode.tableId
    );
    if (!table) {
      return [];
    }

    const isDropped = table.status === "dropped";
    const options = [];
    if (isDropped) {
      options.push({
        key: "restore",
        label: t("schema-editor.actions.restore"),
      });
    } else {
      options.push({
        key: "drop",
        label: t("schema-editor.actions.drop-table"),
        disabled: readonly.value,
      });
    }
    return options;
  }

  return [];
});

watch(
  () => treeData.value,
  () => {
    treeKeyRef.value = Math.random().toString();
  },
  {
    deep: true,
    immediate: true,
  }
);

watch(
  () => tabState.value.currentTabId,
  () => {
    const currentTab = getCurrentTab();
    if (!currentTab) {
      selectedKeysRef.value = [];
      return;
    }

    if (currentTab.type === SchemaDesignerTabType.TabForTable) {
      const schemaTreeNodeKey = `s-${currentTab.schemaId}`;
      if (!expandedKeysRef.value.includes(schemaTreeNodeKey)) {
        expandedKeysRef.value.push(schemaTreeNodeKey);
      }
      const tableTreeNodeKey = `t-${currentTab.schemaId}-${currentTab.tableId}`;
      selectedKeysRef.value = [tableTreeNodeKey];
    }

    if (state.shouldRelocateTreeNode) {
      nextTick(() => {
        treeRef.value?.scrollTo({
          key: selectedKeysRef.value[0],
        });
      });
    }
  }
);

// Render prefix icons before label text.
const renderPrefix = ({ option: treeNode }: { option: TreeNode }) => {
  if (treeNode.type === "schema") {
    return h(SchemaIcon, {
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

  if (treeNode.type === "schema") {
    // do nothing
  } else if (treeNode.type === "table") {
    const table = tableList.value.find(
      (table) => table.id === treeNode.tableId
    );

    if (table) {
      if (table.status === "created") {
        additionalClassList.push("text-green-700");
      } else if (table.status === "dropped") {
        additionalClassList.push("text-red-700 line-through");
      } else if (isTableChanged(treeNode.schemaId, treeNode.tableId)) {
        additionalClassList.push("text-yellow-700");
      }
    }
  }

  return h(
    NEllipsis,
    {
      class: additionalClassList.join(" "),
    },
    () => [
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
  if (readonly.value) {
    return null;
  }

  const menuIcon = h(EllipsisIcon, {
    class: "w-4 h-auto text-gray-600",
    onClick: (e) => {
      handleShowDropdown(e, treeNode);
    },
  });
  const duplicateIcon = h(DuplicateIcon, {
    class: "w-4 h-auto mr-2 text-gray-600",
    onClick: (e) => {
      e.preventDefault();
      e.stopPropagation();
      e.stopImmediatePropagation();

      const schema = schemaList.value.find(
        (schema) => schema.id === treeNode.schemaId
      );
      if (!schema) {
        return;
      }
      const table = tableList.value.find((t) => t.id === treeNode.tableId);
      if (!table) {
        return;
      }

      const matchPattern = new RegExp(
        `^${getOriginalName(table.name)}` + "(_copy[0-9]{0,}){0,1}$"
      );
      const copiedTableNames = tableList.value
        .filter((table) => {
          return matchPattern.test(table.name);
        })
        .sort((t1, t2) => {
          return (
            extractDuplicateNumber(t1.name) - extractDuplicateNumber(t2.name)
          );
        });
      const targetName = copiedTableNames.slice(-1)[0]?.name ?? table.name;

      const newTable: Table = {
        ...table,
        id: uuidv1(),
        name: getDuplicateName(targetName),
        status: "created",
        primaryKey: {
          name: "",
          columnIdList: [],
        },
        columnList: table.columnList.map((column) => {
          return {
            ...column,
            id: uuidv1(),
            status: "created",
          };
        }),
      };

      for (const primaryKeyId of table.primaryKey.columnIdList) {
        const column = table.columnList.find((col) => col.id === primaryKeyId);
        if (!column) {
          continue;
        }
        const newColumn = newTable.columnList.find(
          (col) => col.name === column.name
        );
        if (!newColumn) {
          continue;
        }
        newTable.primaryKey.columnIdList.push(newColumn.id);
      }

      schema.tableList.push(newTable);

      openTabForTable(treeNode, newTable.id);
    },
  });
  if (treeNode.type === "schema") {
    if (engine.value === Engine.POSTGRES) {
      return [menuIcon];
    }
  } else if (treeNode.type === "table") {
    const icons = [menuIcon];
    if (!readonly.value) {
      icons.unshift(duplicateIcon);
    }
    return icons;
  }
  return null;
};

const getOriginalName = (name: string): string => {
  return name.replace(/_copy[0-9]{0,}$/, "");
};

const extractDuplicateNumber = (name: string): number => {
  const match = /_copy[0-9]{0,}$/.exec(name);
  if (!match) {
    return -1;
  }

  const num = Number(match[0].replace("_copy", "0"));
  if (Number.isNaN(num)) {
    return -1;
  }
  return num;
};

const getDuplicateName = (name: string): string => {
  const num = extractDuplicateNumber(name);
  if (num < 0) {
    return `${name}_copy`;
  }
  return `${getOriginalName(name)}_copy${num + 1}`;
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

const handleCreateTable = () => {
  if (engine.value === Engine.MYSQL) {
    const schema = editableSchemas.value[0];
    state.tableNameModalContext = {
      schemaId: schema.id,
    };
  }
};

// Set event handler to tree nodes.
const nodeProps = ({ option: treeNode }: { option: TreeNode }) => {
  return {
    onClick(e: MouseEvent) {
      // Check if clicked on the content part.
      // And ignore the fold/unfold arrow.
      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        openTabForTable(treeNode, treeNode.tableId as string);
      } else {
        nextTick(() => {
          selectedKeysRef.value = [];
        });
      }
    },
    onContextMenu(e: MouseEvent) {
      handleShowDropdown(e, treeNode);
    },
  };
};

const openTabForTable = (treeNode: TreeNode, tableId: string) => {
  state.shouldRelocateTreeNode = false;

  if (treeNode.type === "table") {
    addTab({
      id: generateUniqueTabId(),
      type: SchemaDesignerTabType.TabForTable,
      schemaId: treeNode.schemaId,
      tableId: tableId,
    });
  }

  state.shouldRelocateTreeNode = true;
};

const handleContextMenuDropdownSelect = async (key: string) => {
  const treeNode = contextMenu.treeNode;
  if (!treeNode) {
    return;
  }

  if (treeNode.type === "schema") {
    if (key === "create-table") {
      state.tableNameModalContext = {
        schemaId: treeNode.schemaId,
      };
    }
  } else if (treeNode.type === "table") {
    if (key === "drop") {
      dropTable(treeNode.schemaId, treeNode.tableId);
    } else if (key === "restore") {
      const table = getTable(treeNode.schemaId, treeNode.tableId);
      if (!table) {
        return;
      }
      table.status = "normal";
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
</script>

<style>
.schema-designer-database-tree .n-tree-node-wrapper {
  @apply !py-px;
}
.schema-designer-database-tree .n-tree-node-content__prefix {
  @apply shrink-0 !mr-1;
}
.schema-designer-database-tree .n-tree-node-content__text {
  @apply truncate mr-1;
}
.schema-designer-database-tree .n-tree-node-content__suffix {
  @apply rounded-sm !hidden hover:opacity-80;
}
.schema-designer-database-tree
  .n-tree-node-wrapper:hover
  .n-tree-node-content__suffix {
  @apply !flex;
}
.schema-designer-database-tree
  .n-tree-node-wrapper
  .n-tree-node--selected
  .n-tree-node-content__suffix {
  @apply !flex;
}
.schema-designer-database-tree .n-tree-node-switcher {
  @apply px-0 !w-4 !h-7;
}
</style>

<style scoped>
.schema-designer-database-tree {
  max-height: calc(100% - 80px);
}
</style>
