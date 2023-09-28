<template>
  <div class="flex flex-col gap-y-4 px-4">
    <NavBar />

    <ChangeTable :changes="state.changes" :reorder-mode="reorderMode" />

    <AddChangePanel />
  </div>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { computed, reactive, watch } from "vue";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import AddChangePanel from "./AddChangePanel";
import ChangeTable from "./ChangeTable";
import NavBar from "./NavBar";
import { provideChangelistDetailContext } from "./context";

const { changelist, reorderMode } = provideChangelistDetailContext();

const state = reactive<{
  changes: Change[];
}>({
  changes: [],
});

const documentTitle = computed(() => {
  return changelist.value.description;
});
useTitle(documentTitle);

watch(
  () => changelist.value.changes,
  (changes) => {
    state.changes = [...changes];
  },
  { immediate: true }
);
</script>
