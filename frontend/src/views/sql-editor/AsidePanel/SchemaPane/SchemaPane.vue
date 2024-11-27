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
        <SyncSchemaButton
          :database="database"
          :metadata="metadata"
          size="small"
        />
      </div>
    </div>

    <div
      ref="treeContainerElRef"
      class="schema-tree flex-1 px-1 pb-1 text-sm overflow-hidden select-none"
      :data-height="treeContainerHeight"
    >
      <MaskSpinner v-if="isFetchingMetadata" />
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
      <NEmpty v-else class="mt-[4rem]" />
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

    <MaskSpinner v-if="isFetchingMetadata" class="!bg-white/75" />

    <BBModal :show="!!schemaViewer" @close="schemaViewer = undefined">
      <template v-if="schemaViewer" #title>
        <RichDatabaseName :database="schemaViewer.database" />
      </template>
      <TableSchemaViewer
        v-if="schemaViewer"
        style="width: calc(100vw - 12rem); height: calc(100vh - 12rem)"
        v-bind="schemaViewer"
      />
    </BBModal>
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
import { computed, h, nextTick, reactive, ref, watch } from "vue";
import { watchEffect } from "vue";
import { BBModal } from "@/bbkit";
import TableSchemaViewer from "@/components/TableSchemaViewer.vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { RichDatabaseName, SearchBox } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { isValidDatabaseName } from "@/types";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { allEqual, extractDatabaseResourceName, isDescendantOf } from "@/utils";
import { useEditorPanelContext } from "../../EditorPanel";
import { useSQLEditorContext } from "../../context";
import SyncSchemaButton from "./SyncSchemaButton.vue";
import { Label } from "./TreeNode";
import { useDropdown, useActions } from "./actions";
import {
  type NodeTarget,
  type TreeNode,
  buildDatabaseSchemaTree,
  ExpandableNodeTypes,
  useClickEvents,
} from "./common";

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
const searchPatternByTabId = reactive(new Map<string, string>());
const { viewState: panelViewState } = useEditorPanelContext();
const { schemaViewer } = useSQLEditorContext();
const {
  show: showDropdown,
  context: dropdownContext,
  position: dropdownPosition,
  options: dropdownOptions,
  handleSelect: handleDropdownSelect,
  handleClickoutside: handleDropdownClickoutside,
} = useDropdown();
const { selectAllFromTableOrView, viewDetail } = useActions();
const { currentTab } = storeToRefs(useSQLEditorTabStore());
const { database } = useConnectionOfCurrentSQLEditorTab();
const searchPattern = computed({
  get() {
    const id = currentTab.value?.id ?? "";
    return searchPatternByTabId.get(id) ?? "";
  },
  set(value) {
    const id = currentTab.value?.id ?? "";
    searchPatternByTabId.set(id, value);
  },
});
const debouncedSearchPattern = refDebounced(searchPattern, 200);
const isFetchingMetadata = ref(false);
const metadata = computedAsync(
  async () => {
    const db = database.value;
    if (!isValidDatabaseName(db.name)) return null;
    const metadata = await useDBSchemaV1Store().getOrFetchDatabaseMetadata({
      database: db.name,
      view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
    });
    return metadata;
  },
  /* default */ null,
  {
    evaluating: isFetchingMetadata,
  }
);
const { events: nodeClickEvents, handleClick: handleNodeClick } =
  useClickEvents();
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
  const node = option as any as TreeNode;
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
    // attrs below for trouble-shooting
    "data-node-meta-type": node.meta.type,
    "data-node-key": node.key,
  };
};

// dynamic render the highlight keywords
const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as any as TreeNode;
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
      dependentColumn,
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
    } else if (foreignKey) {
      parts.push(`foreignKeys/${foreignKey}`);
    }
  } else if (view) {
    parts.push(`views/${view}`);
    if (column) {
      parts.push(`columns/${column}`);
    } else if (dependentColumn) {
      parts.push(`dependentColumns/${dependentColumn}`);
    }
  } else if (procedure) {
    parts.push(`procedures/${procedure}`);
  } else if (func) {
    parts.push(`functions/${func}`);
  } else if (sequence) {
    parts.push(`sequences/${sequence}`);
  } else if (trigger) {
    parts.push(`triggers/${trigger}`);
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
  const walk = (node: TreeNode | undefined) => {
    if (!node) return;
    if (ExpandableNodeTypes.includes(node.meta.type)) {
      keysToExpand.push(node.key);
    }
    walk(node.parent);
  };
  walk(node);
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
  expandNode(node);
  if (node.meta.type === "schema") {
    const tab = currentTab.value;
    if (tab) {
      tab.connection.schema = (
        node.meta.target as NodeTarget<"schema">
      ).schema.name;
      return;
    }
  }
  viewDetail(node);
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

watch(
  [isFetchingMetadata, metadata, currentTab],
  ([isFetchingMetadata, metadata, tab]) => {
    const cleanup = () => {
      tree.value = undefined;
    };
    if (isFetchingMetadata || !metadata || !tab) {
      return cleanup();
    }
    if (
      !allEqual(
        extractDatabaseResourceName(metadata.name).database,
        tab.connection.database,
        database.value.name
      )
    ) {
      return cleanup();
    }

    tree.value = buildDatabaseSchemaTree(database.value, metadata);
    if (tab.treeState.database !== tab.connection.database) {
      // Set initial tree state for the tab when it firstly opens or its
      // connection has been changed
      tab.treeState.database = tab.connection.database;
      tab.treeState.keys = defaultExpandedKeys();
    }
  },
  {
    immediate: true,
  }
);

useEventListener(treeContainerElRef, "keydown", () => {
  searchBoxRef.value?.inputRef?.focus();
});
</script>

<style lang="postcss" scoped>
.schema-tree :deep(.n-tree .v-vl) {
  --n-node-content-height: 21px !important;
}
.schema-tree :deep(.n-tree-node-content) {
  @apply !px-0 text-sm;
}
.schema-tree :deep(.n-tree-node-wrapper) {
  padding: 0;
}
.schema-tree :deep(.n-tree-node-indent) {
  width: 20px;
  height: 20px;
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
  @apply shrink-0 !mr-1;
}
.schema-tree.project
  :deep(.n-tree-node[data-node-type="project"] .n-tree-node-content__prefix) {
  @apply hidden;
}
.schema-tree :deep(.n-tree-node-content__text) {
  @apply truncate mr-1;
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
