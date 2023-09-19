<template>
  <NTree
    class="bb-schema-diagram-nav-tree select-none"
    :class="[isFlatTree && 'flat']"
    :data="treeData"
    :block-line="true"
    :pattern="keyword"
    :show-irrelevant-nodes="false"
    :default-expand-all="true"
    :selected-keys="[]"
    :expand-on-click="true"
    :render-label="renderLabel"
    :render-prefix="renderPrefix"
    :render-suffix="renderSuffix"
    :node-props="nodeProps"
  />
</template>

<script lang="ts" setup>
import { NTree, type TreeOption } from "naive-ui";
import { computed, h } from "vue";
import { isDescendantOf } from "@/utils";
import { useSchemaDiagramContext } from "../common";
import { DEFAULT_PADDINGS } from "../common/const";
import { Label, Prefix, Suffix } from "./TreeNode";
import { TreeNode } from "./types";
import { isTypedNode } from "./utils";

const props = withDefaults(
  defineProps<{
    keyword?: string;
  }>(),
  {
    keyword: "",
  }
);

const context = useSchemaDiagramContext();
const { databaseMetadata, events } = context;

const isFlatTree = computed(() => {
  const { schemas } = databaseMetadata.value;
  return schemas.length === 1 && schemas[0].name === "";
});

const treeData = computed(() => {
  const schemaNodeList = databaseMetadata.value.schemas.map<TreeNode<"schema">>(
    (schema) => {
      const children = schema.tables.map<TreeNode<"table">>((table) => {
        return {
          key: table.name,
          label: table.name,
          type: "table",
          data: table,
          isLeaf: true,
          children: [],
        };
      });
      return {
        key: schema.name,
        label: schema.name,
        type: "schema",
        data: schema,
        isLeaf: children.length === 0,
        children,
      };
    }
  );
  if (schemaNodeList.length === 1 && schemaNodeList[0].data.name === "") {
    return schemaNodeList[0].children;
  }
  return schemaNodeList;
});

// dynamic render the highlight keywords
const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as any as TreeNode;
  return h(Label, { node, keyword: props.keyword });
};

// Render icons before nodes.
const renderPrefix = ({ option }: { option: TreeOption }) => {
  const node = option as any as TreeNode;
  return h(Prefix, { node });
};

// Render icons after nodes.
const renderSuffix = ({ option }: { option: TreeOption }) => {
  const node = option as any as TreeNode;
  return h(Suffix, { node });
};

const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as TreeNode;
  return {
    onClick(e: MouseEvent) {
      if (node.disabled) return;

      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        // Check if clicked on the content part.
        // And ignore the fold/unfold arrow.
        if (isTypedNode(node, "table")) {
          events.emit("set-center", {
            type: "table",
            target: node.data,
            padding: DEFAULT_PADDINGS,
          });
        }
      }
    },
    "data-node-type": node.type,
  };
};
</script>

<style lang="postcss">
.bb-schema-diagram-nav-tree.flat .n-tree-node-switcher {
  @apply !hidden;
}
.bb-schema-diagram-nav-tree .n-tree-node-content {
  @apply !pl-0;
}
.bb-schema-diagram-nav-tree .n-tree-node-content__prefix {
  @apply shrink-0 !mr-1;
}
.bb-schema-diagram-nav-tree .n-tree-node-content__text {
  @apply truncate mr-1;
}
.bb-schema-diagram-nav-tree .n-tree-node--pending {
  background-color: transparent !important;
}
.bb-schema-diagram-nav-tree .n-tree-node--pending:hover {
  background-color: var(--n-node-color-hover) !important;
}
.bb-schema-diagram-nav-tree .n-tree-node--selected,
.bb-schema-diagram-nav-tree .n-tree-node--selected:hover {
  background-color: var(--n-node-color-active) !important;
}
</style>
