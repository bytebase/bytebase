<template>
  <div class="flex flex-col gap-y-2 px-4">
    <NavBar />

    <ChangelistTable
      :changelists="filteredChangelists"
      :is-fetching="isFetching"
      :keyword="filter.keyword"
    />

    <CreateChangelistPanel />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useChangelistStore } from "@/store";
import { Changelist } from "@/types/proto/v1/changelist_service";
import ChangelistTable from "./ChangelistTable.vue";
import CreateChangelistPanel from "./CreateChangelistPanel.vue";
import NavBar from "./NavBar.vue";
import { provideChangelistDashboardContext } from "./context";

const { filter, events } = provideChangelistDashboardContext();

const isFetching = ref(false);
const changelists = ref<Changelist[]>([]);

const filteredChangelists = computed(() => {
  const keyword = filter.value.keyword.trim();
  if (!keyword) return changelists.value;
  return changelists.value.filter((changelist) => {
    return changelist.description.toLowerCase().includes(keyword.toLowerCase());
  });
});

const fetchChangeLists = async () => {
  isFetching.value = true;
  try {
    const response = await useChangelistStore().fetchChangelists({
      parent: filter.value.project,
    });

    changelists.value = response.changelists;
  } finally {
    isFetching.value = false;
  }
};

useEmitteryEventListener(events, "refresh", fetchChangeLists);
onMounted(fetchChangeLists);
</script>
