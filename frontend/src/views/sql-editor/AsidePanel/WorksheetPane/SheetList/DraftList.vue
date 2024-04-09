<template>
  <div class="flex flex-col items-stretch gap-y-1">
    <DraftListItem
      v-for="draft in draftList"
      :key="draft.id"
      :draft="draft"
      :keyword="keyword"
    />

    <div
      v-if="draftList.length === 0"
      class="p-2 pl-7 text-control-placeholder"
    >
      {{ $t("common.no-data") }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useSQLEditorTabStore } from "@/store";
import DraftListItem from "./DraftListItem.vue";

const props = defineProps<{
  keyword?: string;
}>();

const emit = defineEmits<{
  (event: "ready"): void;
}>();

const tabStore = useSQLEditorTabStore();

const draftList = computed(() => {
  const tabList = tabStore.tabList.filter((tab) => !tab.sheet);
  const keyword = (props.keyword ?? "").trim().toLowerCase();
  const filteredList = keyword
    ? tabList.filter((tab) => tab.title.toLowerCase().includes(keyword))
    : tabList;
  return filteredList;
});

onMounted(() => {
  emit("ready");
});
</script>
