<template>
  <div class="sql-editor-tree gap-y-1 h-full flex flex-col relative">
    <div class="flex flex-row gap-x-0.5 px-1 items-center">
      <SearchBox v-model:searchPattern="searchPattern" class="flex-1" />
      <GroupingBar class="shrink-0" />
    </div>
    <div
      ref="treeContainerElRef"
      class="sql-editor-tree--tree flex-1 px-1 pb-1 text-sm overflow-hidden select-none"
      :data-height="treeContainerHeight"
    >
      <NTree
        v-if="treeStore.state === 'READY'"
        ref="treeRef"
        v-model:expanded-keys="expandedKeys"
        :block-line="true"
        :data="treeStore.tree"
        :show-irrelevant-nodes="false"
        :selected-keys="selectedKeys"
        :pattern="mounted ? searchPattern : ''"
        :expand-on-click="true"
        :node-props="nodeProps"
        :virtual-scroll="true"
        :theme-overrides="{ nodeHeight: '21px' }"
        :render-label="renderLabel"
      />
    </div>

    <NDropdown
      v-if="treeStore.state === 'READY'"
      placement="bottom-start"
      trigger="manual"
      :x="dropdownPosition.x"
      :y="dropdownPosition.y"
      :options="dropdownOptions"
      :show="showDropdown"
      :on-clickoutside="handleDropdownClickoutside"
      @select="handleDropdownSelect"
    />

    <DatabaseHoverPanel :offset-x="4" :offset-y="4" :margin="4" />

    <MaskSpinner v-if="treeStore.state !== 'READY'" class="!bg-white/75">
      <span class="text-control text-sm">{{
        $t("sql-editor.loading-databases")
      }}</span>
    </MaskSpinner>
  </div>
</template>

<script lang="ts" setup>
import { useElementSize, useMounted } from "@vueuse/core";
import { head } from "lodash-es";
import { NTree, NDropdown, type TreeOption } from "naive-ui";
import { ref, computed, nextTick, watch, h } from "vue";
import { onMounted } from "vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useDatabaseV1Store,
  useInstanceV1Store,
  useIsLoggedIn,
  useSQLEditorTabStore,
  resolveOpeningDatabaseListFromSQLEditorTabList,
  useSQLEditorTreeStore,
  idForSQLEditorTreeNodeTarget,
  useConnectionOfCurrentSQLEditorTab,
} from "@/store";
import type {
  ComposedDatabase,
  SQLEditorTreeNode,
  SQLEditorTreeNodeTarget,
  SQLEditorTreeNodeType,
} from "@/types";
import { ExpandableTreeNodeTypes, UNKNOWN_ID } from "@/types";
import { findAncestor, isDescendantOf } from "@/utils";
import { useSQLEditorContext } from "../../context";
import {
  DatabaseHoverPanel,
  provideHoverStateContext,
} from "./DatabaseHoverPanel";
import GroupingBar from "./GroupingBar";
import SearchBox from "./SearchBox/index.vue";
import useSearchHistory from "./SearchBox/useSearchHistory";
import { Label } from "./TreeNode";
import { setConnection, useDropdown } from "./actions";

const treeStore = useSQLEditorTreeStore();
const tabStore = useSQLEditorTabStore();
const databaseStore = useDatabaseV1Store();
const instanceStore = useInstanceV1Store();
const isLoggedIn = useIsLoggedIn();
const { events: editorEvents, showConnectionPanel } = useSQLEditorContext();
const searchHistory = useSearchHistory();
const {
  state: hoverState,
  position: hoverPosition,
  update: updateHoverNode,
} = provideHoverStateContext();
const {
  show: showDropdown,
  context: dropdownContext,
  position: dropdownPosition,
  options: dropdownOptions,
  handleSelect: handleDropdownSelect,
  handleClickoutside: handleDropdownClickoutside,
} = useDropdown();

const mounted = useMounted();
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

// Highlight the current tab's connection node.
const selectedKeys = computed(() => {
  const connection = tabStore.currentTab?.connection;
  if (!connection) {
    return [];
  }

  if (connection.database) {
    const database = databaseStore.getDatabaseByName(connection.database);
    const node = head(treeStore.nodesByTarget("database", database));
    if (!node) return [];
    return [node.key];
  } else if (connection.instance) {
    const instance = instanceStore.getInstanceByName(connection.instance);
    const nodes = treeStore.nodesByTarget("instance", instance);
    return nodes.map((node) => node.key);
  }
  return [];
});
const expandedKeys = ref<string[]>([]);
const upsertExpandedKeys = (key: string) => {
  if (expandedKeys.value.includes(key)) {
    return;
  }
  expandedKeys.value.push(key);
};
const expandNode = (
  node: SQLEditorTreeNode | undefined,
  keys?: Set<string>
) => {
  if (!node) {
    return;
  }
  if (ExpandableTreeNodeTypes.includes(node.meta.type)) {
    if (keys) {
      keys.add(node.key);
    } else {
      upsertExpandedKeys(node.key);
    }
  }
};
const expandNodeRecursively = (
  node: SQLEditorTreeNode | undefined,
  keys?: Set<string>
) => {
  if (!node) {
    return;
  }
  expandNode(node, keys);

  if (node.parent) {
    expandNodeRecursively(node.parent, keys);
  }
};
const expandNodesByType = <T extends SQLEditorTreeNodeType>(
  type: T,
  target: SQLEditorTreeNodeTarget<T>
) => {
  const nodes = treeStore.nodesByTarget(type, target);

  nodes.forEach((node) => {
    expandNodeRecursively(node);
  });

  return nodes;
};

// dynamic render the highlight keywords
const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as any as SQLEditorTreeNode;
  return h(Label, {
    node,
    factors: treeStore.filteredFactorList,
    keyword: searchPattern.value ?? "",
  });
};

const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as any as SQLEditorTreeNode;
  return {
    onClick(e: MouseEvent) {
      if (node.disabled) return;

      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        const { type } = node.meta;
        // Check if clicked on the content part.
        // And ignore the fold/unfold arrow.
        if (type === "database") {
          setConnection(node);
          showConnectionPanel.value = false;
        }

        // If the search pattern is not empty, append the selected database name to
        // the search history.
        if (searchPattern.value) {
          if (type === "database") {
            const database = node.meta.target as ComposedDatabase;
            searchHistory.appendSearchResult(database.name);
          }
        }
      }
    },
    onContextmenu(e: MouseEvent) {
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
      if (node.meta.type === "database") {
        if (hoverState.value) {
          updateHoverNode({ node }, "before", 0 /* overrideDelay */);
        } else {
          updateHoverNode({ node }, "before");
        }
        nextTick().then(() => {
          // Find the node element and put the database panel to the right corner
          // of the node
          const wrapper = findAncestor(e.target as HTMLElement, ".n-tree-node");
          if (!wrapper) {
            updateHoverNode(undefined, "after", 0 /* overrideDelay */);
            return;
          }
          const bounding = wrapper.getBoundingClientRect();
          hoverPosition.value.x = bounding.right;
          hoverPosition.value.y = bounding.top;
        });
      }
    },
    onmouseleave() {
      updateHoverNode(undefined, "after");
    },
    // attrs below for trouble-shooting
    "data-node-meta-type": node.meta.type,
    "data-node-meta-id": idForSQLEditorTreeNodeTarget(
      node.meta.type,
      node.meta.target
    ),
    "data-node-key": node.key,
  };
};

// Open corresponding tree node when the connection changed.
const { instance, database } = useConnectionOfCurrentSQLEditorTab();
watch(
  [isLoggedIn, instance, database, () => treeStore.state],
  ([isLoggedIn, instance, database, treeState]) => {
    if (!isLoggedIn) {
      // Don't go further and cleanup the state if we signed out.
      // treeStore.expandedKeys = [];
      expandedKeys.value = [];
      return;
    }

    if (treeState !== "READY") {
      return;
    }
    if (instance.uid !== String(UNKNOWN_ID)) {
      expandNodesByType("instance", instance);
    }
    if (database.uid !== String(UNKNOWN_ID)) {
      expandNodesByType("project", database.projectEntity);
      expandNodesByType("database", database);
    }
  },
  { immediate: true }
);

watch(
  selectedKeys,
  (keys) => {
    if (keys.length !== 1) return;
    const key = keys[0];
    nextTick(() => {
      treeRef.value?.scrollTo({ key });
    });
  },
  { immediate: true }
);

const calcDefaultExpandKeys = async () => {
  await nextTick();
  const openingDatabaseList = resolveOpeningDatabaseListFromSQLEditorTabList();
  const keys = new Set<string>();
  // Recursively expand opening databases' parent nodes
  openingDatabaseList.forEach((meta) => {
    const db = meta.target;
    const nodes = treeStore.nodesByTarget("database", db);
    nodes.forEach((node) => expandNodeRecursively(node.parent, keys));
  });
  if (keys.size === 0) {
    // Try expand till the first database node
    const dfsWalk = (node: SQLEditorTreeNode) => {
      if (node.meta.type === "database") {
        expandNodeRecursively(node.parent, keys);
        return true;
      }
      if (!node.children) return false;
      for (let i = 0; i < node.children.length; i++) {
        const child = node.children[i] as SQLEditorTreeNode;
        if (dfsWalk(child)) {
          return true;
        }
      }
      return false;
    };
    for (let i = 0; i < treeStore.tree.length; i++) {
      const node = treeStore.tree[i];
      if (dfsWalk(node)) {
        break;
      }
    }
  }
  console.log("defaultExpandKeys", Array.from(keys).join(","));
  expandedKeys.value = Array.from(keys);
};

useEmitteryEventListener(editorEvents, "tree-ready", () => {
  calcDefaultExpandKeys();
});

onMounted(calcDefaultExpandKeys);
</script>

<style lang="postcss" scoped>
.sql-editor-tree :deep(.n-tree .v-vl) {
  --n-node-content-height: 21px !important;
}
.sql-editor-tree :deep(.n-tree-node-content) {
  @apply !pl-0 text-sm;
}
.sql-editor-tree :deep(.n-tree-node-wrapper) {
  padding: 0;
}
.sql-editor-tree :deep(.n-tree-node-indent) {
  width: 0.25rem;
}
.sql-editor-tree :deep(.n-tree-node-content__prefix) {
  @apply shrink-0 !mr-1;
}
.sql-editor-tree.project
  :deep(.n-tree-node[data-node-type="project"] .n-tree-node-content__prefix) {
  @apply hidden;
}
.sql-editor-tree :deep(.n-tree-node-content__text) {
  @apply truncate mr-1;
}
.sql-editor-tree :deep(.n-tree-node--pending) {
  background-color: transparent !important;
}
.sql-editor-tree :deep(.n-tree-node--pending:hover) {
  background-color: var(--n-node-color-hover) !important;
}
.sql-editor-tree :deep(.n-tree-node--selected),
.sql-editor-tree :deep(.n-tree-node--selected:hover) {
  background-color: var(--n-node-color-active) !important;
  font-weight: 500;
}
</style>
