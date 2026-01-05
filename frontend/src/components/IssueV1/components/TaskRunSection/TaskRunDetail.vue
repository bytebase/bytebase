<template>
  <NTabs v-model:value="state.currentTab" type="line">
    <NTabPane name="LOG" :tab="$t('issue.task-run.logs')">
      <TaskRunLogViewer
        v-if="logEntries.length > 0"
        :key="componentId"
        :entries="logEntries"
        :sheet="sheet"
      />
      <div v-else-if="isFetching" class="py-4 text-gray-500 text-sm">
        {{ $t("common.loading") }}
      </div>
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
import { create } from "@bufbuild/protobuf";
import { computedAsync } from "@vueuse/core";
import { uniqueId } from "lodash-es";
import { RefreshCcwIcon } from "lucide-vue-next";
import { NButton, NTabPane, NTabs } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { TaskRunLogViewer } from "@/components/RolloutV1/components/TaskRunLogViewer";
import { rolloutServiceClientConnect } from "@/connect";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  GetTaskRunLogRequestSchema,
  type TaskRun,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
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

// Sheet is no longer available on TaskRun - it should be fetched from the parent Task
// For now, we'll make it undefined. Components using this should get the sheet from
// the task context if needed.
const sheet = ref<Sheet | undefined>(undefined);

// Fetch task run log
const isFetching = ref(false);
const taskRunLog = computedAsync(
  async () => {
    if (!props.taskRun.name) return undefined;
    const request = create(GetTaskRunLogRequestSchema, {
      parent: props.taskRun.name,
    });
    return await rolloutServiceClientConnect.getTaskRunLog(request);
  },
  undefined,
  { evaluating: isFetching }
);

const logEntries = computed(() => taskRunLog.value?.entries ?? []);

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
