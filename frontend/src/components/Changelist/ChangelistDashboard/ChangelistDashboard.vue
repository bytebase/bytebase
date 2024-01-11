<template>
  <div class="flex flex-col gap-y-4">
    <NavBar :disable-project-select="!!project" />

    <ChangelistTable
      :changelists="filteredChangelists"
      :is-fetching="isFetching"
      :keyword="filter.keyword"
      :hide-project-column="!!project"
    />

    <CreateChangelistPanel
      :project="project"
      :disable-project-select="!!project"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useChangelistStore } from "@/store";
import { ComposedProject } from "@/types";
import { Changelist } from "@/types/proto/v1/changelist_service";
import ChangelistTable from "./ChangelistTable.vue";
import CreateChangelistPanel from "./CreateChangelistPanel.vue";
import NavBar from "./NavBar.vue";
import { provideChangelistDashboardContext } from "./context";

const props = defineProps<{
  project?: ComposedProject;
}>();

const { filter, events } = provideChangelistDashboardContext(
  props.project?.name
);

const isFetching = ref(false);
const changelists = ref<Changelist[]>([]);

const filteredChangelists = computed(() => {
  let list = changelists.value;
  const keyword = filter.value.keyword.trim();
  if (keyword) {
    list = list.filter((changelist) => {
      return changelist.description
        .toLowerCase()
        .includes(keyword.toLowerCase());
    });
  }
  return list;
});

const fetchChangelists = async () => {
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

useEmitteryEventListener(events, "refresh", fetchChangelists);
onMounted(fetchChangelists);
</script>
