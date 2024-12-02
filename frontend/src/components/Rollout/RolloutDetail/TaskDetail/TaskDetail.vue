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
    <BasicInfo />
    <NTabs
      v-model:value="state.selectedTab"
      class="mt-2 w-full grow"
      type="line"
    >
      <NTabPane name="overview" :tab="$t('common.overview')">
        <Overview />
      </NTabPane>
      <NTabPane
        v-if="taskRuns.length > 0"
        name="logs"
        :tab="$t('issue.task-run.logs')"
      >
        <TaskRunLogs />
      </NTabPane>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import { NBreadcrumb, NBreadcrumbItem, NTabs, NTabPane } from "naive-ui";
import { reactive } from "vue";
import { useRouter } from "vue-router";
import { useRolloutDetailContext } from "../context";
import BasicInfo from "./BasicInfo.vue";
import Overview from "./Panels/Overview.vue";
import TaskRunLogs from "./Panels/TaskRunLogs.vue";
import { provideTaskDetailContext } from "./context";

const props = defineProps<{
  stageId: string;
  taskId: string;
}>();

interface LocalState {
  selectedTab?: "overview" | "logs";
}

const router = useRouter();
const { rollout } = useRolloutDetailContext();
const state = reactive<LocalState>({});

const { stage, task, taskRuns } = provideTaskDetailContext(
  props.stageId,
  props.taskId
);
</script>
