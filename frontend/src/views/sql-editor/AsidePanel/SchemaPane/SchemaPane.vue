<template>
  <div class="gap-y-1 h-full flex flex-col">
    <teleport to="#sql-editor-debug">
      <li>[SchemaPane]database: {{ database.name }}</li>
      <li>[SchemaPane]isFetchingMetadata: {{ isFetchingMetadata }}</li>
      <li>[SchemaPane]typeof metadata: {{ typeof metadata }}</li>
      <li>[SchemaPane]typeof tree: {{ typeof tree }}</li>
      <li>[SchemaPane]treeContainerHeight: {{ treeContainerHeight }}</li>
    </teleport>

    <div class="px-1">
      <SearchBox size="small" style="width: 100%; max-width: 100%" />
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
  </div>
</template>

<script setup lang="tsx">
import { computedAsync, useElementSize, useMounted } from "@vueuse/core";
import type { TreeOption } from "naive-ui";
import { computed, ref } from "vue";
import { SearchBox } from "@/components/v2";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { Label } from "./TreeNode";
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
const { database } = useConnectionOfCurrentSQLEditorTab();
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
    ondblclick() {
      // if (node.meta.type === "table" || node.meta.type === "view") {
      //   selectAllFromTableOrView(node);
      // }
    },
    "data-node-meta-type": node.meta.type,
    "data-node-key": node.key,
  };
};

// dynamic render the highlight keywords
const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as any as TreeNode;
  return <Label node={node} keyword={searchPattern.value} />;
};

// const selectAllFromTableOrView = async (node: SQLEditorTreeNode) => {
//   const { type, target } = (node as SQLEditorTreeNode<"table" | "view">).meta;
//   const { database } = target;
//   const { engine } = database.instanceEntity;
//   const LIMIT = 50; // default pagesize of SQL Editor

//   const language = languageOfEngineV1(engine);
//   if (language === "redis") {
//     return; // not supported
//   }
//   const schema = target.schema.name;
//   const tableOrViewName =
//     type === "table"
//       ? (target as SQLEditorTreeNodeTarget<"table">).table.name
//       : type === "view"
//         ? (target as SQLEditorTreeNodeTarget<"view">).view.name
//         : "";
//   if (!tableOrViewName) {
//     return;
//   }

//   const runQuery = async (statement: string) => {
//     const tab: CoreSQLEditorTab = {
//       connection: {
//         instance: database.instance,
//         database: database.name,
//         schema,
//         table: tableOrViewName,
//       },
//       mode: DEFAULT_SQL_EDITOR_TAB_MODE,
//       sheet: "",
//     };
//     if (
//       tabStore.currentTab &&
//       (tabStore.currentTab.status === "NEW" || !tabStore.currentTab.sheet)
//     ) {
//       // If the current tab is "fresh new" or unsaved, update its connection directly.
//       tabStore.updateCurrentTab({
//         ...tab,
//         title: suggestedTabTitleForSQLEditorConnection(tab.connection),
//         status: "DIRTY",
//         statement,
//       });
//     } else {
//       // Otherwise select or add a new tab and set its connection
//       tabStore.addTab(
//         {
//           ...tab,
//           title: suggestedTabTitleForSQLEditorConnection(tab.connection),
//           statement,
//           status: "DIRTY",
//         },
//         /* beside */ true
//       );
//     }
//     await nextTick();
//     executeReadonly({
//       statement,
//       connection: { ...tab.connection },
//       explain: false,
//       engine: database.instanceEntity.engine,
//     });
//   };

//   if (language === "javascript" && type === "table") {
//     // mongodb
//     const query = `db["${tableOrViewName}"].find().limit(${LIMIT});`;
//     runQuery(query);
//     return;
//   }

//   if (type === "table") {
//     maybeSelectTable(node);
//   }

//   const query = generateSimpleSelectAllStatement(
//     engine,
//     schema,
//     tableOrViewName,
//     LIMIT
//   );
//   runQuery(query);
// };
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
