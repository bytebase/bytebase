<template>
  <div
    class="gap-y-1 h-full flex flex-col items-stretch relative overflow-hidden"
  >
    <div class="px-1 flex flex-row gap-1">
      <div class="flex-1 overflow-hidden">
        <SearchBox
          ref="searchBoxRef"
          v-model:value="searchPattern"
          :disabled="!currentTab"
          size="small"
          style="width: 100%; max-width: 100%"
        />
      </div>
      <div class="shrink-0 flex items-center">
        <SyncSchemaButton size="small" />
      </div>
    </div>

    <div
      ref="treeContainerElRef"
      class="schema-tree flex-1 px-1 pb-1 text-sm overflow-hidden select-none"
      :data-height="treeContainerHeight"
    >
      <MaskSpinner v-if="isFetchingMetadata" />

      <!-- Adaptive UI: Use tree for small DBs, flat list for large DBs -->
      <template v-else-if="metadata">
        <!-- For large databases (>1000 tables): Use flat list -->
        <FlatTableList
          v-if="totalTableCount > 1000"
          :metadata="metadata"
          :search="searchPattern"
          :database="database.name"
          @select="handleFlatListSelect"
          @select-all="handleFlatListSelectAll"
          @contextmenu="handleFlatListContextMenu"
        />

        <!-- For small databases: Use tree view -->
        <NTree
          v-else-if="tree"
          ref="treeRef"
          v-model:expanded-keys="expandedKeys"
          :selected-keys="selectedKeys"
          :block-line="true"
          :data="tree"
          :show-irrelevant-nodes="false"
          :pattern="mounted ? debouncedSearchPattern : ''"
          :virtual-scroll="true"
          :node-props="nodeProps"
          :theme-overrides="{ nodeHeight: '21px' }"
          :render-label="renderLabel"
        />
        <NEmpty v-else class="mt-16" />
      </template>
      <NEmpty v-else class="mt-16" />
    </div>

    <NDropdown
      v-if="!isFetchingMetadata"
      placement="bottom-start"
      trigger="manual"
      :x="dropdownPosition.x"
      :y="dropdownPosition.y"
      :options="dropdownOptions"
      :show="showDropdown"
      :on-clickoutside="handleDropdownClickoutside"
      @select="handleDropdownSelect"
    />

    <MaskSpinner v-if="isFetchingMetadata" class="bg-white/75!" />

    <BBModal :show="!!schemaViewer" @close="schemaViewer = undefined">
      <template v-if="schemaViewer" #title>
        <RichDatabaseName :database="database" />
      </template>
      <TableSchemaViewer
        v-if="schemaViewer"
        style="width: calc(100vw - 12rem); height: calc(100vh - 12rem)"
        v-bind="schemaViewer"
        :database="database"
      />
    </BBModal>

    <HoverPanel :offset-x="8" :offset-y="0" :margin="4" />
  </div>
</template>

<script setup lang="ts">
import {
  computedAsync,
  refDebounced,
  useElementSize,
  useEventListener,
  useMounted,
} from "@vueuse/core";
import { head, uniq, without } from "lodash-es";
import {
  NDropdown,
  NEmpty,
  NTree,
  type TreeInst,
  type TreeOption,
} from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, h, nextTick, ref, watch, watchEffect } from "vue";
import { BBModal } from "@/bbkit";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import TableSchemaViewer from "@/components/TableSchemaViewer.vue";
import { RichDatabaseName, SearchBox } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { isValidDatabaseName } from "@/types";
import type { TableMetadata } from "@/types/proto-es/v1/database_service_pb";
import {
  allEqual,
  extractDatabaseResourceName,
  findAncestor,
  isDescendantOf,
} from "@/utils";
import { useSQLEditorContext } from "../../context";
import { useCurrentTabViewStateContext } from "../../EditorPanel";
import { useActions, useDropdown } from "./actions";
import { useClickEvents } from "./click";
import FlatTableList from "./FlatTableList.vue";
import HoverPanel, { provideHoverStateContext } from "./HoverPanel";
import SyncSchemaButton from "./SyncSchemaButton.vue";
import { Label } from "./TreeNode";
import {
  buildDatabaseSchemaTree,
  ExpandableNodeTypes,
  type NodeTarget,
  type TreeNode,
} from "./tree";

const mounted = useMounted();
const searchBoxRef = ref<InstanceType<typeof SearchBox>>();
const treeRef = ref<TreeInst>();
const treeContainerElRef = ref<HTMLElement>();
const { height: treeContainerHeight } = useElementSize(
  treeContainerElRef,
  undefined,
  {
    box: "content-box",
  }
);

const { viewState: panelViewState } = useCurrentTabViewStateContext();
const { schemaViewer } = useSQLEditorContext();
const {
  show: showDropdown,
  context: dropdownContext,
  position: dropdownPosition,
  options: dropdownOptions,
  handleSelect: handleDropdownSelect,
  handleClickoutside: handleDropdownClickoutside,
} = useDropdown();
const { selectAllFromTableOrView } = useActions();
const { currentTab } = storeToRefs(useSQLEditorTabStore());
const { database } = useConnectionOfCurrentSQLEditorTab();

// Simple search pattern that resets when tab changes
const searchPattern = ref("");
const debouncedSearchPattern = refDebounced(searchPattern, 200);

// Reset search when tab changes
watch(
  () => currentTab.value?.id,
  () => {
    searchPattern.value = "";
  }
);
const isFetchingMetadata = ref(false);
const totalTableCount = ref(0);
const metadata = computedAsync(
  async () => {
    const db = database.value;
    if (!isValidDatabaseName(db.name)) return null;
    const metadata = await useDBSchemaV1Store().getOrFetchDatabaseMetadata({
      database: db.name,
    });
    totalTableCount.value = metadata.schemas.reduce(
      (sum, schema) => sum + (schema.tables?.length || 0),
      0
    );
    return metadata;
  },
  /* default */ null,
  {
    evaluating: isFetchingMetadata,
  }
);
const { events: nodeClickEvents, handleClick: handleNodeClick } =
  useClickEvents();
const {
  state: hoverState,
  position: hoverPosition,
  update: updateHoverState,
} = provideHoverStateContext();
const tree = ref<TreeNode[]>();

const expandedKeys = computed({
  get() {
    const treeState = currentTab.value?.treeState;
    if (!treeState) return [];
    if (treeState.database !== database.value.name) return [];
    return treeState.keys;
  },
  set(keys) {
    if (!currentTab.value) return;
    const treeState = currentTab.value.treeState;
    if (treeState.database !== database.value.name) return;
    treeState.keys = keys;
  },
});

const upsertExpandedKeys = (keys: string[]) => {
  const curr = expandedKeys.value;
  if (!curr) return;
  expandedKeys.value = uniq([...curr, ...keys]);
};
const removeExpandedKeys = (keys: string[]) => {
  const curr = expandedKeys.value;
  if (!curr) return;
  expandedKeys.value = without(curr, ...keys);
};

const defaultExpandedKeys = () => {
  if (!tree.value) return [];
  const keys: string[] = [];
  const collect = (node: TreeNode) => {
    keys.push(node.key);
    node.children?.forEach((child) => walk(child as TreeNode));
  };
  const walk = (node: TreeNode) => {
    const { type } = node.meta;
    if (type === "database" || type === "schema") {
      collect(node);
    } else if (type === "expandable-text") {
      const { mockType } = node.meta.target as NodeTarget<"expandable-text">;
      if (mockType === "table" || mockType === "view") {
        collect(node);
      }
    }
  };
  walk(tree.value[0]);
  return keys;
};
const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as TreeNode;
  return {
    onclick(e: MouseEvent) {
      if (node.disabled) return;

      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        handleNodeClick(node);
      }
    },
    oncontextmenu(e: MouseEvent) {
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
      const { type } = node.meta;
      if (
        type === "table" ||
        type === "external-table" ||
        type === "column" ||
        type === "view" ||
        type === "partition-table"
      ) {
        const target = node.meta.target as NodeTarget<
          "table" | "external-table" | "column" | "view"
        >;
        // Use a small delay even when transitioning between nodes
        // to prevent excessive re-renders when moving quickly
        const delay = hoverState.value ? 150 : undefined;
        updateHoverState(target, "before", delay);

        nextTick().then(() => {
          // Find the node element and put the database panel to the bottom
          // of the node, near the cursor
          const wrapper = findAncestor(e.target as HTMLElement, ".n-tree-node");
          if (!wrapper) {
            // Clear hover state if we can't find the wrapper
            updateHoverState(undefined, "after", 150);
            return;
          }
          const bounding = wrapper.getBoundingClientRect();
          hoverPosition.value.x = e.clientX;
          hoverPosition.value.y = bounding.bottom;
        });
      }
    },
    onmouseleave() {
      updateHoverState(undefined, "after");
    },
    // attrs below for trouble-shooting
    "data-node-meta-type": node.meta.type,
    "data-node-key": node.key,
  };
};

// dynamic render the highlight keywords
const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as TreeNode;
  const keyword = searchPattern.value;
  return h(Label, { node, keyword });
};

const selectedKeys = computed(() => {
  const db = database.value;
  if (!isValidDatabaseName(db.name)) return [];
  const keys: string[] = [];
  const tab = currentTab.value;
  if (tab && tab.connection.schema !== undefined) {
    keys.push(`${db.name}/schemas/${tab.connection.schema}`);
  }
  if (!panelViewState.value || panelViewState.value.view === "CODE") {
    return keys;
  }
  const {
    schema,
    detail: {
      table,
      column,
      view,
      procedure,
      func,
      sequence,
      trigger,
      externalTable,
      package: pack,
      partition,
      index,
      foreignKey,
    },
  } = panelViewState.value;

  const parts = [db.name, `schemas/${schema}`];
  if (table) {
    parts.push(`tables/${table}`);
    if (column) {
      parts.push(`columns/${column}`);
    } else if (index) {
      parts.push(`indexes/${index}`);
    } else if (partition) {
      parts.push(`partitionTables/${partition}`);
    } else if (trigger) {
      parts.push(`triggers/${trigger}`);
    } else if (foreignKey) {
      parts.push(`foreignKeys/${foreignKey}`);
    }
  } else if (view) {
    parts.push(`views/${view}`);
    if (column) {
      parts.push(`columns/${column}`);
    }
  } else if (procedure) {
    parts.push(`procedures/${procedure}`);
  } else if (func) {
    parts.push(`functions/${func}`);
  } else if (sequence) {
    parts.push(`sequences/${sequence}`);
  } else if (externalTable) {
    parts.push(`externalTables/${externalTable}`);
  } else if (pack) {
    parts.push(`packages/${pack}`);
  }

  return [parts.join("/")];
});
watchEffect(() => {
  const key = head(selectedKeys.value);
  if (key) {
    treeRef.value?.scrollTo({ key });
  }
});

// bottom-up recursively
const expandNode = (node: TreeNode) => {
  const keysToExpand: string[] = [];
  if (ExpandableNodeTypes.includes(node.meta.type)) {
    keysToExpand.push(node.key);
  }
  upsertExpandedKeys(keysToExpand);
};
const collapseNode = (node: TreeNode) => {
  removeExpandedKeys([node.key]);
};
const toggleNode = (node: TreeNode) => {
  if (expandedKeys.value.includes(node.key)) {
    collapseNode(node);
  } else {
    expandNode(node);
  }
};

const singleClick = (node: TreeNode) => {
  toggleNode(node);
  if (node.meta.type === "schema") {
    const tab = currentTab.value;
    if (tab) {
      tab.connection.schema = (node.meta.target as NodeTarget<"schema">).schema;
      return;
    }
  }
};
useEmitteryEventListener(nodeClickEvents, "single-click", ({ node }) => {
  singleClick(node);
});
useEmitteryEventListener(nodeClickEvents, "double-click", ({ node }) => {
  if (node.meta.type === "table" || node.meta.type === "view") {
    selectAllFromTableOrView(node);
  } else if (
    node.meta.type === "expandable-text" ||
    node.meta.type === "schema"
  ) {
    toggleNode(node);
  } else {
    singleClick(node);
  }
});

// Watch only the properties we actually need, not the entire tab object
// This prevents re-renders on every keystroke in the editor
watch(
  [
    isFetchingMetadata,
    metadata,
    () => currentTab.value?.id,
    () => currentTab.value?.connection.database,
    () => currentTab.value?.treeState.database,
  ],
  ([isFetchingMetadata, metadata, _, connectionDb, treeStateDb]) => {
    const cleanup = () => {
      tree.value = undefined;
    };

    const tab = currentTab.value;
    if (isFetchingMetadata || !metadata || !tab) {
      return cleanup();
    }
    if (
      !allEqual(
        extractDatabaseResourceName(metadata.name).database,
        connectionDb,
        database.value.name
      )
    ) {
      return cleanup();
    }

    if (totalTableCount.value <= 1000) {
      requestAnimationFrame(() => {
        tree.value = buildDatabaseSchemaTree(database.value, metadata);
        if (treeStateDb !== connectionDb && connectionDb) {
          // Set initial tree state for the tab when it firstly opens or its
          // connection has been changed
          tab.treeState.database = connectionDb;
          tab.treeState.keys = defaultExpandedKeys();
        }
      });
    }
  },
  {
    immediate: true,
  }
);

// Handlers for flat list events
const handleFlatListSelect = (table: { schema?: string }) => {
  // Update connection schema if needed
  const tab = currentTab.value;
  if (tab && table.schema) {
    tab.connection.schema = table.schema;
  }
};

const handleFlatListSelectAll = (table: {
  key: string;
  schema: string;
  metadata: TableMetadata;
}) => {
  // Execute SELECT * for the table
  if (table.metadata) {
    selectAllFromTableOrView({
      key: table.key,
      meta: {
        type: "table",
        target: {
          database: database.value.name,
          schema: table.schema,
          table: table.metadata.name,
        },
      },
    });
  }
};

const handleFlatListContextMenu = (
  event: MouseEvent,
  table: {
    key: string;
    schema: string;
    metadata: TableMetadata;
  }
) => {
  dropdownContext.value = {
    key: table.key,
    meta: {
      type: "table",
      target: {
        database: database.value.name,
        schema: table.schema,
        table: table.metadata.name,
      },
    },
  };

  nextTick().then(() => {
    showDropdown.value = true;
    dropdownPosition.value.x = event.clientX;
    dropdownPosition.value.y = event.clientY;
  });
};

useEventListener(treeContainerElRef, "keydown", () => {
  searchBoxRef.value?.inputRef?.focus();
});
</script>

<style lang="postcss" scoped>
.schema-tree :deep(.n-tree .v-vl) {
  --n-node-content-height: 21px !important;
}
.schema-tree :deep(.n-tree-node-content) {
  padding-left: 0 !important;
  padding-right: 0 !important;
  font-size: 0.875rem;
  line-height: 1.25rem;
}
.schema-tree :deep(.n-tree-node-wrapper) {
  padding: 0;
}
.schema-tree :deep(.n-tree-node-indent) {
  width: 10px;
  height: 10px;
}
.schema-tree :deep(.n-tree-node-switcher) {
  width: 20px !important;
  height: 20px !important;
  align-self: center;
}
.schema-tree :deep(.n-tree-node-switcher--hide) {
  width: 0 !important;
}
.schema-tree :deep(.n-tree-node-content__prefix) {
  flex-shrink: 0;
  margin-right: 0.25rem !important;
}
.schema-tree.project
  :deep(.n-tree-node[data-node-type="project"] .n-tree-node-content__prefix) {
  display: none;
}
.schema-tree :deep(.n-tree-node-content__text) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 0.25rem;
}
.schema-tree :deep(.n-tree-node--pending) {
  background-color: transparent !important;
}
.schema-tree :deep(.n-tree-node--pending:hover) {
  background-color: var(--n-node-color-hover) !important;
}
.schema-tree :deep(.n-tree-node--selected),
.schema-tree :deep(.n-tree-node--selected:hover) {
  background-color: var(--n-node-color-active) !important;
  font-weight: 500;
}
</style>
