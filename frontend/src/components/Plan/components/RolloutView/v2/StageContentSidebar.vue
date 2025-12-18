<template>
  <!-- Desktop: Sticky sidebar -->
  <div
    v-if="isWideScreen"
    class="shrink-0 pr-4 self-start sticky top-0 w-72 xl:w-80"
  >
    <!-- Rollback action at top -->
    <StageTaskRunsRollback class="px-3 pt-1 pb-2" :stage="stage" />
    <!-- Run history section with border -->
    <div class="border rounded-lg">
      <StageTimeline
        :stage="stage"
        :task-runs="taskRuns"
        mode="sidebar"
      />
    </div>
  </div>

  <!-- Mobile: Drawer trigger button -->
  <template v-else>
    <NButton
      size="small"
      quaternary
      class="!px-2"
      @click="showDrawer = true"
    >
      <template #icon>
        <MenuIcon :size="16" />
      </template>
    </NButton>

    <!-- Mobile drawer -->
    <Drawer v-model:show="showDrawer" placement="right">
      <DrawerContent :title="environment?.title" style="width: 20rem">
        <!-- Rollback action at top -->
        <StageTaskRunsRollback class="pb-3" :stage="stage" />
        <StageTimeline
          :stage="stage"
          :task-runs="taskRuns"
          mode="drawer"
        />
      </DrawerContent>
    </Drawer>
  </template>
</template>

<script lang="ts" setup>
import { useWindowSize } from "@vueuse/core";
import { MenuIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useEnvironmentV1Store } from "@/store";
import type { Stage, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import StageTaskRunsRollback from "./StageTaskRunsRollback.vue";
import StageTimeline from "./StageTimeline.vue";

const props = defineProps<{
  stage: Stage | null | undefined;
  taskRuns: TaskRun[];
}>();

const environmentStore = useEnvironmentV1Store();
const { width: windowWidth } = useWindowSize();
const isWideScreen = computed(() => windowWidth.value >= 768);
const showDrawer = ref(false);

const environment = computed(() => {
  if (!props.stage) {
    return null;
  }
  return environmentStore.getEnvironmentByName(props.stage.environment);
});
</script>
