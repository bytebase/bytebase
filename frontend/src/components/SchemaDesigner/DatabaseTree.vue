<template>
  <div class="w-full h-full px-2 relative overflow-y-hidden">
    <div class="w-full sticky top-0 pt-2 h-12 bg-white z-10">
      <NInput
        v-model:value="searchPattern"
        :placeholder="$t('schema-editor.search-database-and-table')"
      >
        <template #prefix>
          <heroicons-outline:search class="w-4 h-auto text-gray-300" />
        </template>
      </NInput>
    </div>
    <div ref="treeRef" class="schema-editor-database-tree pb-2 h-auto">
      <NTree
        :block-line="true"
        :data="treeDataRef"
        :pattern="searchPattern"
        :render-prefix="renderPrefix"
        :render-label="renderLabel"
        :render-suffix="renderSuffix"
        :node-props="nodeProps"
        :expanded-keys="expandedKeysRef"
        :selected-keys="selectedKeysRef"
        :on-update:expanded-keys="handleExpandedKeysChange"
        :on-update:selected-keys="handleSelectedKeysChange"
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
    :schema="state.tableNameModalContext.schema"
    @close="state.tableNameModalContext = undefined"
  />
</template>

<script lang="ts" setup>
import { escape, isUndefined } from "lodash-es";
import { TreeOption, NEllipsis, NInput, NDropdown, NTree } from "naive-ui";
import { computed, watch, ref, h, reactive, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import scrollIntoView from "scroll-into-view-if-needed";
import SchemaIcon from "~icons/heroicons-outline/view-columns";
import TableIcon from "~icons/heroicons-outline/table-cells";
import EllipsisIcon from "~icons/heroicons-solid/ellipsis-horizontal";

import { Engine } from "@/types/proto/v1/common";
import { useSchemaDesignerContext, SchemaDesignerTabType } from "./common";
import { generateUniqueTabId } from "@/store";
import { getHighlightHTMLByKeyWords, isDescendantOf } from "@/utils";
import SchemaNameModal from "./Modals/SchemaNameModal.vue";
import TableNameModal from "./Modals/TableNameModal.vue";

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
    schema: string;
  };
}

const { t } = useI18n();
const {
  engine,
  editableSchemas,
  tabState,
  addTab,
  getCurrentTab,
  dropSchema,
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
const treeRef = ref();
const searchPattern = ref("");
const expandedKeysRef = ref<string[]>([]);
const selectedKeysRef = ref<string[]>([]);
const treeDataRef = ref<TreeNode[]>([]);

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
    if (engine === Engine.POSTGRES) {
      const schema = schemaList.value.find(
        (schema) => schema.name === treeNode.schema
      );
      if (!schema) {
        return [];
      }

      options.push({
        key: "create-table",
        label: t("schema-editor.actions.create-table"),
      });
      options.push({
        key: "drop-schema",
        label: t("schema-editor.actions.drop-schema"),
      });
    }
    return options;
  } else if (treeNode.type === "table") {
    const schema = schemaList.value.find(
      (schema) => schema.name === treeNode.schema
    );
    if (!schema) {
      return [];
    }

    const table = schema.tableList.find((table) => table.id === treeNode.table);
    if (!table) {
      return [];
    }

    const options = [];
    options.push({
      key: "drop",
      label: t("schema-editor.actions.drop-table"),
    });
    return options;
  }

  return [];
});

watch(
  () => [schemaList.value, tableList.value],
  () => {
    const treeNodeList: TreeNode[] = [];
    if (engine === Engine.MYSQL) {
      const schema = schemaList.value[0];
      if (!schema) {
        return;
      }
      for (const table of tableList.value) {
        const tableTreeNode: TreeNodeForTable = {
          type: "table",
          key: `t-${table.id}`,
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

    treeDataRef.value = treeNodeList;
  },
  {
    immediate: true,
    deep: true,
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
        const element = treeRef.value?.querySelector(".n-tree-node--selected");
        if (element) {
          scrollIntoView(element, {
            scrollMode: "if-needed",
          });
        }
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
  const icon = h(EllipsisIcon, {
    class: "w-4 h-auto text-gray-600",
    onClick: (e) => {
      handleShowDropdown(e, treeNode);
    },
  });
  if (treeNode.type === "schema") {
    if (engine === Engine.POSTGRES) {
      return icon;
    }
  } else if (treeNode.type === "table") {
    return icon;
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

// Set event handler to tree nodes.
const nodeProps = ({ option: treeNode }: { option: TreeNode }) => {
  return {
    onClick(e: MouseEvent) {
      // Check if clicked on the content part.
      // And ignore the fold/unfold arrow.
      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        state.shouldRelocateTreeNode = false;

        if (treeNode.type === "table") {
          addTab({
            id: generateUniqueTabId(),
            type: SchemaDesignerTabType.TabForTable,
            schemaId: treeNode.schemaId,
            tableId: treeNode.tableId,
          });
        }

        nextTick(() => {
          if (treeNode.type === "table") {
            selectedKeysRef.value = [
              `t-${treeNode.databaseId}-${treeNode.tableId}`,
            ];
          }
          state.shouldRelocateTreeNode = true;
        });
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

const handleContextMenuDropdownSelect = async (key: string) => {
  const treeNode = contextMenu.treeNode;
  if (!treeNode) {
    return;
  }

  if (treeNode.type === "schema") {
    if (key === "create-table") {
      state.tableNameModalContext = {
        schema: treeNode.schemaId,
      };
    } else if (key === "drop-schema") {
      dropSchema(treeNode.schemaId);
    }
  } else if (treeNode.type === "table") {
    if (key === "drop") {
      dropTable(treeNode.schemaId, treeNode.tableId);
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
.schema-editor-database-tree .n-tree-node-wrapper {
  @apply !py-px;
}
.schema-editor-database-tree .n-tree-node-content__prefix {
  @apply shrink-0 !mr-1;
}
.schema-editor-database-tree .n-tree-node-content__text {
  @apply truncate mr-1;
}
.schema-editor-database-tree .n-tree-node-content__suffix {
  @apply rounded-sm !hidden hover:opacity-80;
}
.schema-editor-database-tree
  .n-tree-node-wrapper:hover
  .n-tree-node-content__suffix {
  @apply !flex;
}
.schema-editor-database-tree
  .n-tree-node-wrapper
  .n-tree-node--selected
  .n-tree-node-content__suffix {
  @apply !flex;
}
.schema-editor-database-tree .n-tree-node-switcher {
  @apply px-0 !w-4 !h-7;
}
</style>

<style scoped>
.schema-editor-database-tree {
  @apply overflow-y-auto;
  max-height: calc(100% - 48px);
}
</style>
