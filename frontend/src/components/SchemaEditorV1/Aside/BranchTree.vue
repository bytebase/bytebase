<template>
  <div
    class="w-full h-full px-2 flex flex-col gap-y-2 relative overflow-y-hidden"
  >
    <div class="w-full flex pt-2 bg-white z-10 space-x-2">
      <NInput
        v-model:value="searchPattern"
        size="small"
        :placeholder="$t('schema-designer.search-tables')"
      >
        <template #prefix>
          <heroicons-outline:search class="h-auto text-gray-300" />
        </template>
      </NInput>
      <button
        v-if="!readonly"
        class="text-gray-400 hover:text-gray-500 disabled:cursor-not-allowed"
        @click="handleCreateSchemaOrTable"
      >
        <heroicons-outline:plus class="w-4 h-auto" />
      </button>
    </div>
    <div
      ref="treeContainerElRef"
      class="flex-1 text-sm overflow-hidden select-none"
      :data-height="treeContainerHeight"
    >
      <NTree
        :key="treeKeyRef"
        ref="treeRef"
        block-line
        virtual-scroll
        :style="{
          height: `${treeContainerHeight}px`,
        }"
        :data="treeData"
        :pattern="searchPattern"
        :render-prefix="renderPrefix"
        :render-label="renderLabel"
        :render-suffix="renderSuffix"
        :node-props="nodeProps"
        :expanded-keys="expandedKeysRef"
        :selected-keys="selectedKeysRef"
        :on-update:expanded-keys="handleExpandedKeysChange"
        :theme-overrides="{ nodeHeight: '28px' }"
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
    :parent-name="state.schemaNameModalContext.parentName"
    @close="state.schemaNameModalContext = undefined"
  />

  <TableNameModal
    v-if="state.tableNameModalContext !== undefined"
    :parent-name="state.tableNameModalContext.parentName"
    :schema-id="state.tableNameModalContext.schemaId"
    :table-id="state.tableNameModalContext.tableId"
    @close="state.tableNameModalContext = undefined"
  />
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { escape, head, isUndefined } from "lodash-es";
import { TreeOption, NEllipsis, NInput, NDropdown, NTree } from "naive-ui";
import { v1 as uuidv1 } from "uuid";
import { computed, watch, ref, h, reactive, nextTick, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import DuplicateIcon from "~icons/heroicons-outline/document-duplicate";
import TableIcon from "~icons/heroicons-outline/table-cells";
import SchemaIcon from "~icons/heroicons-outline/view-columns";
import EllipsisIcon from "~icons/heroicons-solid/ellipsis-horizontal";
import { generateUniqueTabId, useSchemaEditorV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { Table } from "@/types/v1/schemaEditor";
import { BranchSchema, SchemaEditorTabType } from "@/types/v1/schemaEditor";
import { getHighlightHTMLByKeyWords, isDescendantOf } from "@/utils";
import SchemaNameModal from "../Modals/SchemaNameModal.vue";
import TableNameModal from "../Modals/TableNameModal.vue";
import { isTableChanged } from "../utils";
import { isSchemaChanged } from "../utils/schema";

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
    parentName: string;
  };
  tableNameModalContext?: {
    parentName: string;
    schemaId: string;
    tableId?: string;
  };
}

const { t } = useI18n();
const schemaEditorV1Store = useSchemaEditorV1Store();
const state = reactive<LocalState>({
  shouldRelocateTreeNode: false,
});
const readonly = computed(() => schemaEditorV1Store.readonly);
const currentTab = computed(() => schemaEditorV1Store.currentTab);

const treeContainerElRef = ref<HTMLElement>();
const { height: treeContainerHeight } = useElementSize(treeContainerElRef);
const treeRef = ref<InstanceType<typeof NTree>>();
const searchPattern = ref("");
const expandedKeysRef = ref<string[]>([]);
const selectedKeysRef = ref<string[]>([]);
// Trigger re-render when the tree data is changed.
const treeKeyRef = ref<string>("");
const contextMenu = reactive<TreeContextMenu>({
  showDropdown: false,
  clientX: 0,
  clientY: 0,
  treeNode: undefined,
});

// NOTE: we only support editing one branch for now.
const branchSchema = computed(
  () =>
    head(
      Array.from(schemaEditorV1Store.resourceMap.branch.values())
    ) as BranchSchema
);
const engine = computed(() => {
  return (branchSchema.value.branch as any as SchemaDesign).engine;
});

const schemaList = computed(() => branchSchema.value.schemaList);
const tableList = computed(() =>
  schemaList.value.map((schema) => schema.tableList).flat()
);
const treeData = computed(() => {
  const treeNodeList: TreeNode[] = [];
  for (const schema of schemaList.value) {
    const schemaTreeNode: TreeNodeForSchema = {
      type: "schema",
      key: `s-${schema.id}`,
      label: schema.name,
      isLeaf: false,
      schemaId: schema.id,
      children: schema.tableList.map((table) => ({
        type: "table",
        key: `t-${schema.id}-${table.id}`,
        label: table.name,
        children: [],
        isLeaf: true,
        schemaId: schema.id,
        tableId: table.id,
      })),
    };
    if (schemaTreeNode.children!.length === 0) {
      schemaTreeNode.isLeaf = true;
    }
    treeNodeList.push(schemaTreeNode);
  }

  return treeNodeList;
});

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

      const isDropped = schema.status === "dropped";
      if (isDropped) {
        options.push({
          key: "restore",
          label: t("schema-editor.actions.restore"),
        });
      } else {
        options.push({
          key: "create-table",
          label: t("schema-editor.actions.create-table"),
          disabled: readonly.value,
        });
        options.push({
          key: "drop",
          label: t("schema-editor.actions.drop-schema"),
          disabled: readonly.value,
        });
      }
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
        key: "rename",
        label: t("schema-editor.actions.rename"),
      });
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

onMounted(() => {
  if (!branchSchema.value) {
    throw new Error("branch should not be empty");
  }
});

watch(
  () => treeData.value,
  () => {
    treeKeyRef.value = Math.random().toString();
    const firstChildNode = head(treeData.value);
    if (firstChildNode) {
      nextTick(() => {
        // Auto expand the first tree node.
        openTabForTreeNode(firstChildNode);
      });
    }
  },
  {
    deep: true,
    immediate: true,
  }
);

watch(
  () => currentTab.value,
  () => {
    if (!currentTab.value) {
      selectedKeysRef.value = [];
      return;
    }

    if (currentTab.value.type === SchemaEditorTabType.TabForDatabase) {
      if (currentTab.value.selectedSchemaId) {
        const schemaTreeNodeKey = `s-${currentTab.value.selectedSchemaId}`;
        selectedKeysRef.value = [schemaTreeNodeKey];
      }
    } else if (currentTab.value.type === SchemaEditorTabType.TabForTable) {
      const schemaTreeNodeKey = `s-${currentTab.value.schemaId}`;
      if (!expandedKeysRef.value.includes(schemaTreeNodeKey)) {
        expandedKeysRef.value.push(schemaTreeNodeKey);
      }
      const tableTreeNodeKey = `t-${currentTab.value.schemaId}-${currentTab.value.tableId}`;
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
const renderPrefix = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
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
const renderLabel = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  const additionalClassList: string[] = ["select-none"];
  let label = treeNode.label;

  if (treeNode.type === "schema") {
    const schema = schemaList.value.find(
      (schema) => schema.id === treeNode.schemaId
    );
    if (schema) {
      if (engine.value !== Engine.POSTGRES) {
        label = "Tables";
      }
      if (schema.status === "created") {
        additionalClassList.push("text-green-700");
      } else if (schema.status === "dropped") {
        additionalClassList.push("text-red-700 line-through");
      } else {
        if (
          isSchemaChanged(branchSchema.value.branch.name, treeNode.schemaId)
        ) {
          additionalClassList.push("text-yellow-700");
        }
      }
    }
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
      } else if (
        isTableChanged(
          branchSchema.value.branch.name,
          treeNode.schemaId,
          treeNode.tableId
        )
      ) {
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
          escape(label),
          escape(searchPattern.value)
        ),
      }),
    ]
  );
};

// Render a 'menu' icon in the right of the node
const renderSuffix = ({ option }: { option: TreeOption }) => {
  if (readonly.value) {
    return null;
  }

  const treeNode = option as TreeNode;
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
      openTabForTreeNode(treeNode);
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

const handleCreateSchemaOrTable = () => {
  if (engine.value === Engine.MYSQL || engine.value === Engine.TIDB) {
    const schema = schemaList.value[0];
    if (schema) {
      state.tableNameModalContext = {
        parentName: branchSchema.value.branch.name,
        schemaId: schema.id,
      };
    }
  } else if (engine.value === Engine.POSTGRES) {
    state.schemaNameModalContext = {
      parentName: branchSchema.value.branch.name,
    };
  }
};

// Set event handler to tree nodes.
const nodeProps = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  return {
    onClick(e: MouseEvent) {
      // Check if clicked on the content part.
      // And ignore the fold/unfold arrow.
      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        openTabForTreeNode(treeNode);
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

const openTabForTreeNode = (treeNode: TreeNode) => {
  state.shouldRelocateTreeNode = false;

  if (treeNode.type === "table") {
    schemaEditorV1Store.addTab({
      id: generateUniqueTabId(),
      type: SchemaEditorTabType.TabForTable,
      parentName: branchSchema.value.branch.name,
      schemaId: treeNode.schemaId,
      tableId: treeNode.tableId,
      name: treeNode.label,
    });
  } else if (treeNode.type === "schema") {
    const index = expandedKeysRef.value.findIndex(
      (key) => key === treeNode.key
    );
    if (index < 0) {
      expandedKeysRef.value.push(treeNode.key);
    }
    schemaEditorV1Store.addTab({
      id: generateUniqueTabId(),
      type: SchemaEditorTabType.TabForDatabase,
      parentName: branchSchema.value.branch.name,
      selectedSchemaId: treeNode.schemaId,
      name: treeNode.label,
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
        parentName: branchSchema.value.branch.name,
        schemaId: treeNode.schemaId,
      };
    } else if (key === "drop") {
      schemaEditorV1Store.dropSchema(
        branchSchema.value.branch.name,
        treeNode.schemaId
      );
    } else if (key === "restore") {
      const schema = schemaEditorV1Store.getSchema(
        branchSchema.value.branch.name,
        treeNode.schemaId
      );
      if (!schema) {
        return;
      }
      schema.status = "normal";
    }
  } else if (treeNode.type === "table") {
    if (key === "rename") {
      state.tableNameModalContext = {
        parentName: branchSchema.value.branch.name,
        schemaId: treeNode.schemaId,
        tableId: treeNode.tableId,
      };
    } else if (key === "drop") {
      schemaEditorV1Store.dropTable(
        branchSchema.value.branch.name,
        treeNode.schemaId,
        treeNode.tableId
      );
    } else if (key === "restore") {
      const table = schemaEditorV1Store.getTable(
        branchSchema.value.branch.name,
        treeNode.schemaId,
        treeNode.tableId
      );
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

const handleExpandedKeysChange = (expandedKeys: string[]) => {
  expandedKeysRef.value = expandedKeys;
};
</script>

<style>
.schema-designer-database-tree .n-tree-node-wrapper {
  @apply !p-0;
}
.schema-designer-database-tree .n-tree-node-content {
  @apply !pl-2 text-sm;
}
.schema-designer-database-tree .n-tree-node-indent {
  width: 0.25rem;
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
