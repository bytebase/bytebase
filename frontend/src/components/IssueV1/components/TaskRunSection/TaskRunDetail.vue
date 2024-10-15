<template>
  <NTabs v-model:value="state.currentTab" type="line">
    <NTabPane name="LOG" :tab="$t('issue.task-run.logs')">
      <TaskRunLogTable :key="componentId" :task-run="taskRun" :sheet="sheet" />
    </NTabPane>
    <NTabPane
      v-if="showTaskRunSessionTab"
      name="SESSION"
      :tab="$t('issue.task-run.session')"
    >
      <TaskRunSession :key="componentId" :task-run="taskRun" />
    </NTabPane>

    <template v-if="showRefreshButton" #suffix>
      <NButton text @click="componentId = uniqueId()">
        <template #icon>
          <RefreshCcwIcon class="w-4 h-auto" />
        </template>
        {{ $t("common.refresh") }}
      </NButton>
    </template>
  </NTabs>
</template>

<script lang="ts" setup>
import { uniqueId } from "lodash-es";
import { RefreshCcwIcon } from "lucide-vue-next";
import { NButton, NTabs, NTabPane } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useSheetV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { TaskRun_Status, type TaskRun } from "@/types/proto/v1/rollout_service";
import { sheetNameOfTaskV1 } from "@/utils";
import { databaseForTask, useIssueContext } from "../../logic";
import TaskRunLogTable from "./TaskRunLogTable";
import TaskRunSession from "./TaskRunSession";

const TASK_RUN_SESSION_SUPPORTED_ENGINES = [Engine.POSTGRES];

interface LocalState {
  currentTab: "LOG" | "SESSION";
}

const props = defineProps<{
  taskRun: TaskRun;
}>();

const { issue, selectedTask } = useIssueContext();

const state = reactive<LocalState>({
  currentTab: "LOG",
});
// Mainly using to force re-render of TaskRunSession component which will re-fetch the session data.
const componentId = ref<string>(uniqueId());

const sheet = computed(() =>
  useSheetV1Store().getSheetByName(sheetNameOfTaskV1(selectedTask.value))
);

const showTaskRunSessionTab = computed(() =>
  TASK_RUN_SESSION_SUPPORTED_ENGINES.includes(
    database.value.instanceResource.engine
  )
);

const showRefreshButton = computed(
  () => props.taskRun.status === TaskRun_Status.RUNNING
);

const database = computed(() =>
  databaseForTask(issue.value, selectedTask.value)
);
</script>
