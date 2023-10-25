<template>
  <div class="flex flex-col gap-y-2">
    <NavBar :disable-project-select="!!project" />

    <ChangelistTable
      :changelists="filteredChangelists"
      :is-fetching="isFetching"
      :keyword="filter.keyword"
    />

    <CreateChangelistPanel
      :project-uid="project?.uid"
      :disable-project-select="!!project"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useChangelistStore,
  useCurrentUserV1,
  useProjectV1Store,
} from "@/store";
import { ComposedProject } from "@/types";
import { Changelist } from "@/types/proto/v1/changelist_service";
import { extractProjectResourceName, isMemberOfProjectV1 } from "@/utils";
import ChangelistTable from "./ChangelistTable.vue";
import CreateChangelistPanel from "./CreateChangelistPanel.vue";
import NavBar from "./NavBar.vue";
import { provideChangelistDashboardContext } from "./context";

const props = defineProps<{
  project?: ComposedProject;
}>();

const me = useCurrentUserV1();
const { filter, events } = provideChangelistDashboardContext(
  props.project?.name
);

const isFetching = ref(false);
const changelists = ref<Changelist[]>([]);

const filteredChangelists = computed(() => {
  let list = changelists.value.filter((changelist) => {
    // The server-side has not implemented ACL by now.
    // So we manually filter the changelists by project here to workaround.
    const project = useProjectV1Store().getProjectByName(
      `projects/${extractProjectResourceName(changelist.name)}`
    );
    return isMemberOfProjectV1(project.iamPolicy, me.value);
  });
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
