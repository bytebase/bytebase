<template>
  <CommonDrawer
    :show="true"
    :title="title"
    :loading="state.loading"
    @show="resetState"
    @close="$emit('close')"
  >
    <template #default>
      <div class="flex flex-col gap-y-4 h-full overflow-y-hidden px-1">
        <div
          class="flex flex-row gap-x-2 shrink-0 overflow-y-hidden justify-start items-center"
        >
          <label class="font-medium text-control">
            {{ $t("common.stage") }}
          </label>
          <span class="break-all">
            {{
              environmentStore.getEnvironmentByName(targetStage.environment)
                .title
            }}
          </span>
        </div>

        <div
          class="flex flex-col gap-y-1 shrink overflow-y-hidden justify-start"
        >
          <div class="flex items-center justify-between">
            <label class="font-medium text-control">
              <template v-if="eligibleTasks.length === 1">
                {{ $t("common.task") }}
              </template>
              <template v-else>{{ $t("common.tasks") }}</template>
              <span class="opacity-80" v-if="eligibleTasks.length > 1">
                <template v-if="showTaskSelection">
                  ({{ selectedTaskNames.size }} / {{ eligibleTasks.length }})
                </template>
                <template v-else> ({{ eligibleTasks.length }}) </template>
              </span>
            </label>
            <div v-if="showTaskSelection" class="flex gap-x-2">
              <NButton
                size="tiny"
                :disabled="selectedTaskNames.size === eligibleTasks.length"
                @click="selectAllTasks"
              >
                {{ $t("sql-review.select-all") }}
              </NButton>
              <NButton
                size="tiny"
                :disabled="selectedTaskNames.size === 0"
                @click="selectNoTasks"
              >
                {{ "Clear" }}
              </NButton>
            </div>
          </div>
          <div class="flex-1 overflow-y-auto">
            <template v-if="useVirtualScroll">
              <NVirtualList
                :items="eligibleTasks"
                :item-size="itemHeight"
                class="max-h-64"
                item-resizable
              >
                <template #default="{ item: task }">
                  <div
                    :key="task.name"
                    class="flex items-center text-sm"
                    :style="{ height: `${itemHeight}px` }"
                  >
                    <NCheckbox
                      v-if="showTaskSelection"
                      :checked="selectedTaskNames.has(task.name)"
                      @update:checked="
                        (checked) => handleTaskSelectionChange(task, checked)
                      "
                      class="mr-2"
                    />
                    <NTag
                      v-if="semanticTaskType(task.type)"
                      class="mr-2"
                      size="small"
                    >
                      <span class="inline-block text-center">
                        {{ semanticTaskType(task.type) }}
                      </span>
                    </NTag>
                    <TaskDatabaseName :task="task" />
                  </div>
                </template>
              </NVirtualList>
            </template>
            <template v-else>
              <NScrollbar class="max-h-64">
                <ul class="text-sm space-y-2">
                  <li
                    v-for="task in eligibleTasks"
                    :key="task.name"
                    class="flex items-center"
                  >
                    <NCheckbox
                      v-if="showTaskSelection"
                      :checked="selectedTaskNames.has(task.name)"
                      @update:checked="
                        (checked) => handleTaskSelectionChange(task, checked)
                      "
                      class="mr-2"
                    />
                    <NTag
                      v-if="semanticTaskType(task.type)"
                      class="mr-2"
                      size="small"
                    >
                      <span class="inline-block text-center">
                        {{ semanticTaskType(task.type) }}
                      </span>
                    </NTag>
                    <TaskDatabaseName :task="task" />
                  </li>
                </ul>
              </NScrollbar>
            </template>
          </div>
        </div>

        <div
          v-if="showScheduledTimePicker"
          class="flex flex-col gap-y-3 shrink-0"
        >
          <div class="flex items-center">
            <NCheckbox
              :checked="runTimeInMS === undefined"
              @update:checked="
                (checked) =>
                  (runTimeInMS = checked
                    ? undefined
                    : Date.now() + DEFAULT_RUN_DELAY_MS)
              "
            >
              {{ $t("task.run-immediately") }}
            </NCheckbox>
          </div>
          <div v-if="runTimeInMS !== undefined" class="flex flex-col gap-y-1">
            <p class="font-medium text-control">
              {{ $t("task.scheduled-time", { count: runnableTasks.length }) }}
            </p>
            <NDatePicker
              v-model:value="runTimeInMS"
              type="datetime"
              :placeholder="$t('task.select-scheduled-time')"
              :is-date-disabled="
                (date: number) => date < dayjs().startOf('day').valueOf()
              "
              format="yyyy-MM-dd HH:mm:ss"
              clearable
            />
          </div>
        </div>

        <div class="flex flex-col gap-y-1 shrink-0">
          <p class="font-medium text-control">
            {{ $t("common.comment") }}
          </p>
          <NInput
            v-model:value="comment"
            type="textarea"
            :placeholder="$t('issue.leave-a-comment')"
            :autosize="{
              minRows: 3,
              maxRows: 10,
            }"
          />
        </div>
      </div>
    </template>
    <template #footer>
      <div class="w-full flex flex-row justify-between items-center gap-x-2">
        <div class="flex justify-end gap-x-3 w-full">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>

          <NTooltip :disabled="confirmErrors.length === 0" placement="top">
            <template #trigger>
              <NButton
                :disabled="confirmErrors.length > 0"
                type="primary"
                @click="handleConfirm"
              >
                <template v-if="action === 'RUN'">{{
                  $t("common.run")
                }}</template>
                <template v-else-if="action === 'SKIP'">{{
                  $t("common.skip")
                }}</template>
                <template v-else-if="action === 'CANCEL'">{{
                  $t("common.cancel")
                }}</template>
              </NButton>
            </template>
            <template #default>
              <ErrorList :errors="confirmErrors" />
            </template>
          </NTooltip>
        </div>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import Long from "long";
import {
  NButton,
  NCheckbox,
  NDatePicker,
  NInput,
  NScrollbar,
  NTag,
  NTooltip,
  NVirtualList,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { semanticTaskType } from "@/components/IssueV1";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import { ErrorList } from "@/components/IssueV1/components/common";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { pushNotification, useEnvironmentV1Store } from "@/store";
import {
  BatchRunTasksRequestSchema,
  BatchSkipTasksRequestSchema,
  BatchCancelTaskRunsRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Stage, Task, TaskRun } from "@/types/proto/v1/rollout_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import TaskDatabaseName from "./TaskDatabaseName.vue";

// Default delay for running tasks if not scheduled immediately.
const DEFAULT_RUN_DELAY_MS = 60000;

type LocalState = {
  loading: boolean;
};

export type TargetType =
  | { type: "tasks"; tasks?: Task[]; stage: Stage }
  | { type: "taskRuns"; taskRuns: TaskRun[]; stage: Stage };

const props = defineProps<{
  action: "RUN" | "SKIP" | "CANCEL";
  target: TargetType;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});
const environmentStore = useEnvironmentV1Store();
const comment = ref("");
const runTimeInMS = ref<number | undefined>(undefined);
const selectedTaskNames = ref<Set<string>>(new Set());

// Extract stage from target
const targetStage = computed(() => {
  return props.target.stage;
});

// Extract tasks if provided directly
const targetTasks = computed(() => {
  if (props.target.type === "tasks") {
    return props.target.tasks;
  }
  return undefined;
});

// Extract task runs if provided
const targetTaskRuns = computed(() => {
  if (props.target.type === "taskRuns") {
    return props.target.taskRuns;
  }
  return undefined;
});

const title = computed(() => {
  switch (props.action) {
    case "RUN":
      return t("common.run");
    case "SKIP":
      return t("common.skip");
    case "CANCEL":
      return t("common.cancel");
    default:
      return "";
  }
});

// Get eligible tasks based on action type and target
const eligibleTasks = computed(() => {
  // If specific tasks are provided, use them
  if (
    props.target.type === "tasks" &&
    targetTasks.value &&
    targetTasks.value.length > 0
  ) {
    return targetTasks.value;
  }

  // Otherwise filter from stage tasks based on action
  const stageTasks = targetStage.value.tasks || [];

  if (props.action === "RUN") {
    return stageTasks.filter(
      (task) =>
        task.status === Task_Status.NOT_STARTED ||
        task.status === Task_Status.PENDING ||
        task.status === Task_Status.FAILED
    );
  } else if (props.action === "SKIP") {
    return stageTasks.filter(
      (task) =>
        task.status === Task_Status.NOT_STARTED ||
        task.status === Task_Status.FAILED ||
        task.status === Task_Status.CANCELED
    );
  } else if (props.action === "CANCEL") {
    return stageTasks.filter(
      (task) =>
        task.status === Task_Status.PENDING ||
        task.status === Task_Status.RUNNING
    );
  }

  return [];
});

// Show task selection UI when there are multiple eligible tasks and no specific tasks provided
const showTaskSelection = computed(() => {
  return (
    eligibleTasks.value.length > 1 &&
    props.target.type === "tasks" &&
    !targetTasks.value
  );
});

// Get the tasks that will actually be run (selected or all)
const runnableTasks = computed(() => {
  if (showTaskSelection.value && selectedTaskNames.value.size > 0) {
    return eligibleTasks.value.filter((task) =>
      selectedTaskNames.value.has(task.name)
    );
  }
  return eligibleTasks.value;
});

// Virtual scroll configuration
const useVirtualScroll = computed(() => eligibleTasks.value.length > 50);
const itemHeight = computed(() => (showTaskSelection.value ? 40 : 32)); // Height of each task item in pixels

const showScheduledTimePicker = computed(() => {
  return props.action === "RUN";
});

const confirmErrors = computed(() => {
  const errors: string[] = [];

  if (runnableTasks.value.length === 0) {
    if (showTaskSelection.value && eligibleTasks.value.length > 0) {
      errors.push("Please select at least one task");
    } else {
      errors.push(t("common.no-data"));
    }
  }

  // Validate scheduled time if not running immediately (only for RUN)
  if (props.action === "RUN" && runTimeInMS.value !== undefined) {
    if (runTimeInMS.value <= Date.now()) {
      errors.push(t("task.error.scheduled-time-must-be-in-the-future"));
    }
  }

  // For CANCEL, we need task runs
  if (
    props.action === "CANCEL" &&
    (!targetTaskRuns.value || targetTaskRuns.value.length === 0)
  ) {
    errors.push("No active task runs to cancel");
  }

  return errors;
});

const handleConfirm = async () => {
  state.loading = true;
  try {
    if (props.action === "RUN") {
      // Prepare the request parameters
      const requestParams: any = {
        parent: targetStage.value.name,
        tasks: runnableTasks.value.map((task) => task.name),
        reason: comment.value,
      };

      if (runTimeInMS.value !== undefined) {
        // Convert timestamp to protobuf Timestamp format
        const runTimeSeconds = Math.floor(runTimeInMS.value / 1000);
        const runTimeNanos = (runTimeInMS.value % 1000) * 1000000;
        requestParams.runTime = {
          seconds: Long.fromNumber(runTimeSeconds),
          nanos: runTimeNanos,
        };
      }

      const request = create(BatchRunTasksRequestSchema, requestParams);
      await rolloutServiceClientConnect.batchRunTasks(request);
    } else if (props.action === "SKIP") {
      const request = create(BatchSkipTasksRequestSchema, {
        parent: targetStage.value.name,
        tasks: runnableTasks.value.map((task) => task.name),
        reason: comment.value,
      });
      await rolloutServiceClientConnect.batchSkipTasks(request);
    } else if (props.action === "CANCEL") {
      const request = create(BatchCancelTaskRunsRequestSchema, {
        parent: `${targetStage.value.name}/tasks/-`,
        taskRuns: targetTaskRuns.value?.map((taskRun) => taskRun.name) || [],
        reason: comment.value,
      });
      await rolloutServiceClientConnect.batchCancelTaskRuns(request);
    }

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.success"),
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: String(error),
    });
  } finally {
    state.loading = false;
    emit("close");
  }
};

const handleTaskSelectionChange = (task: Task, checked: boolean) => {
  if (checked) {
    selectedTaskNames.value.add(task.name);
  } else {
    selectedTaskNames.value.delete(task.name);
  }
  // Trigger reactivity
  selectedTaskNames.value = new Set(selectedTaskNames.value);
};

const selectAllTasks = () => {
  selectedTaskNames.value = new Set(eligibleTasks.value.map((t) => t.name));
};

const selectNoTasks = () => {
  selectedTaskNames.value.clear();
  selectedTaskNames.value = new Set();
};

const resetState = () => {
  comment.value = "";
  runTimeInMS.value = undefined;
  selectedTaskNames.value.clear();

  // Pre-select all tasks by default when showing selection
  if (showTaskSelection.value) {
    selectAllTasks();
  }
};
</script>
