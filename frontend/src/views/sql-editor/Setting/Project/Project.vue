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
import { reactive } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import type { ComposedProject } from "@/types";
import ProjectDashboard from "@/views/ProjectDashboard.vue";
import Detail from "./Detail.vue";

interface LocalState {
  project: ComposedProject | undefined;
}

const state = reactive<LocalState>({
  project: undefined,
});

const handleClick = (project: ComposedProject) => {
  state.project = project;
};
</script>
