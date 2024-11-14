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
    <BasicInfo :task="task" />
    <NTabs
      v-model:value="state.selectedTab"
      class="mt-2 w-full grow"
      type="line"
    >
      <NTabPane name="overview" :tab="$t('common.overview')">
        <Overview :task="task" :latest-task-run="head(state.taskRuns)" />
      </NTabPane>
      <NTabPane
        v-if="state.taskRuns.length > 0"
        name="logs"
        :tab="$t('issue.task-run.logs')"
      >
        <TaskRunLogs :task="task" :task-runs="state.taskRuns" />
      </NTabPane>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import { isEqual, head, sortBy } from "lodash-es";
import { NBreadcrumb, NBreadcrumbItem, NTabs, NTabPane } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { rolloutServiceClient } from "@/grpcweb";
import { getDateForPbTimestamp, unknownStage, unknownTask } from "@/types";
import { TaskRun } from "@/types/proto/v1/rollout_service";
import { isValidTaskName } from "@/utils";
import { useRolloutDetailContext } from "../context";
import BasicInfo from "./BasicInfo.vue";
import Overview from "./Panels/Overview.vue";
import TaskRunLogs from "./Panels/TaskRunLogs.vue";

const props = defineProps<{
  stageId: string;
  taskId: string;
}>();

interface LocalState {
  selectedTab?: "overview" | "logs";
  taskRuns: TaskRun[];
}

const router = useRouter();
const { rollout } = useRolloutDetailContext();
const state = reactive<LocalState>({
  taskRuns: [],
});

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

watchEffect(async () => {
  if (!isValidTaskName(task.value.name)) {
    return;
  }

  // Prepare task runs.
  const { taskRuns } = await rolloutServiceClient.listTaskRuns({
    parent: task.value.name,
  });
  const sorted = sortBy(taskRuns, (t) =>
    getDateForPbTimestamp(t.createTime)
  ).reverse();
  if (!isEqual(sorted, state.taskRuns)) {
    state.taskRuns = sorted;
  }
});
</script>
