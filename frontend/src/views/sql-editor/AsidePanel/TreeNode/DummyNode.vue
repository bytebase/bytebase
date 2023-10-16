<template>
  <NPopover :disabled="!error">
    <template #trigger>
      <span class="text-control-placeholder">{{ text }}</span>
    </template>
    <template #default>
      <div class="text-error max-w-[20rem] break-words break-all">
        {{ error }}
      </div>
    </template>
  </NPopover>
</template>

<script setup lang="ts">
import { NPopover } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeFactor as Factor,
} from "@/types";

const props = defineProps<{
  node: TreeNode;
  factors: Factor[];
  keyword: string;
}>();

const { t } = useI18n();

const error = computed(() => {
  return (props.node as TreeNode<"dummy">).meta.target.error;
});

const text = computed(() => {
  return `<${error.value ? t("common.error") : t("common.empty")}>`;
});
</script>
