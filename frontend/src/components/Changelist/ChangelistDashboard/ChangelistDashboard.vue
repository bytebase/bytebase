<template>
  <div class="flex flex-col gap-y-4">
    <NavBar :allow-create="allowCreate" />
    <ChangelistDataTable
      :changelists="filteredChangelists"
      :loading="isFetching"
      :show-project="!project"
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
import type { Changelist } from "@/types/proto-es/v1/changelist_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import ChangelistDataTable from "./ChangelistDataTable.vue";
import CreateChangelistPanel from "./CreateChangelistPanel.vue";
import NavBar from "./NavBar.vue";
import { provideChangelistDashboardContext } from "./context";

const props = defineProps<{
  project: Project;
}>();

const { filter, events } = provideChangelistDashboardContext(
  computed(() => props.project.name)
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

const allowCreate = computed(() => {
  return hasProjectPermissionV2(props.project, "bb.changelists.create");
});
</script>
