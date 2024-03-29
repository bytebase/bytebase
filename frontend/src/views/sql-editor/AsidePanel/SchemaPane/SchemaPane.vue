<template>
  <div class="gap-y-1 h-full flex flex-col relative">
    <div class="px-1">
      <SearchBox
        v-model:value="searchPattern"
        size="small"
        style="width: 100%; max-width: 100%"
      />
    </div>

    <div
      ref="treeContainerElRef"
      class="schema-tree flex-1 px-1 pb-1 text-sm overflow-hidden select-none"
      :data-height="treeContainerHeight"
    >
      <NTree
        v-if="tree"
        ref="treeRef"
        :default-expanded-keys="defaultExpandedKeys"
        :selected-keys="selectedKeys"
        :block-line="true"
        :data="tree"
        :show-irrelevant-nodes="false"
        :pattern="mounted ? searchPattern : ''"
        :expand-on-click="true"
        :virtual-scroll="true"
        :node-props="nodeProps"
        :theme-overrides="{ nodeHeight: '21px' }"
        :render-label="renderLabel"
      />
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

    <HoverPanel :offset-x="4" :offset-y="0" :margin="4" />

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

<script setup lang="tsx">
import { computedAsync, useElementSize, useMounted } from "@vueuse/core";
import { NDropdown, type TreeOption } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, nextTick, ref, watch } from "vue";
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
import { findAncestor, isDescendantOf } from "@/utils";
import { useSQLEditorContext } from "../../context";
import { provideHoverStateContext } from "./HoverPanel";
import HoverPanel from "./HoverPanel";
import { Label } from "./TreeNode";
import { selectAllFromTableOrView, useDropdown } from "./actions";
import {
  type NodeTarget,
  type TreeNode,
  buildDatabaseSchemaTree,
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
  state: hoverState,
  position: hoverPosition,
  update: updateHoverState,
} = provideHoverStateContext();
const {
  show: showDropdown,
  context: dropdownContext,
  position: dropdownPosition,
  options: dropdownOptions,
  handleSelect: handleDropdownSelect,
  handleClickoutside: handleDropdownClickoutside,
} = useDropdown();
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
const tree = computed(() => {
  if (isFetchingMetadata.value) return null;
  if (!metadata.value) return null;
  return [buildDatabaseSchemaTree(database.value, metadata.value)];
});
const defaultExpandedKeys = computed(() => {
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
});
const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as any as TreeNode;
  return {
    onClick(e: MouseEvent) {
      if (node.disabled) return;

      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        const { type, target } = node.meta;
        // Check if clicked on the content part.
        // And ignore the fold/unfold arrow.
        const tab = currentTab.value;
        if (tab) {
          if (type === "table" || type === "column") {
            if ("table" in target) {
              const { schema, table } = target as NodeTarget<"table">;
              tab.connection.schema = schema.name;
              tab.connection.table = table.name;
            }
          }
        }
      }
    },
    ondblclick() {
      if (node.meta.type === "table" || node.meta.type === "view") {
        selectAllFromTableOrView(node);
      }
    },
    onmouseenter(e: MouseEvent) {
      const { type } = node.meta;
      if (
        type === "table" ||
        type === "external-table" ||
        type === "column" ||
        type === "view"
      ) {
        const target = node.meta.target as NodeTarget<
          "table" | "external-table" | "column" | "view"
        >;
        if (hoverState.value) {
          updateHoverState(target, "before", 0 /* overrideDelay */);
        } else {
          updateHoverState(target, "before");
        }
        nextTick().then(() => {
          // Find the node element and put the database panel to the right corner
          // of the node
          const wrapper = findAncestor(e.target as HTMLElement, ".n-tree-node");
          if (!wrapper) {
            updateHoverState(undefined, "after", 0 /* overrideDelay */);
            return;
          }
          const bounding = wrapper.getBoundingClientRect();
          hoverPosition.value.x = bounding.right;
          hoverPosition.value.y = bounding.top;
        });
      }
    },
    onmouseleave() {
      updateHoverState(undefined, "after");
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
  return <Label node={node} keyword={searchPattern.value} />;
};

const selectedKeys = computed(() => {
  const db = database.value;
  if (db.uid === String(UNKNOWN_ID)) return [];
  const { schema, table } = connection.value;
  if (!table) return [];
  return [`${db.name}/schemas/${schema}/tables/${table}`];
});

watch(tree, () => {
  showDropdown.value = false;
  dropdownContext.value = undefined;
});
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
  width: 0.5rem;
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
