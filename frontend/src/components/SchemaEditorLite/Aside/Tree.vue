<template>
  <div
    class="w-full h-full pl-1 pr-2 flex flex-col gap-y-2 relative overflow-y-hidden"
  >
    <div class="w-full sticky top-0 pt-2 h-12 bg-white z-10">
      <NInput
        v-model:value="searchPattern"
        :placeholder="
          resourceType === 'branch'
            ? $t('schema-designer.search-tables')
            : $t('schema-editor.search-database-and-table')
        "
      >
        <template #prefix>
          <heroicons-outline:search class="w-4 h-auto text-gray-300" />
        </template>
      </NInput>
    </div>
    <div
      ref="treeContainerElRef"
      class="schema-editor-tree flex-1 pb-1 text-sm overflow-hidden select-none"
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
    :database="state.schemaNameModalContext.db"
    :metadata="state.schemaNameModalContext.database"
    @close="state.schemaNameModalContext = undefined"
  />

  <TableNameModal
    v-if="state.tableNameModalContext !== undefined"
    :database="state.tableNameModalContext.db"
    :metadata="state.tableNameModalContext.database"
    :schema="state.tableNameModalContext.schema"
    :table="state.tableNameModalContext.table"
    @close="state.tableNameModalContext = undefined"
  />
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { cloneDeep, debounce, escape, head } from "lodash-es";
import {
  TreeOption,
  NInput,
  NDropdown,
  NTree,
  DropdownOption,
  NPerformantEllipsis,
} from "naive-ui";
import {
  computed,
  onMounted,
  watch,
  ref,
  h,
  reactive,
  nextTick,
  VNodeChild,
} from "vue";
import { useI18n } from "vue-i18n";
import DuplicateIcon from "~icons/heroicons-outline/document-duplicate";
import EllipsisIcon from "~icons/heroicons-solid/ellipsis-horizontal";
import { DatabaseIcon, SchemaIcon, TableIcon } from "@/components/Icon";
import { InstanceV1EngineIcon } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
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
import { engineHasSchema } from "../engine-specs";
import NodeCheckbox from "./NodeCheckbox";
import {
  TreeNode,
  TreeNodeForInstance,
  TreeNodeForSchema,
  TreeNodeForTable,
  buildTree,
} from "./common";

interface TreeContextMenu {
  showDropdown: boolean;
  clientX: number;
  clientY: number;
  treeNode?: TreeNode;
}
interface LocalState {
  shouldRelocateTreeNode: boolean;
  schemaNameModalContext?: {
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema?: SchemaMetadata;
  };
  tableNameModalContext?: {
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table?: TableMetadata;
  };
}

const { t } = useI18n();
const {
  events,
  resourceType,
  targets,
  readonly,
  currentTab,
  addTab,
  markEditStatus,
  removeEditStatus,
  getSchemaStatus,
  getTableStatus,
  getColumnStatus,
  selectionEnabled,
  upsertTableConfig,
  queuePendingScrollToTable,
  queuePendingScrollToColumn,
} = useSchemaEditorContext();
const state = reactive<LocalState>({
  shouldRelocateTreeNode: false,
});

const contextMenu = reactive<TreeContextMenu>({
  showDropdown: false,
  clientX: 0,
  clientY: 0,
  treeNode: undefined,
});
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
const treeNodeMap = new Map<string, TreeNode>();

const contextMenuOptions = computed(() => {
  const treeNode = contextMenu.treeNode;
  if (!treeNode) return [];
  if (treeNode.type === "instance") return [];
  const { engine } = treeNode.db.instanceEntity;
  if (treeNode.type === "database") {
    const options: DropdownOption[] = [];
    if (engineHasSchema(engine)) {
      options.push({
        key: "create-schema",
        label: t("schema-editor.actions.create-schema"),
      });
    } else {
      options.push({
        key: "create-table",
        label: t("schema-editor.actions.create-table"),
      });
    }
    return options;
  } else if (treeNode.type === "schema") {
    const options: DropdownOption[] = [];
    if (engine === Engine.POSTGRES) {
      const status = getSchemaStatus(treeNode.db, treeNode.metadata);
      if (status === "dropped") {
        options.push({
          key: "restore",
          label: t("schema-editor.actions.restore"),
        });
      } else {
        options.push({
          key: "create-table",
          label: t("schema-editor.actions.create-table"),
        });
        options.push({
          key: "rename",
          label: t("schema-editor.actions.rename"),
        });
        options.push({
          key: "drop-schema",
          label: t("schema-editor.actions.drop-schema"),
        });
      }
    }
    return options;
  } else if (treeNode.type === "table") {
    const options: DropdownOption[] = [];
    const status = getTableStatus(treeNode.db, treeNode.metadata);
    if (status === "dropped") {
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
      });
    }
    return options;
  }

  return [];
});

const upsertExpandedKeys = (key: string) => {
  if (expandedKeysRef.value.includes(key)) return;
  expandedKeysRef.value.push(key);
};
const expandNodeRecursively = (node: TreeNode | undefined) => {
  if (!node) return;
  if (node.type === "column") {
    // column nodes are not expandable
    expandNodeRecursively(node.parent);
  }
  if (node.type === "table") {
    const key = keyForResource(node.db, node.metadata);
    upsertExpandedKeys(key);
    expandNodeRecursively(node.parent);
  }
  if (node.type === "schema") {
    const key = keyForResource(node.db, node.metadata);
    upsertExpandedKeys(key);
    expandNodeRecursively(node.parent);
  }
  if (node.type === "database") {
    const key = node.db.name;
    upsertExpandedKeys(key);
    expandNodeRecursively(node.parent);
  }
  if (node.type === "instance") {
    const key = node.instance.name;
    upsertExpandedKeys(key);
  }
};
const findFirstSchemaNode = (
  nodeList: TreeNodeForSchema[] | TreeNodeForInstance[]
) => {
  const first = nodeList[0];
  if (!first) return undefined;
  if (first.type === "schema") {
    return first;
  }
  const firstDatabaseNode = head(first.children);
  return head(firstDatabaseNode?.children);
};
const buildDatabaseTreeData = (openFirstChild: boolean) => {
  const treeNodeList = buildTree(targets.value, treeNodeMap, {
    byInstance: true,
  });
  treeDataRef.value = treeNodeList;

  if (openFirstChild) {
    const firstSchemaNode = findFirstSchemaNode(treeNodeList);
    if (firstSchemaNode) {
      nextTick(() => {
        // Auto expand the first tree node.
        openTabForTreeNode(firstSchemaNode);
      });
    }
  }
};
const debouncedBuildDatabaseTreeData = debounce(buildDatabaseTreeData, 100);
useEmitteryEventListener(events, "rebuild-tree", (params) => {
  debouncedBuildDatabaseTreeData(params.openFirstChild);
});

const tabWatchKey = computed(() => {
  const tab = currentTab.value;
  if (!tab) return undefined;
  if (tab.type === "database") {
    return keyForResourceName(tab.database.name, tab.selectedSchema);
  }
  return keyForResource(tab.database, tab.metadata);
});
// Sync tree expandedKeys and selectedKeys with tabs
watch(tabWatchKey, () => {
  requestAnimationFrame(() => {
    const tab = currentTab.value;
    if (!tab) {
      selectedKeysRef.value = [];
      return;
    }

    if (tab.type === "database") {
      const { database, selectedSchema: schema } = tab;
      if (schema) {
        const key = keyForResourceName(database.name, schema);
        const node = treeNodeMap.get(key);
        if (node) {
          expandNodeRecursively(node);
        }
        selectedKeysRef.value = [key];
      }
    } else if (tab.type === "table") {
      const {
        database,
        metadata: { schema, table },
      } = tab;
      const schemaKey = keyForResource(database, { schema });
      const schemaNode = treeNodeMap.get(schemaKey);
      if (schemaNode) {
        expandNodeRecursively(schemaNode);
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
});

const openTabForTreeNode = (node: TreeNode) => {
  state.shouldRelocateTreeNode = false;

  if (node.type === "column") {
    openTabForTreeNode(node.parent);
    queuePendingScrollToColumn({
      db: node.db,
      metadata: node.metadata,
    });
    return;
  } else if (node.type === "table") {
    expandNodeRecursively(node);
    addTab({
      type: "table",
      database: node.db,
      metadata: node.metadata,
    });
  } else if (node.type === "schema") {
    expandNodeRecursively(node);
    addTab({
      type: "database",
      database: node.db,
      metadata: node.metadata,
      selectedSchema: node.metadata.schema.name,
    });
  }

  state.shouldRelocateTreeNode = true;
};

onMounted(() => {
  buildDatabaseTreeData(/* openFirstChild */ true);
});

// Render prefix icons before label text.
const renderPrefix = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;

  if (treeNode.type === "instance") {
    const { instance } = treeNode;
    const children = [
      h(InstanceV1EngineIcon, {
        instance,
      }),
      h(
        "span",
        {
          class: "text-gray-500 text-sm",
        },
        `(${instance.environmentEntity.title})`
      ),
    ];

    return h("span", { class: "flex items-center gap-x-1" }, children);
  } else if (treeNode.type === "database") {
    return h("span", { class: "flex items-center gap-x-1" }, [
      h(DatabaseIcon, {
        class: "w-4 h-auto text-gray-400",
      }),
      h(
        "span",
        {
          class: "text-gray-500 text-sm",
        },
        `(${treeNode.db.effectiveEnvironmentEntity.title})`
      ),
    ]);
  }

  const children: VNodeChild[] = [];
  if (selectionEnabled.value) {
    children.push(
      h(NodeCheckbox, {
        node: treeNode,
      })
    );
  }
  if (treeNode.type === "schema") {
    children.push(
      h(SchemaIcon, {
        class: "w-4 h-auto text-gray-400",
      })
    );
  } else if (treeNode.type === "table") {
    children.push(
      h(TableIcon, {
        class: "w-4 h-auto text-gray-400",
      })
    );
  }
  if (children.length > 0) {
    return h("div", { class: "flex flex-row items-center gap-x-1" }, children);
  }
  return null;
};

// Render label text.
const renderLabel = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  const additionalClassList: string[] = ["select-none"];
  let label = treeNode.label;

  if (treeNode.type === "schema") {
    const { db, metadata } = treeNode;
    additionalClassList.push(getSchemaStatus(db, metadata));

    if (db.instanceEntity.engine !== Engine.POSTGRES) {
      label = t("db.tables");
    } else {
      label = metadata.schema.name;
    }
  } else if (treeNode.type === "table") {
    const { db, metadata } = treeNode;
    additionalClassList.push(getTableStatus(db, metadata));
    label = metadata.table.name;
  } else if (treeNode.type === "column") {
    const { db, metadata } = treeNode;
    additionalClassList.push(getColumnStatus(db, metadata));
    const { name } = metadata.column;
    if (name) {
      label = name;
    } else {
      label = `<${t("common.untitled")}>`;
      additionalClassList.push("text-control-placeholder italic");
    }
  }
  return h(
    NPerformantEllipsis,
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
  const treeNode = option as TreeNode;
  const menuIcon = h(EllipsisIcon, {
    class: "w-4 h-auto text-gray-600",
    onClick: (e) => {
      handleShowDropdown(e, treeNode);
    },
  });
  if (treeNode.type === "database") {
    return menuIcon;
  } else if (treeNode.type === "schema") {
    const { engine } = treeNode.db.instanceEntity;
    if (engine === Engine.POSTGRES) {
      return menuIcon;
    }
  } else if (treeNode.type === "table") {
    const icons = [menuIcon];

    if (!readonly.value) {
      const duplicateIcon = h(DuplicateIcon, {
        class: "w-4 h-auto mr-2 text-gray-600",
        onClick: (e) => {
          e.preventDefault();
          e.stopPropagation();
          e.stopImmediatePropagation();

          handleDuplicateTable(treeNode);
        },
      });
      icons.unshift(duplicateIcon);
    }
    return icons;
  }
  return null;
};

const handleDuplicateTable = (treeNode: TreeNodeForTable) => {
  const { db } = treeNode;
  const { schema, table } = treeNode.metadata;
  const flattenTableList = treeNode.metadata.database.schemas.flatMap(
    (schema) => schema.tables
  );
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

  const matchPattern = new RegExp(
    `^${getOriginalName(table.name)}` + "(_copy[0-9]{0,}){0,1}$"
  );
  const copiedTableNames = flattenTableList
    .filter((table) => {
      return matchPattern.test(table.name);
    })
    .sort((t1, t2) => {
      return extractDuplicateNumber(t1.name) - extractDuplicateNumber(t2.name);
    });
  const targetName = copiedTableNames.slice(-1)[0]?.name ?? table.name;

  const newTable = cloneDeep(table);
  newTable.name = getDuplicateName(targetName);
  schema.tables.push(newTable);
  markEditStatus(
    db,
    {
      ...treeNode.metadata,
      table: newTable,
    },
    "created"
  );
  newTable.columns.forEach((newColumn) => {
    markEditStatus(
      db,

      {
        ...treeNode.metadata,
        table: newTable,
        column: newColumn,
      },
      "created"
    );
  });
  const tableConfig = treeNode.metadata.database.schemaConfigs
    .find((sc) => sc.name === schema.name)
    ?.tableConfigs.find((tc) => tc.name === table.name);
  if (tableConfig) {
    const tableConfigCopy = cloneDeep(tableConfig);
    tableConfigCopy.name = newTable.name;
    upsertTableConfig(
      db,
      {
        ...treeNode.metadata,
        table: newTable,
      },
      (config) => {
        Object.assign(config, tableConfigCopy);
      }
    );
  }
  addTab({
    type: "table",
    database: db,
    metadata: {
      ...treeNode.metadata,
      table: newTable,
    },
  });
  queuePendingScrollToTable({
    db,
    metadata: {
      ...treeNode.metadata,
      table: newTable,
    },
  });
  events.emit("rebuild-tree", {
    openFirstChild: false,
  });
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
const nodeProps = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  return {
    async onClick(e: MouseEvent) {
      // Check if clicked on the content part.
      // And ignore the fold/unfold arrow.
      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        state.shouldRelocateTreeNode = false;
        if (treeNode.type === "instance") {
          expandNodeRecursively(treeNode);
        } else if (treeNode.type === "database") {
          expandNodeRecursively(treeNode);
          addTab({
            type: "database",
            database: treeNode.db,
            metadata: treeNode.metadata,
            selectedSchema: head(treeNode.metadata.database.schemas)?.name,
          });
        } else {
          openTabForTreeNode(treeNode);
        }
      } else {
        nextTick(() => {
          selectedKeysRef.value = [];
        });
      }
    },
    oncontextmenu(e: MouseEvent) {
      handleShowDropdown(e, treeNode);
    },
  };
};

const handleContextMenuDropdownSelect = async (key: string) => {
  const treeNode = contextMenu.treeNode;
  if (!treeNode) return;
  if (treeNode.type === "database") {
    const engine = treeNode.db.instanceEntity.engine;
    if (key === "create-table") {
      if (!engineHasSchema(engine)) {
        const schema = head(treeNode.metadata.database.schemas);
        if (!schema) {
          return;
        }
        state.tableNameModalContext = {
          db: treeNode.db,
          database: treeNode.metadata.database,
          schema: schema,
          table: undefined,
        };
      }
    } else if (key === "create-schema") {
      if (engineHasSchema(engine)) {
        state.schemaNameModalContext = {
          db: treeNode.db,
          database: treeNode.metadata.database,
          schema: undefined,
        };
      }
    }
  } else if (treeNode.type === "schema") {
    if (key === "create-table") {
      state.tableNameModalContext = {
        db: treeNode.db,
        database: treeNode.metadata.database,
        schema: treeNode.metadata.schema,
        table: undefined,
      };
    } else if (key === "drop-schema") {
      markEditStatus(treeNode.db, treeNode.metadata, "dropped");
    } else if (key === "restore") {
      removeEditStatus(treeNode.db, treeNode.metadata, /* recursive */ false);
    }
  } else if (treeNode.type === "table") {
    if (key === "rename") {
      state.tableNameModalContext = {
        db: treeNode.db,
        database: treeNode.metadata.database,
        schema: treeNode.metadata.schema,
        table: treeNode.metadata.table,
      };
    } else if (key === "drop") {
      markEditStatus(treeNode.db, treeNode.metadata, "dropped");
    } else if (key === "restore") {
      removeEditStatus(treeNode.db, treeNode.metadata, /* recursive */ false);
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
.schema-editor-tree :deep(.n-tree-node-wrapper) {
  @apply !p-0;
}
.schema-editor-tree :deep(.n-tree-node-content) {
  @apply !pl-2 text-sm;
}
.schema-editor-tree :deep(.n-tree-node-indent) {
  width: 0.25rem;
}
.schema-editor-tree :deep(.n-tree-node-content__prefix) {
  @apply shrink-0 !mr-1;
}
.schema-editor-tree
  :deep(.n-tree-node-wrapper:hover .n-tree-node-content__suffix) {
  @apply !flex;
}
.schema-editor-tree
  :deep(.n-tree-node-wrapper
    .n-tree-node--selected
    .n-tree-node-content__suffix) {
  @apply !flex;
}

.schema-editor-tree :deep(.n-tree-node-content__text) {
  @apply truncate mr-1;
}
.schema-editor-tree :deep(.n-tree-node-switcher) {
  @apply px-0 !w-4 !h-7;
}
.schema-editor-tree :deep(.n-tree-node-content .created) {
  @apply text-green-700;
}
.schema-editor-tree :deep(.n-tree-node-content .dropped) {
  @apply text-red-700 line-through;
}
.schema-editor-tree :deep(.n-tree-node-content .updated) {
  @apply text-yellow-700;
}
</style>
