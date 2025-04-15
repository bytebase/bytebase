<template>
  <div class="sql-editor-tree gap-y-1 h-full flex flex-col relative">
    <div class="flex flex-row gap-x-0.5 px-1 items-center">
      <SearchBox
        :loading="editorStore.loading"
        v-model:search-pattern="searchPattern"
        class="flex-1"
      />
      <GroupingBar :disabled="editorStore.loading" class="shrink-0" />
    </div>
    <div
      v-if="hasMissingQueryDatabases"
      class="flex items-center space-x-2 px-2 py-2"
    >
      <NCheckbox
        :disabled="editorStore.loading"
        v-model:checked="showMissingQueryDatabases"
      >
        <span class="textinfolabel text-sm">
          {{ $t("sql-editor.show-databases-without-query-permission") }}
        </span>
      </NCheckbox>
    </div>
    <div
      ref="treeContainerElRef"
      class="relative sql-editor-tree--tree flex-1 px-1 pb-1 text-sm select-none"
      :data-height="treeContainerHeight"
    >
      <div
        v-if="treeStore.state === 'READY'"
        class="flex flex-col space-y-2 pb-4"
      >
        <NTree
          ref="treeRef"
          :block-line="true"
          :data="treeStore.tree"
          :show-irrelevant-nodes="false"
          :selected-keys="selectedKeys"
          :pattern="mounted ? searchPattern : ''"
          :default-expand-all="true"
          :expand-on-click="true"
          :node-props="nodeProps"
          :theme-overrides="{ nodeHeight: '21px' }"
          :render-label="renderLabel"
        />
        <div
          v-if="editorStore.canLoadMore"
          class="w-full flex items-center justify-center"
        >
          <NButton
            quaternary
            :size="'small'"
            :loading="editorStore.loading"
            @click="
              () =>
                editorStore
                  .fetchDatabases(searchPattern)
                  .then(() => treeStore.buildTree())
            "
          >
            {{ $t("common.load-more") }}
          </NButton>
        </div>
      </div>
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
import {
  NButton,
  NTree,
  NDropdown,
  NCheckbox,
  type TreeOption,
} from "naive-ui";
import { storeToRefs } from "pinia";
import { ref, nextTick, watch, h, computed } from "vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useDatabaseV1Store,
  useSQLEditorTabStore,
  resolveOpeningDatabaseListFromSQLEditorTabList,
  useSQLEditorTreeStore,
  useSQLEditorStore,
  idForSQLEditorTreeNodeTarget,
  useInstanceResourceByName,
} from "@/store";
import type { ComposedDatabase, SQLEditorTreeNode } from "@/types";
import { DEFAULT_SQL_EDITOR_TAB_MODE } from "@/types";
import { findAncestor, isDescendantOf, isDatabaseV1Queryable } from "@/utils";
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
const editorStore = useSQLEditorStore();
const databaseStore = useDatabaseV1Store();

const editorContext = useSQLEditorContext();
const { events: editorEvents, showConnectionPanel } = editorContext;
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
const selectedKeys = ref<string[]>([]);

// Highlight the current tab's connection node.
const getSelectedKeys = async () => {
  const connection = tabStore.currentTab?.connection;
  if (!connection) {
    return [];
  }

  if (connection.database) {
    const database = await databaseStore.getOrFetchDatabaseByName(
      connection.database
    );
    const node = head(treeStore.nodesByTarget("database", database));
    if (!node) return [];
    return [node.key];
  } else if (connection.instance) {
    const { instance } = useInstanceResourceByName(connection.instance);
    const nodes = treeStore.nodesByTarget("instance", instance.value);
    return nodes.map((node) => node.key);
  }
  return [];
};

const connectedDatabases = computed(() =>
  resolveOpeningDatabaseListFromSQLEditorTabList()
);

const { hasMissingQueryDatabases, showMissingQueryDatabases } =
  storeToRefs(treeStore);

// dynamic render the highlight keywords
const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as any as SQLEditorTreeNode;
  return h(Label, {
    node,
    factors: treeStore.filteredFactorList,
    keyword: searchPattern.value ?? "",
    connectedDatabases: connectedDatabases.value,
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
          if (isDatabaseV1Queryable(node.meta.target as ComposedDatabase)) {
            setConnection(node, {
              extra: {
                worksheet: tabStore.currentTab?.worksheet ?? "",
                mode: DEFAULT_SQL_EDITOR_TAB_MODE,
              },
              context: editorContext,
            });
            showConnectionPanel.value = false;
          }
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

useEmitteryEventListener(editorEvents, "tree-ready", async () => {
  selectedKeys.value = await getSelectedKeys();
});

watch(
  [
    () => editorStore.project,
    () => editorStore.projectContextReady,
    () => searchPattern.value,
  ],
  async ([, ready, search]) => {
    if (!ready) {
      treeStore.state = "LOADING";
    } else {
      await editorStore.prepareDatabases(search);
      treeStore.buildTree();
      treeStore.state = "READY";
      editorEvents.emit("tree-ready");
    }
  },
  { immediate: true }
);
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
  width: 1rem;
}
.sql-editor-tree :deep(.n-tree-node-switcher--hide) {
  width: 0.5rem !important;
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
