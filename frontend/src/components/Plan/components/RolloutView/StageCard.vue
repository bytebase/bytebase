<template>
  <div class="!w-80 bg-white z-[1] rounded-lg p-1 hover:shadow">
    <div
      class="w-full flex flex-row justify-between items-center gap-2 px-2 pt-2 pb-1"
    >
      <p class="textlabel">
        {{ environmentStore.getEnvironmentByName(stage.environment).title }}
      </p>
    </div>
    <NVirtualList
      style="max-height: 100vh"
      :items="filteredTasks"
      :item-size="56"
    >
      <template #default="{ item: task }: { item: Task }">
        <div
          :key="task.name"
          class="w-full border-t border-zinc-50 flex flex-col items-start justify-start truncate px-2 py-1 h-14"
        >
          <div class="w-full flex flex-row items-center text-sm truncate">
            <TaskStatus :status="task.status" size="small" />
            <InstanceV1EngineIcon
              class="inline-block ml-2 mr-1"
              :instance="databaseForTask(project, task).instanceResource"
            />
            <span class="truncate">
              {{ databaseForTask(project, task).instanceResource.title }}
            </span>
            <ChevronRightIcon class="inline opacity-60 w-4 shrink-0" />
            <span class="truncate">
              {{ databaseForTask(project, task).databaseName }}
            </span>
          </div>
          <p class="space-x-1 mt-0.5 leading-4">
            <NTooltip>
              <template #trigger>
                <NTag round size="tiny">{{ semanticTaskType(task.type) }}</NTag>
              </template>
              {{ $t("common.type") }}
            </NTooltip>
            <NTooltip v-if="extractSchemaVersionFromTask(task)">
              <template #trigger>
                <NTag round size="tiny">
                  {{ extractSchemaVersionFromTask(task) }}
                </NTag>
              </template>
              {{ $t("common.version") }}
            </NTooltip>
          </p>
        </div>
      </template>
    </NVirtualList>
  </div>
</template>

<script setup lang="ts">
import { ChevronRightIcon } from "lucide-vue-next";
import { NTag, NVirtualList, NTooltip } from "naive-ui";
import { computed } from "vue";
import { semanticTaskType } from "@/components/IssueV1";
import TaskStatus from "@/components/Rollout/RolloutDetail/Panels/kits/TaskStatus.vue";
import { databaseForTask } from "@/components/Rollout/RolloutDetail/utils";
import { InstanceV1EngineIcon } from "@/components/v2";
import { useEnvironmentV1Store, useCurrentProjectV1 } from "@/store";
import {
  Stage,
  type Task,
  type Rollout,
  type Task_Status,
} from "@/types/proto/v1/rollout_service";
import { extractSchemaVersionFromTask } from "@/utils";

const props = defineProps<{
  stage: Stage;
  rollout: Rollout;
  taskStatusFilter: Task_Status[];
}>();

const environmentStore = useEnvironmentV1Store();
const { project } = useCurrentProjectV1();

const filteredTasks = computed(() => {
  if (props.taskStatusFilter.length === 0) {
    return props.stage.tasks;
  }
  return props.stage.tasks.filter((task) =>
    props.taskStatusFilter.includes(task.status)
  );
});
</script>
