<template>
  <AbstractListItem
    :title="draft.title"
    :selected="selected"
    :data-item-key="keyForDraft(draft)"
    :keyword="keyword"
    @click="handleClick"
  >
    <template #suffix></template>
  </AbstractListItem>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useSQLEditorTabStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { keyForDraft } from "../common";
import AbstractListItem from "./AbstractListItem.vue";

const props = defineProps<{
  draft: SQLEditorTab;
  keyword?: string;
}>();

const tabStore = useSQLEditorTabStore();

const selected = computed(() => {
  const tab = tabStore.currentTab;

  return tab?.id === props.draft.id;
});

const handleClick = () => {
  tabStore.setCurrentTabId(props.draft.id);
};
</script>
