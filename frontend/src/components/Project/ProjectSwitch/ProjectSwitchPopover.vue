<template>
  <NPopover
    v-model:show="state.showPopover"
    class="max-h-[80vh] w-[24rem] max-w-full"
    placement="bottom-start"
    scrollable
    trigger="click"
    :show-arrow="false"
  >
    <template #trigger>
      <NButton
        class="hidden sm:inline"
        size="small"
        @click="state.showPopover = !state.showPopover"
        icon-placement="right"
      >
        <div class="min-w-[8rem] text-left">
          <ProjectNameCell
            v-if="isValidProjectName(project.name)"
            mode="ALL_SHORT"
            :project="project"
          />
          <span v-else class="text-control-placeholder text-sm">
            {{ $t("project.select") }}
          </span>
        </div>
        <template #icon>
          <ChevronDownIcon class="w-5 h-auto opacity-80" />
        </template>
      </NButton>
    </template>

    <ProjectSwitchContent
      @on-create="
        () => {
          state.showCreateDrawer = true;
          state.showPopover = false;
        }
      "
    />
  </NPopover>
  <Drawer
    :auto-focus="true"
    :close-on-esc="true"
    :show="state.showCreateDrawer"
    @close="state.showCreateDrawer = false"
  >
    <ProjectCreatePanel @dismiss="state.showCreateDrawer = false" />
  </Drawer>
</template>

<script lang="ts" setup>
import { ChevronDownIcon } from "lucide-vue-next";
import { NButton, NPopover } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useRouter } from "vue-router";
import ProjectCreatePanel from "@/components/Project/ProjectCreatePanel.vue";
import { Drawer } from "@/components/v2";
import { ProjectNameCell } from "@/components/v2/Model/DatabaseV1Table/cells";
import { isValidProjectName } from "@/types";
import { useCurrentProject } from "../useCurrentProject";
import ProjectSwitchContent from "./ProjectSwitchContent.vue";

interface LocalState {
  showPopover: boolean;
  showCreateDrawer: boolean;
}

const state = reactive<LocalState>({
  showPopover: false,
  showCreateDrawer: false,
});
const router = useRouter();

const params = computed(() => {
  const route = router.currentRoute.value;
  return {
    projectId: route.params.projectId as string | undefined,
    issueSlug: route.params.issueSlug as string | undefined,
    instanceId: route.params.instanceId as string | undefined,
    databaseName: route.params.databaseName as string | undefined,
    changelogId: route.params.changelogId as string | undefined,
  };
});

const { project } = useCurrentProject(params);

// Close popover when current project changed.
watch(
  () => project.value.name,
  () => {
    state.showPopover = false;
  }
);
</script>
