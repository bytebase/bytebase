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
      class="schema-designer-database-tree flex-1 pb-1 text-sm overflow-hidden select-none"
      :data-height="treeContainerHeight"
    >
      <NTree
        v-if="treeContainerHeight > 0"
        ref="treeRef"
        block-line
        virtual-scroll
        :style="{
          height: `${treeContainerHeight}px`,
        }"
        :data="treeDataRef"
        :pattern="searchPattern"
        :render-prefix="renderPrefix"
        :render-label="renderLabel"
        :render-suffix="renderSuffix"
        :node-props="nodeProps"
        :expanded-keys="expandedKeysRef"
        :selected-keys="selectedKeysRef"
        :show-irrelevant-nodes="false"
        :theme-overrides="{ nodeHeight: '28px' }"
        @update:expanded-keys="handleExpandedKeysChange"
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
    :database="state.schemaNameModalContext.database"
    :metadata="state.schemaNameModalContext.metadata"
    @close="state.schemaNameModalContext = undefined"
  />

  <TableNameModal
    v-if="state.tableNameModalContext !== undefined"
    :database="state.tableNameModalContext.database"
    :metadata="state.tableNameModalContext.metadata"
    :schema="state.tableNameModalContext.schema"
    :table="state.tableNameModalContext.table"
    @close="state.tableNameModalContext = undefined"
  />
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { cloneDeep, escape, head } from "lodash-es";
import { TreeOption, NEllipsis, NInput, NDropdown, NTree } from "naive-ui";
import { computed, watch, ref, h, reactive, nextTick, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import DuplicateIcon from "~icons/heroicons-outline/document-duplicate";
import TableIcon from "~icons/heroicons-outline/table-cells";
import SchemaIcon from "~icons/heroicons-outline/view-columns";
import EllipsisIcon from "~icons/heroicons-solid/ellipsis-horizontal";
import { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { getHighlightHTMLByKeyWords, isDescendantOf } from "@/utils";
import SchemaNameModal from "../Modals/SchemaNameModal.vue";
import TableNameModal from "../Modals/TableNameModal.vue";
import { useSchemaEditorContext } from "../context";
import { keyForResource, keyForResourceName } from "../context/common";

interface BaseTreeNode extends TreeOption {
  key: string;
  label: string;
  isLeaf: boolean;
  children?: TreeNode[];
}

interface TreeNodeForSchema extends BaseTreeNode {
  type: "schema";
  schema: SchemaMetadata;
}

interface TreeNodeForTable extends BaseTreeNode {
  type: "table";
  schema: SchemaMetadata;
  table: TableMetadata;
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
    database: ComposedDatabase;
    metadata: DatabaseMetadata;
  };
  tableNameModalContext?: {
    database: ComposedDatabase;
    metadata: DatabaseMetadata;
    schema: SchemaMetadata;
    table?: TableMetadata;
  };
}

const { t } = useI18n();

const state = reactive<LocalState>({
  shouldRelocateTreeNode: false,
});
const {
  targets,
  readonly,
  currentTab,
  addTab,
  markEditStatus,
  removeEditStatus,
  getSchemaStatus,
  getTableStatus,
  upsertTableConfig,
} = useSchemaEditorContext();
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
const expandedKeysRef = ref<string[]>([]);
const selectedKeysRef = ref<string[]>([]);
const treeDataRef = ref<TreeNode[]>([]);
const contextMenu = reactive<TreeContextMenu>({
  showDropdown: false,
  clientX: 0,
  clientY: 0,
  treeNode: undefined,
});

// NOTE: we only support editing one branch for now.
const database = computed(() => {
  return targets.value[0].database;
});
const metadata = computed(() => {
  return targets.value[0].metadata;
});
const engine = computed(() => {
  return database.value.instanceEntity.engine;
});

const schemaList = computed(() => {
  return metadata.value.schemas;
});
const flattenTableList = computed(() => {
  return metadata.value.schemas.flatMap((schema) => schema.tables);
});

const metadataForSchema = (schema: SchemaMetadata) => {
  return {
    database: metadata.value,
    schema,
  };
};
const metadataForTable = (schema: SchemaMetadata, table: TableMetadata) => {
  return { ...metadataForSchema(schema), table };
};
const statusForSchema = (schema: SchemaMetadata) => {
  return getSchemaStatus(database.value, metadataForSchema(schema));
};
const statusForTable = (schema: SchemaMetadata, table: TableMetadata) => {
  return getTableStatus(database.value, metadataForTable(schema, table));
};

const contextMenuOptions = computed(() => {
  const { treeNode } = contextMenu;
  if (!treeNode) return;

  if (treeNode.type === "schema") {
    const options = [];
    if (engine.value === Engine.POSTGRES) {
      const { schema } = treeNode;

      const status = statusForSchema(schema);
      const isDropped = status === "dropped";
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
    const { schema, table } = treeNode;

    const status = statusForTable(schema, table);
    const options = [];
    if (status === "dropped") {
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
    if (status === "created") {
      options.push({
        key: "rename",
        label: t("schema-editor.actions.rename"),
      });
    }
    return options;
  }

  return [];
});

onMounted(() => {
  buildBranchTreeData();
});

const buildBranchTreeData = () => {
  const db = database.value;
  const treeNodeList: TreeNode[] = [];
  for (const schema of schemaList.value) {
    const schemaTreeNode: TreeNodeForSchema = {
      type: "schema",
      key: keyForResource(db, {
        schema,
      }),
      label: schema.name,
      isLeaf: false,
      schema: schema,
      children: schema.tables.map<TreeNodeForTable>((table) => ({
        type: "table",
        key: keyForResource(db, {
          schema,
          table,
        }),
        label: table.name,
        children: [],
        isLeaf: true,
        schema: schema,
        table: table,
      })),
    };
    if (schemaTreeNode.children!.length === 0) {
      schemaTreeNode.isLeaf = true;
    }
    treeNodeList.push(schemaTreeNode);
  }
  treeDataRef.value = treeNodeList;
  const firstChildNode = head(treeDataRef.value);
  if (firstChildNode) {
    nextTick(() => {
      // Auto expand the first tree node.
      openTabForTreeNode(firstChildNode);
    });
  }
};

watch(
  [() => schemaList.value.length, () => flattenTableList.value.length],
  buildBranchTreeData,
  {
    deep: false,
  }
);

const tabWatchKey = computed(() => {
  const tab = currentTab.value;
  if (!tab) return undefined;
  if (tab.type === "database") {
    return keyForResourceName(tab.database.name, tab.selectedSchema);
  }
  return keyForResource(tab.database, tab.metadata);
});
watch(tabWatchKey, () => {
  const tab = currentTab.value;
  if (!tab) {
    selectedKeysRef.value = [];
    return;
  }

  if (tab.type === "database") {
    const { database, selectedSchema: schema } = tab;
    if (schema) {
      const key = keyForResourceName(database.name, schema);
      if (!expandedKeysRef.value.includes(key)) {
        expandedKeysRef.value.push(key);
      }
      selectedKeysRef.value = [key];
    }
  } else if (tab.type === "table") {
    const {
      database,
      metadata: { schema, table },
    } = tab;
    const schemaKey = keyForResource(database, { schema });
    if (!expandedKeysRef.value.includes(schemaKey)) {
      expandedKeysRef.value.push(schemaKey);
    }
    const tableKey = keyForResource(database, { schema, table });
    selectedKeysRef.value = [tableKey];
  }

  if (state.shouldRelocateTreeNode) {
    nextTick(() => {
      treeRef.value?.scrollTo({
        key: selectedKeysRef.value[0],
      });
    });
  }
});

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
    const { schema } = treeNode as TreeNodeForSchema;
    if (schema) {
      if (engine.value !== Engine.POSTGRES) {
        label = t("db.tables");
      }
      const status = statusForSchema(schema);
      if (status === "created") {
        additionalClassList.push("text-green-700");
      } else if (status === "dropped") {
        additionalClassList.push("text-red-700 line-through");
      } else if (status === "updated") {
        additionalClassList.push("text-yellow-700");
      }
    }
  } else if (treeNode.type === "table") {
    const { schema, table } = treeNode as TreeNodeForTable;

    if (table) {
      const status = statusForTable(schema, table);
      if (status === "created") {
        additionalClassList.push("text-green-700");
      } else if (status === "dropped") {
        additionalClassList.push("text-red-700 line-through");
      } else if (status === "updated") {
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

      const db = database.value;
      const { schema, table } = treeNode as TreeNodeForTable;
      if (!schema || !table) {
        return;
      }

      const matchPattern = new RegExp(
        `^${getOriginalName(table.name)}` + "(_copy[0-9]{0,}){0,1}$"
      );
      const copiedTableNames = flattenTableList.value
        .filter((table) => {
          return matchPattern.test(table.name);
        })
        .sort((t1, t2) => {
          return (
            extractDuplicateNumber(t1.name) - extractDuplicateNumber(t2.name)
          );
        });
      const targetName = copiedTableNames.slice(-1)[0]?.name ?? table.name;

      const newTable = cloneDeep(table);
      newTable.name = getDuplicateName(targetName);
      schema.tables.push(newTable);
      markEditStatus(db, metadataForTable(schema, newTable), "created");
      newTable.columns.forEach((newColumn) => {
        markEditStatus(
          db,
          { ...metadataForTable(schema, newTable), column: newColumn },
          "created"
        );
      });
      const tableConfig = metadata.value.schemaConfigs
        .find((sc) => sc.name === schema.name)
        ?.tableConfigs.find((tc) => tc.name === table.name);
      if (tableConfig) {
        const tableConfigCopy = cloneDeep(tableConfig);
        tableConfigCopy.name = newTable.name;
        upsertTableConfig(
          db,
          {
            database: metadata.value,
            schema,
            table: newTable,
          },
          (config) => {
            Object.assign(config, tableConfigCopy);
          }
        );
      }
      addTab({
        type: "table",
        database: database.value,
        metadata: {
          database: metadata.value,
          schema: schema,
          table: newTable,
        },
      });
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
    const schema = head(schemaList.value);
    if (schema) {
      state.tableNameModalContext = {
        database: database.value,
        metadata: metadata.value,
        schema,
        table: undefined,
      };
    }
  } else if (engine.value === Engine.POSTGRES) {
    state.schemaNameModalContext = {
      database: database.value,
      metadata: metadata.value,
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

const openTabForTreeNode = (node: TreeNode) => {
  state.shouldRelocateTreeNode = false;

  if (node.type === "table") {
    addTab({
      type: "table",
      database: database.value,
      metadata: {
        database: metadata.value,
        schema: node.schema,
        table: node.table,
      },
    });
  } else if (node.type === "schema") {
    const index = expandedKeysRef.value.findIndex((key) => key === node.key);
    if (index < 0) {
      expandedKeysRef.value.push(node.key);
    }
    addTab({
      type: "database",
      database: database.value,
      metadata: {
        database: metadata.value,
      },
      selectedSchema: node.schema.name,
    });
  }

  state.shouldRelocateTreeNode = true;
};

const handleContextMenuDropdownSelect = async (key: string) => {
  const { treeNode } = contextMenu;
  if (!treeNode) {
    return;
  }

  if (treeNode.type === "schema") {
    const { schema } = treeNode;
    if (key === "create-table") {
      state.tableNameModalContext = {
        database: database.value,
        metadata: metadata.value,
        schema,
      };
    } else if (key === "drop") {
      markEditStatus(database.value, metadataForSchema(schema), "dropped");
    } else if (key === "restore") {
      removeEditStatus(
        database.value,
        metadataForSchema(schema),
        /* recursive */ false
      );
    }
  } else if (treeNode.type === "table") {
    const { schema, table } = treeNode;
    if (key === "rename") {
      state.tableNameModalContext = {
        database: database.value,
        metadata: metadata.value,
        schema,
        table,
      };
    } else if (key === "drop") {
      // We don't physically remove it, mark it as 'dropped' instead
      // If it a 'created' table, it will remains till the page is refreshed.
      markEditStatus(
        database.value,
        metadataForTable(schema, table),
        "dropped"
      );
    } else if (key === "restore") {
      removeEditStatus(
        database.value,
        metadataForTable(schema, table),
        /* recursive */ false
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
</script>

<style lang="postcss" scoped>
.schema-designer-database-tree :deep(.n-tree-node-wrapper) {
  @apply !p-0;
}
.schema-designer-database-tree :deep(.n-tree-node-content) {
  @apply !pl-2 text-sm;
}
.schema-designer-database-tree :deep(.n-tree-node-indent) {
  width: 0.25rem;
}
.schema-designer-database-tree :deep(.n-tree-node-content__suffix) {
  @apply rounded-sm !hidden hover:opacity-80;
}
.schema-designer-database-tree
  :deep(.n-tree-node-wrapper:hover .n-tree-node-content__suffix) {
  @apply !flex;
}
.schema-designer-database-tree
  :deep(.n-tree-node-wrapper
    .n-tree-node--selected
    .n-tree-node-content__suffix) {
  @apply !flex;
}
.schema-designer-database-tree :deep(.n-tree-node-switcher) {
  @apply px-0 !w-4 !h-7;
}
</style>
