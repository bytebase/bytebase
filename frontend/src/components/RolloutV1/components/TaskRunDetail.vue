<template>
  <NTabs v-model:value="state.currentTab" type="line">
    <NTabPane name="LOG" :tab="$t('issue.task-run.logs')">
      <TaskRunLogViewer
        v-if="taskRun.name"
        :key="componentId"
        :task-run-name="taskRun.name"
      />
    </NTabPane>
    <NTabPane
      v-if="showTaskRunSessionTab"
      name="SESSION"
      :tab="$t('issue.task-run.session')"
    >
      <TaskRunSession :key="componentId" :task-run="taskRun" />
    </NTabPane>

    <template v-if="showRefreshButton" #suffix>
      <NButton text @click="refresh">
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
import { NButton, NTabPane, NTabs } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  type TaskRun,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRunLogViewer } from "./TaskRunLogViewer";
import TaskRunSession from "./TaskRunSession";

const TASK_RUN_SESSION_SUPPORTED_ENGINES = [Engine.POSTGRES];

interface LocalState {
  currentTab: "LOG" | "SESSION";
}

const props = defineProps<{
  taskRun: TaskRun;
  database?: Database;
}>();

const state = reactive<LocalState>({
  currentTab: "LOG",
});
// Mainly using to force re-render of TaskRunSession component which will re-fetch the session data.
const componentId = ref<string>(uniqueId());

const refresh = () => {
  componentId.value = uniqueId();
};

const showTaskRunSessionTab = computed(
  () =>
    props.database?.instanceResource &&
    TASK_RUN_SESSION_SUPPORTED_ENGINES.includes(
      props.database.instanceResource.engine
    )
);

const showRefreshButton = computed(
  () => props.taskRun.status === TaskRun_Status.RUNNING
);
</script>
