<template>
  <div
    class="w-full h-full flex flex-col gap-4 py-4 overflow-y-auto"
    v-bind="$attrs"
  >
    <ProjectDashboard :on-row-click="handleClick" />
  </div>
  <Drawer
    v-if="state.project"
    :show="state.project !== undefined"
    :close-on-esc="true"
    :mask-closable="true"
    @update:show="() => (state.project = undefined)"
  >
    <DrawerContent
      :title="`${$t('common.project')} - ${state.project?.title}`"
      class="project-detail-drawer"
      body-content-class="flex flex-col gap-2 overflow-hidden"
    >
      <Detail :project="state.project" />
    </DrawerContent>
  </Drawer>
</template>

<script lang="tsx" setup>
import { reactive, onMounted } from "vue";
import { useRoute } from "vue-router";
import { Drawer, DrawerContent } from "@/components/v2";
import { useProjectV1Store } from "@/store";
import { isValidProjectName } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import ProjectDashboard from "@/views/ProjectDashboard.vue";
import Detail from "./Detail.vue";

interface LocalState {
  project: Project | undefined;
}

const route = useRoute();
const state = reactive<LocalState>({
  project: undefined,
});
const projectStore = useProjectV1Store();

const handleClick = (project: Project) => {
  state.project = project;
};

onMounted(async () => {
  const projectName = route.hash.slice(1);
  if (!isValidProjectName(projectName)) {
    return;
  }

  try {
    const project = await projectStore.getOrFetchProjectByName(
      projectName,
      true
    );
    handleClick(project);
  } catch {
    // nothing
  }
});
</script>
