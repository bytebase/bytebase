<template>
  <div
    class="gap-y-1 h-full flex flex-col items-stretch relative overflow-hidden"
  >
    <div class="px-1 flex flex-row gap-1">
      <div class="flex-1 overflow-hidden">
        <SearchBox
          v-model:value="searchPattern"
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
      <NTree
        v-if="tree"
        ref="treeRef"
        v-model:expanded-keys="expandedKeys"
        :selected-keys="selectedKeys"
        :block-line="true"
        :data="tree"
        :show-irrelevant-nodes="false"
        :pattern="mounted ? searchPattern : ''"
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
import { computedAsync, useElementSize, useMounted } from "@vueuse/core";
import { uniq } from "lodash-es";
import { NDropdown, NEmpty, NTree, type TreeOption } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, h, nextTick, ref, watch } from "vue";
import { BBModal } from "@/bbkit";
import TableSchemaViewer from "@/components/TableSchemaViewer.vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { RichDatabaseName, SearchBox } from "@/components/v2";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { extractDatabaseResourceName, isDescendantOf } from "@/utils";
import { useSQLEditorContext } from "../../context";
import SyncSchemaButton from "./SyncSchemaButton.vue";
import { Label } from "./TreeNode";
import { useDropdown, useActions } from "./actions";
import {
  type NodeTarget,
  type TreeNode,
  buildDatabaseSchemaTree,
  ExpandableNodeTypes,
} from "./common";

const mounted = useMounted();
const treeContainerElRef = ref<HTMLElement>();
const { height: treeContainerHeight } = useElementSize(
  treeContainerElRef,
  undefined,
  {
    box: "content-box",
  }
);
const searchPattern = ref("");
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
const { connection, database } = useConnectionOfCurrentSQLEditorTab();
const isFetchingMetadata = ref(false);
const metadata = computedAsync(
  async () => {
    const db = database.value;
    if (db.uid === String(UNKNOWN_ID)) return null;
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
// const tree = computed(() => {
//   if (isFetchingMetadata.value) return null;
//   if (!metadata.value) return null;
//   return buildDatabaseSchemaTree(database.value, metadata.value);
// });
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

      // if (isDescendantOf(e.target as Element, ".n-tree-node-switcher")) {
      //   toggleExpanded(node);
      //   return;
      // }

      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        // handleClick(node);
        handleSingleClickNode(node);
      }
    },
    ondblclick(e: MouseEvent) {
      if (node.disabled) return;
      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        handleDoubleClickNode(node);
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
  if (db.uid === String(UNKNOWN_ID)) return [];
  const { schema, table } = connection.value;
  if (!table) return [];
  return [`${db.name}/schemas/${schema}/tables/${table}`];
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

const handleSingleClickNode = (node: TreeNode) => {
  expandNode(node);
  viewDetail(node);
};
const handleDoubleClickNode = (node: TreeNode) => {
  if (node.meta.type === "table" || node.meta.type === "view") {
    selectAllFromTableOrView(node);
  }
};

watch(
  [isFetchingMetadata, metadata],
  ([isFetchingMetadata, metadata]) => {
    if (isFetchingMetadata || !metadata) {
      tree.value = undefined;
      return;
    }
    tree.value = buildDatabaseSchemaTree(database.value, metadata);

    const tab = currentTab.value;
    if (tab) {
      const { database } = extractDatabaseResourceName(metadata.name);
      if (database !== tab.treeState.database) {
        tab.treeState.database = database;
        tab.treeState.keys = defaultExpandedKeys();
      }
    }
  },
  {
    immediate: true,
  }
);
</script>

<style lang="postcss" scoped>
.schema-tree :deep(.n-tree .v-vl) {
  --n-node-content-height: 21px !important;
}
.schema-tree :deep(.n-tree-node-content) {
  @apply !pl-0 text-sm;
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
