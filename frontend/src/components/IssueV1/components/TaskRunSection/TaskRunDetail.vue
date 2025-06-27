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
import { computed, reactive, ref, watch } from "vue";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import { useSheetV1Store, useCurrentProjectV1 } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { TaskRun_Status, type TaskRun } from "@/types/proto/v1/rollout_service";
import { useIssueContext } from "../../logic";
import TaskRunLogTable from "./TaskRunLogTable";
import TaskRunSession from "./TaskRunSession";

const TASK_RUN_SESSION_SUPPORTED_ENGINES = [Engine.POSTGRES];

interface LocalState {
  currentTab: "LOG" | "SESSION";
}

const props = defineProps<{
  taskRun: TaskRun;
}>();

const { project } = useCurrentProjectV1();
const { selectedTask } = useIssueContext();
const sheetStore = useSheetV1Store();

const state = reactive<LocalState>({
  currentTab: "LOG",
});
// Mainly using to force re-render of TaskRunSession component which will re-fetch the session data.
const componentId = ref<string>(uniqueId());

const sheet = computed(() =>
  useSheetV1Store().getSheetByName(props.taskRun.sheet)
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
  databaseForTask(project.value, selectedTask.value)
);

watch(
  () => props.taskRun.name,
  async () => {
    // Prepare the sheet data from task run.
    if (props.taskRun.sheet) {
      await sheetStore.getOrFetchSheetByName(props.taskRun.sheet, "FULL");
    }
  },
  { immediate: true }
);
</script>
