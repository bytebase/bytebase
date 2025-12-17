<template>
  <div
    class="w-full h-full pl-1 pr-2 flex flex-col gap-y-2 relative overflow-y-hidden"
  >
    <div class="w-full sticky top-0 pt-2">
      <SearchBox
        :size="'small'"
        v-model:value="searchPattern"
        style="max-width: 100%"
        :placeholder="$t('schema-editor.search-database-and-table')"
      />
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
        :pattern="loading ? undefined : debouncedSearchPattern"
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
        :show="contextMenu.show"
        :options="contextMenuOptions"
        :x="contextMenu.clientX"
        :y="contextMenu.clientY"
        to="body"
        @select="handleContextMenuSelect"
        @clickoutside="handleContextMenuClickoutside"
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

  <ProcedureNameModal
    v-if="state.procedureNameModalContext !== undefined"
    :database="state.procedureNameModalContext.db"
    :metadata="state.procedureNameModalContext.database"
    :schema="state.procedureNameModalContext.schema"
    :procedure="state.procedureNameModalContext.procedure"
    @close="state.procedureNameModalContext = undefined"
  />

  <ViewNameModal
    v-if="state.viewNameModalContext !== undefined"
    :database="state.viewNameModalContext.db"
    :metadata="state.viewNameModalContext.database"
    :schema="state.viewNameModalContext.schema"
    :view="state.viewNameModalContext.view"
    @close="state.viewNameModalContext = undefined"
  />

  <FunctionNameModal
    v-if="state.functionNameModalContext !== undefined"
    :database="state.functionNameModalContext.db"
    :metadata="state.functionNameModalContext.database"
    :schema="state.functionNameModalContext.schema"
    :func="state.functionNameModalContext.function"
    @close="state.functionNameModalContext = undefined"
  />
</template>

<script lang="ts" setup>
import { refDebounced, useElementSize } from "@vueuse/core";
import { MD5 } from "crypto-js";
import { cloneDeep, debounce, escape, head } from "lodash-es";
import { CopyIcon, MoreHorizontalIcon } from "lucide-vue-next";
import type { TreeOption } from "naive-ui";
import { NDropdown, NPerformantEllipsis, NTree } from "naive-ui";
import {
  computed,
  h,
  nextTick,
  onMounted,
  reactive,
  ref,
  type VNode,
  watch,
} from "vue";
import { useI18n } from "vue-i18n";
import { SearchBox } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  FunctionMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  getFixedPrimaryKey,
  getHighlightHTMLByKeyWords,
  isDescendantOf,
} from "@/utils";
import { useSchemaEditorContext } from "../context";
import { keyForResource, keyForResourceName } from "../context/common";
import FunctionNameModal from "../Modals/FunctionNameModal.vue";
import ProcedureNameModal from "../Modals/ProcedureNameModal.vue";
import SchemaNameModal from "../Modals/SchemaNameModal.vue";
import TableNameModal from "../Modals/TableNameModal.vue";
import ViewNameModal from "../Modals/ViewNameModal.vue";
import { engineSupportsMultiSchema } from "../spec";
import type {
  TreeNode,
  TreeNodeForGroup,
  TreeNodeForInstance,
  TreeNodeForSchema,
  TreeNodeForTable,
} from "./common";
import { useBuildTree } from "./common";
import { useContextMenu } from "./context-menu";
import NodePrefix from "./NodePrefix.vue";

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
  procedureNameModalContext?: {
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    procedure?: ProcedureMetadata;
  };
  viewNameModalContext?: {
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    view?: ViewMetadata;
  };
  functionNameModalContext?: {
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    function?: FunctionMetadata;
  };
}

const { t } = useI18n();
const {
  events,
  targets,
  readonly,
  currentTab,
  addTab,
  markEditStatus,
  removeEditStatus,
  getSchemaStatus,
  getTableStatus,
  getColumnStatus,
  getViewStatus,
  getProcedureStatus,
  getFunctionStatus,
  queuePendingScrollToTable,
  queuePendingScrollToColumn,
} = useSchemaEditorContext();
const buildTree = useBuildTree();
const state = reactive<LocalState>({
  shouldRelocateTreeNode: false,
});

const dbSchemaStore = useDBSchemaV1Store();
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
const debouncedSearchPattern = refDebounced(searchPattern, 200);
const loading = ref(false);
const expandedKeysRef = ref<string[]>([]);
const selectedKeysRef = ref<string[]>([]);
const treeDataRef = ref<TreeNode[]>([]);
const treeNodeMap = new Map<string, TreeNode>();

const handleSearch = debounce(
  async (search: string) => {
    if (loading.value) return;
    loading.value = true;

    if (search.trim() !== "") {
      const responses = await Promise.all(
        targets.value.map(async (target) => {
          const metadata = await dbSchemaStore.getOrFetchDatabaseMetadata({
            database: target.database.name,
            skipCache: false,
            limit: 100,
            filter: `table.matches("${search.trim()}")`,
          });
          return metadata;
        })
      );
      await events.emit("merge-metadata", responses);
      buildDatabaseTreeData(false);
    }

    loading.value = false;
  },
  500,
  { trailing: true }
);

watch(
  () => searchPattern.value,
  (search) => handleSearch(search)
);

const {
  menu: contextMenu,
  options: contextMenuOptions,
  events: contextMenuEvents,
  handleShow: handleContextMenuShow,
  handleSelect: handleContextMenuSelect,
  handleClickOutside: handleContextMenuClickoutside,
} = useContextMenu();

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
  if (node.type === "group") {
    upsertExpandedKeys(node.key);
    expandNodeRecursively(node.parent);
  }
  if (node.type === "table") {
    upsertExpandedKeys(node.key);
    expandNodeRecursively(node.parent);
  }
  if (node.type === "schema") {
    upsertExpandedKeys(node.key);
    expandNodeRecursively(node.parent);
  }
  if (node.type === "database") {
    upsertExpandedKeys(node.key);
    expandNodeRecursively(node.parent);
  }
  if (node.type === "instance") {
    upsertExpandedKeys(node.key);
  }
};
const findFirstTableGroupNode = (
  nodeList: TreeNodeForSchema[] | TreeNodeForInstance[] | TreeNodeForGroup[]
): TreeNodeForGroup<"table"> | undefined => {
  const first = nodeList[0];
  if (!first) {
    return undefined;
  }
  if (first.type === "group" && first.group === "table") {
    return first as TreeNodeForGroup<"table">;
  }
  if (first.type === "instance") {
    const firstDatabaseNode = head(first.children);
    return findFirstTableGroupNode(firstDatabaseNode?.children ?? []);
  }
  if (first.type === "schema") {
    return findFirstTableGroupNode(first.children);
  }
  return undefined;
};

const buildDatabaseTreeData = (openFirstChild: boolean) => {
  const treeNodeList = buildTree(targets.value, treeNodeMap, {
    byInstance: true,
  });
  treeDataRef.value = treeNodeList;

  if (openFirstChild) {
    const firstTableGroupNode = findFirstTableGroupNode(treeNodeList);
    if (firstTableGroupNode) {
      nextTick(() => {
        // Auto expand the first tree node.
        openTabForTreeNode(firstTableGroupNode);
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
    return keyForResourceName(
      {
        database: tab.database.name,
        schema: tab.selectedSchema,
      },
      "table-group"
    );
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
      if (schema !== undefined) {
        const key = keyForResourceName(
          { database: database.name, schema },
          "table-group"
        );
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
    } else if (tab.type === "procedure") {
      const {
        database,
        metadata: { schema, procedure },
      } = tab;
      const schemaKey = keyForResource(database, { schema });
      const schemaNode = treeNodeMap.get(schemaKey);
      if (schemaNode) {
        expandNodeRecursively(schemaNode);
      }
      const procedureKey = keyForResource(database, { schema, procedure });
      selectedKeysRef.value = [procedureKey];
    } else if (tab.type === "function") {
      const {
        database,
        metadata: { schema, function: func },
      } = tab;
      const schemaKey = keyForResource(database, { schema });
      const schemaNode = treeNodeMap.get(schemaKey);
      if (schemaNode) {
        expandNodeRecursively(schemaNode);
      }
      const procedureKey = keyForResource(database, { schema, function: func });
      selectedKeysRef.value = [procedureKey];
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
  }
  if (node.type === "table") {
    expandNodeRecursively(node);
    addTab({
      type: "table",
      database: node.db,
      metadata: node.metadata,
    });
  }
  if (
    node.type === "schema" ||
    (node.type === "group" && node.group === "table")
  ) {
    expandNodeRecursively(node);
    addTab({
      type: "database",
      database: node.db,
      metadata: node.metadata,
      selectedSchema: node.metadata.schema.name,
    });
  }
  if (node.type === "view") {
    expandNodeRecursively(node);
    addTab({
      type: "view",
      database: node.db,
      metadata: node.metadata,
    });
  }
  if (node.type === "procedure") {
    expandNodeRecursively(node);
    addTab({
      type: "procedure",
      database: node.db,
      metadata: node.metadata,
    });
  }
  if (node.type === "function") {
    expandNodeRecursively(node);
    addTab({
      type: "function",
      database: node.db,
      metadata: node.metadata,
    });
  }

  state.shouldRelocateTreeNode = true;
};

onMounted(() => {
  buildDatabaseTreeData(/* openFirstChild */ true);
});

// Render prefix icons before label text.
const renderPrefix = ({ option }: { option: TreeOption }) => {
  const node = option as TreeNode;
  return h(NodePrefix, { node });
};

// Render label text.
const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as TreeNode;
  const additionalClassList: string[] = ["select-none"];
  let label = node.label;

  if (node.type === "placeholder") {
    return h(
      "span",
      { class: "text-control-placeholder italic" },
      `<${t("common.empty")}>`
    );
  }

  if (node.type === "schema") {
    const { db, metadata } = node;
    additionalClassList.push(getSchemaStatus(db, metadata));
    label = metadata.schema.name;
  }
  if (node.type === "group") {
    label = `(${node.group} - group)`;
    if (node.group === "table") {
      return t("schema-editor.tables");
    }
    if (node.group === "view") {
      return t("schema-editor.views");
    }
    if (node.group === "procedure") {
      return t("schema-editor.procedures");
    }
    if (node.group === "function") {
      return t("schema-editor.functions");
    }
  }
  if (node.type === "table") {
    const { db, metadata } = node;
    additionalClassList.push(getTableStatus(db, metadata));
    label = metadata.table.name;
  }
  if (node.type === "column") {
    const { db, metadata } = node;
    additionalClassList.push(getColumnStatus(db, metadata));
    const { name } = metadata.column;
    if (name) {
      label = name;
    } else {
      label = `<${t("common.untitled")}>`;
      additionalClassList.push("text-control-placeholder italic");
    }
  }
  if (node.type === "view") {
    const { db, metadata } = node;
    additionalClassList.push(getViewStatus(db, metadata));
    label = metadata.view.name;
  }
  if (node.type === "procedure") {
    const { db, metadata } = node;
    additionalClassList.push(getProcedureStatus(db, metadata));
    label = metadata.procedure.name;
  }
  if (node.type === "function") {
    const { db, metadata } = node;
    additionalClassList.push(getFunctionStatus(db, metadata));
    label = metadata.function.name;
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
  const node = option as TreeNode;
  const icons: VNode[] = [];

  const renderIcons = () => {
    if (icons.length === 0) return null;
    return h("div", { class: "inline-flex gap-1" }, icons);
  };

  if (readonly.value) {
    return renderIcons();
  }

  const menuIcon = h(MoreHorizontalIcon, {
    class: "w-4 h-auto text-gray-600",
    onClick: (e) => {
      handleContextMenuShow(e, node);
    },
  });
  if (node.type === "database") {
    const { engine } = node.db.instanceResource;
    if (engineSupportsMultiSchema(engine)) {
      icons.push(menuIcon);
    }
  }
  if (node.type === "schema") {
    icons.push(menuIcon);
  }
  if (node.type === "group") {
    icons.push(menuIcon);
  }
  if (node.type === "table") {
    const duplicateIcon = h(CopyIcon, {
      class: "w-4 h-auto text-gray-600",
      onClick: (e) => {
        e.preventDefault();
        e.stopPropagation();
        e.stopImmediatePropagation();

        handleDuplicateTable(node);
      },
    });
    icons.push(duplicateIcon);
    icons.push(menuIcon);
  }
  if (node.type === "procedure") {
    icons.push(menuIcon);
  }
  if (node.type === "function") {
    icons.push(menuIcon);
  }

  return renderIcons();
};

const handleDuplicateTable = (treeNode: TreeNodeForTable) => {
  const { db } = treeNode;
  const engine = db.instanceResource.engine;
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
  // As index names should be unique, we need to generate new names for them.
  for (const index of newTable.indexes) {
    let name = `${index.name}_${MD5(`${newTable.name}_${Date.now()}`).toString().slice(0, 6)}`;
    if (index.primary) {
      const fixedName = getFixedPrimaryKey(engine);
      // If the primary key name is fixed, use it instead.
      if (fixedName) {
        name = fixedName;
      }
    }
    index.name = name;
  }
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

// Set event handler to tree nodes.
const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as TreeNode;
  return {
    async onClick(e: MouseEvent) {
      // Check if clicked on the content part.
      // And ignore the fold/unfold arrow.
      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        state.shouldRelocateTreeNode = false;
        if (node.type === "group") {
          expandNodeRecursively(node);
          if (node.group === "table") {
            openTabForTreeNode(node);
          }
        } else if (node.type === "instance") {
          expandNodeRecursively(node);
        } else if (node.type === "database") {
          expandNodeRecursively(node);
          addTab({
            type: "database",
            database: node.db,
            metadata: node.metadata,
            selectedSchema: head(node.metadata.database.schemas)?.name,
          });
        } else {
          openTabForTreeNode(node);
        }
      } else {
        nextTick(() => {
          selectedKeysRef.value = [];
        });
      }
    },
    oncontextmenu(e: MouseEvent) {
      handleContextMenuShow(e, node);
    },
  };
};

const emit = defineEmits(["update-is-editing"]);

// Handle context menu actions
useEmitteryEventListener(contextMenuEvents, "show", ({ node }) => {
  selectedKeysRef.value = [node.key];
});
useEmitteryEventListener(contextMenuEvents, "hide", () => {
  selectedKeysRef.value = [];
});

useEmitteryEventListener(contextMenuEvents, "create-schema", (node) => {
  state.schemaNameModalContext = {
    db: node.db,
    database: node.metadata.database,
    schema: undefined,
  };
  emit("update-is-editing", true);
});
useEmitteryEventListener(contextMenuEvents, "drop-schema", (node) => {
  markEditStatus(node.db, node.metadata, "dropped");
  emit("update-is-editing", true);
});
useEmitteryEventListener(contextMenuEvents, "restore-schema", (node) => {
  removeEditStatus(node.db, node.metadata, /* recursive */ false);
  emit("update-is-editing", true);
});

useEmitteryEventListener(contextMenuEvents, "create-table", (node) => {
  state.tableNameModalContext = {
    db: node.db,
    database: node.metadata.database,
    schema: node.metadata.schema,
    table: undefined,
  };
  emit("update-is-editing", true);
});
useEmitteryEventListener(contextMenuEvents, "drop-table", (node) => {
  markEditStatus(node.db, node.metadata, "dropped");
  emit("update-is-editing", true);
});
useEmitteryEventListener(contextMenuEvents, "restore-table", (node) => {
  removeEditStatus(node.db, node.metadata, /* recursive */ false);
  emit("update-is-editing", true);
});
useEmitteryEventListener(contextMenuEvents, "rename-table", (node) => {
  state.tableNameModalContext = {
    db: node.db,
    database: node.metadata.database,
    schema: node.metadata.schema,
    table: node.metadata.table,
  };
  emit("update-is-editing", true);
});

useEmitteryEventListener(contextMenuEvents, "create-view", (node) => {
  state.viewNameModalContext = {
    db: node.db,
    database: node.metadata.database,
    schema: node.metadata.schema,
    view: undefined,
  };
  expandNodeRecursively(node);
  emit("update-is-editing", true);
});
useEmitteryEventListener(contextMenuEvents, "drop-view", (node) => {
  markEditStatus(node.db, node.metadata, "dropped");
  emit("update-is-editing", true);
});
useEmitteryEventListener(contextMenuEvents, "restore-view", (node) => {
  removeEditStatus(node.db, node.metadata, /* recursive */ false);
  emit("update-is-editing", true);
});

useEmitteryEventListener(contextMenuEvents, "create-procedure", (node) => {
  state.procedureNameModalContext = {
    db: node.db,
    database: node.metadata.database,
    schema: node.metadata.schema,
    procedure: undefined,
  };
  expandNodeRecursively(node);
  emit("update-is-editing", true);
});
useEmitteryEventListener(contextMenuEvents, "drop-procedure", (node) => {
  markEditStatus(node.db, node.metadata, "dropped");
  emit("update-is-editing", true);
});
useEmitteryEventListener(contextMenuEvents, "restore-procedure", (node) => {
  removeEditStatus(node.db, node.metadata, /* recursive */ false);
  emit("update-is-editing", true);
});

useEmitteryEventListener(contextMenuEvents, "create-function", (node) => {
  state.functionNameModalContext = {
    db: node.db,
    database: node.metadata.database,
    schema: node.metadata.schema,
    function: undefined,
  };
  expandNodeRecursively(node);
  emit("update-is-editing", true);
});
useEmitteryEventListener(contextMenuEvents, "drop-function", (node) => {
  markEditStatus(node.db, node.metadata, "dropped");
  emit("update-is-editing", true);
});
useEmitteryEventListener(contextMenuEvents, "restore-function", (node) => {
  removeEditStatus(node.db, node.metadata, /* recursive */ false);
  emit("update-is-editing", true);
});

const handleExpandedKeysChange = (expandedKeys: string[]) => {
  expandedKeysRef.value = expandedKeys;
};
</script>

<style lang="postcss" scoped>
.schema-editor-tree :deep(.n-tree-node-wrapper) {
  padding: 0 !important;
}
.schema-editor-tree :deep(.n-tree-node-content) {
  padding-left: 0.5rem !important;
  font-size: 0.875rem;
  line-height: 1.25rem;
}
.schema-editor-tree :deep(.n-tree-node-indent) {
  width: 0.25rem;
}
.schema-editor-tree :deep(.n-tree-node-content__prefix) {
  flex-shrink: 0;
  margin-right: 0.25rem !important;
}
.schema-editor-tree
  :deep(.n-tree-node-wrapper:hover .n-tree-node-content__suffix) {
  display: flex !important;
}
.schema-editor-tree
  :deep(
    .n-tree-node-wrapper .n-tree-node--selected .n-tree-node-content__suffix
  ) {
  display: flex !important;
}

.schema-editor-tree :deep(.n-tree-node-content__text) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 0.25rem;
}
.schema-editor-tree :deep(.n-tree-node-switcher) {
  padding-left: 0;
  padding-right: 0;
  width: 1rem !important;
  height: 1.75rem !important;
}
.schema-editor-tree:not(.disable-diff-coloring)
  :deep(.n-tree-node-content .created) {
  color: var(--color-green-700);
}
.schema-editor-tree:not(.disable-diff-coloring)
  :deep(.n-tree-node-content .dropped) {
  color: var(--color-red-700);
  text-decoration-line: line-through;
}
.schema-editor-tree:not(.disable-diff-coloring)
  :deep(.n-tree-node-content .updated) {
  color: var(--color-yellow-700);
}
</style>
