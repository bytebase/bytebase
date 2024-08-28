<template>
  <CommonNode :text="text" :keyword="keyword" :highlight="true" :indent="0">
    <template #icon>
      <ColumnIcon class="w-4 h-4" />
    </template>
    <template #suffix>
      <div
        class="flex items-center justify-end gap-1 overflow-hidden whitespace-nowrap shrink-0"
      >
        <NButton size="tiny" text @click.stop="goRef">
          <template #icon>
            <FileSymlinkIcon class="w-4 h-4" />
          </template>
        </NButton>
      </div>
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { FileSymlinkIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { ColumnIcon } from "@/components/Icon";
import { useEditorPanelContext } from "@/views/sql-editor/EditorPanel";
import type { TreeNode } from "../common";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();
const { updateViewState } = useEditorPanelContext();

const target = computed(
  () => (props.node as TreeNode<"dependent-column">).meta.target
);

const text = computed(() => {
  const { schema, table, column } = target.value.dependentColumn;
  const parts = [table, column];
  if (schema) parts.unshift(schema);
  return parts.join(".");
});

const goRef = () => {
  const { schema, table, column } = target.value.dependentColumn;
  updateViewState({
    view: "TABLES",
    schema,
    detail: {
      table,
      column,
    },
  });
};
</script>
