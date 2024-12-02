<template>
  <div class="w-full flex flex-col">
    <div class="w-full flex flex-row justify-between items-center gap-4">
      <div class="w-full flex flex-row items-center gap-2">
        <TaskStatus :size="'large'" :status="task.status" />
        <p class="text-2xl flex flex-row items-center">
          <InstanceV1EngineIcon
            class="mr-1"
            :size="'large'"
            :instance="
              databaseForTask(rollout.projectEntity, task).instanceResource
            "
          />
          <span>{{
            databaseForTask(rollout.projectEntity, task).instanceResource.title
          }}</span>
          <ChevronRightIcon class="inline opacity-60 mx-0.5 w-5" />
          <span>{{
            databaseForTask(rollout.projectEntity, task).databaseName
          }}</span>
        </p>
      </div>
      <div class="flex flex-row justify-end">
        <TaskStatusActions v-if="showActionButtons" :task="task" />
      </div>
    </div>
    <div class="mt-3 space-x-2">
      <NTooltip>
        <template #trigger>
          <NTag round>{{ semanticTaskType(task.type) }}</NTag>
        </template>
        {{ $t("common.type") }}
      </NTooltip>
      <NTooltip v-if="extractSchemaVersionFromTask(task)">
        <template #trigger>
          <NTag round>
            {{ extractSchemaVersionFromTask(task) }}
          </NTag>
        </template>
        {{ $t("common.version") }}
      </NTooltip>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ChevronRightIcon } from "lucide-vue-next";
import { NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import { semanticTaskType } from "@/components/IssueV1";
import { InstanceV1EngineIcon } from "@/components/v2";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { extractSchemaVersionFromTask } from "@/utils";
import TaskStatus from "../Panels/kits/TaskStatus.vue";
import { useRolloutDetailContext } from "../context";
import { databaseForTask } from "../utils";
import TaskStatusActions from "./TaskStatusActions.vue";
import { useTaskDetailContext } from "./context";

const { rollout } = useRolloutDetailContext();
const { task } = useTaskDetailContext();

const showActionButtons = computed(() => {
  return task.value.status !== Task_Status.DONE;
});
</script>
