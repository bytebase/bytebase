<template>
  <HighlightLabelText :text="text" :keyword="keyword" class="truncate" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeFactor as Factor,
  DEFAULT_PROJECT_ID,
} from "@/types";
import HighlightLabelText from "./HighlightLabelText.vue";

const props = defineProps<{
  node: TreeNode;
  factors: Factor[];
  keyword: string;
}>();

const { t } = useI18n();
const project = computed(() => (props.node as TreeNode<"project">).meta.target);

const text = computed(() => {
  if (project.value.uid === String(DEFAULT_PROJECT_ID)) {
    return t("database.unassigned-databases");
  }
  return project.value.title;
});
</script>
