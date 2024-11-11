<template>
  <NBreadcrumb class="mb-4">
    <NBreadcrumbItem @click="router.push(`/${rollout.name}`)">
      {{ rollout.title }}
    </NBreadcrumbItem>
    <NBreadcrumbItem :clickable="false">
      {{ stage.title }}
    </NBreadcrumbItem>
    <NBreadcrumbItem @click="router.push(`/${rollout.name}#tasks`)">
      {{ $t("common.tasks") }}
    </NBreadcrumbItem>
    <NBreadcrumbItem :clickable="false">
      {{ task.title }}
    </NBreadcrumbItem>
  </NBreadcrumb>
  <div v-if="task" class="w-full flex flex-col">
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
        <span class="truncate flex items-center">
          {{
            databaseForTask(rollout.projectEntity, task).instanceResource.title
          }}
          <ChevronRightIcon class="inline opacity-60 mx-0.5 w-5" />
          {{ databaseForTask(rollout.projectEntity, task).databaseName }}
        </span>
      </p>
    </div>
    <div class="mt-3 space-x-2">
      <NTag round>{{ semanticTaskType(task.type) }}</NTag>
      <NTag v-if="extractSchemaVersionFromTask(task)" round>
        {{ extractSchemaVersionFromTask(task) }}
      </NTag>
    </div>
    <template v-if="latestTaskRun">
      <NDivider />
      <p class="w-auto flex items-center text-base text-main mb-2">
        {{ $t("issue.task-run.logs") }}
      </p>
      <TaskRunLogTable
        :task-run="latestTaskRun"
        :sheet="sheetStore.getSheetByName(sheetNameOfTaskV1(task))"
      />
    </template>
    <NDivider />
    <p class="w-auto flex items-center text-base text-main mb-2">
      {{ $t("common.statement") }}
      <button
        tabindex="-1"
        class="btn-icon ml-1"
        @click.prevent="copyStatement"
      >
        <ClipboardIcon class="w-4 h-4" />
      </button>
    </p>
    <MonacoEditor
      class="h-auto max-h-[480px] min-h-[120px] border rounded-[3px] text-sm overflow-clip relative"
      :content="statement"
      :readonly="true"
      :auto-height="{ min: 256, max: 512 }"
    />
  </div>
</template>

<script lang="ts" setup>
import { head, isEqual } from "lodash-es";
import { ClipboardIcon, ChevronRightIcon } from "lucide-vue-next";
import { NBreadcrumb, NBreadcrumbItem, NDivider, NTag } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { semanticTaskType } from "@/components/IssueV1";
import TaskRunLogTable from "@/components/IssueV1/components/TaskRunSection/TaskRunLogTable/TaskRunLogTable.vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { InstanceV1EngineIcon } from "@/components/v2";
import { rolloutServiceClient } from "@/grpcweb";
import { pushNotification, useSheetV1Store } from "@/store";
import { unknownStage, unknownTask } from "@/types";
import type { TaskRun } from "@/types/proto/v1/rollout_service";
import {
  extractSchemaVersionFromTask,
  getSheetStatement,
  isValidTaskName,
  sheetNameOfTaskV1,
  toClipboard,
} from "@/utils";
import TaskStatus from "../Panels/kits/TaskStatus.vue";
import { useRolloutDetailContext } from "../context";
import { databaseForTask } from "../utils";

const props = defineProps<{
  stageId: string;
  taskId: string;
}>();

const router = useRouter();
const { rollout } = useRolloutDetailContext();
const sheetStore = useSheetV1Store();
const latestTaskRun = ref<TaskRun | undefined>(undefined);

const stage = computed(() => {
  return (
    rollout.value.stages.find((stage) =>
      stage.name.endsWith(`/${props.stageId}`)
    ) || unknownStage()
  );
});

const task = computed(() => {
  return (
    stage.value.tasks.find((task) => task.name.endsWith(`/${props.taskId}`)) ||
    unknownTask()
  );
});

const statement = computed(() => {
  const sheet = sheetStore.getSheetByName(sheetNameOfTaskV1(task.value));
  if (sheet) {
    return getSheetStatement(sheet);
  }
  return "";
});

watchEffect(async () => {
  if (!isValidTaskName(task.value.name)) {
    return;
  }

  // Prepare the sheet for the task.
  const sheet = sheetNameOfTaskV1(task.value);
  if (sheet) {
    await sheetStore.getOrFetchSheetByName(sheet);
  }

  // Prepare the latest task run logs.
  const { taskRuns } = await rolloutServiceClient.listTaskRuns({
    parent: task.value.name,
  });
  if (!isEqual(latestTaskRun.value, head(taskRuns))) {
    latestTaskRun.value = head(taskRuns);
  }
});

const copyStatement = async () => {
  toClipboard(statement.value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: `Statement copied to clipboard.`,
    });
  });
};
</script>
