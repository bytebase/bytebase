<template>
  <CommonDrawer
    :show="show"
    :title="title"
    :loading="loading"
    @show="resetState"
    @close="$emit('close')"
  >
    <template #default>
      <div class="flex flex-col gap-y-4 h-full px-1">
        <!-- Error Alert -->
        <NAlert
          v-if="validationErrors.length > 0"
          type="error"
          :title="$t('rollout.task-execution-errors')"
        >
          <ul class="list-disc list-inside flex flex-col gap-y-1">
            <li
              v-for="(error, index) in validationErrors"
              :key="index"
              class="text-sm"
            >
              {{ error }}
            </li>
          </ul>
        </NAlert>

        <!-- Stage information -->
        <div
          v-if="shouldShowStageInfo"
          class="flex flex-row gap-x-2 shrink-0 overflow-y-hidden justify-start items-center"
        >
          <label class="font-medium text-control">
            {{ $t("common.stage") }}
          </label>
          <EnvironmentV1Name
            :environment="
              environmentStore.getEnvironmentByName(
                target.stage?.environment ?? ''
              )
            "
            :link="false"
          />
        </div>

        <!-- Task information -->
        <div v-if="shouldShowTaskInfo" class="flex flex-col gap-y-1 shrink-0">
          <label class="text-control">
            <span class="font-medium">{{
              $t("common.task", eligibleTasks.length)
            }}</span>
            <span v-if="taskCountSuffix" class="opacity-80">
              ({{ taskCountSuffix }})
            </span>
          </label>
          <!-- Virtual scroll for large lists, regular scroll for small -->
          <NVirtualList
            v-if="eligibleTasks.length > 50"
            :items="eligibleTasks"
            :item-size="32"
            class="max-h-64"
            item-resizable
          >
            <template #default="{ item: task }">
              <div
                :key="task.name"
                class="flex items-center text-sm gap-2 h-8"
              >
                <TaskStatus :status="task.status" size="small" />
                <TaskDatabaseName :task="task" />
              </div>
            </template>
          </NVirtualList>
          <NScrollbar v-else class="max-h-64">
            <ul class="text-sm flex flex-col gap-y-2">
              <li
                v-for="task in eligibleTasks"
                :key="task.name"
                class="flex items-center gap-2"
              >
                <TaskStatus :status="task.status" size="small" />
                <TaskDatabaseName :task="task" />
              </li>
            </ul>
          </NScrollbar>
        </div>

        <!-- Scheduled time picker (RUN only) -->
        <div v-if="action === 'RUN'" class="flex flex-col">
          <h3 class="font-medium text-control mb-1">
            {{ $t("task.execution-time") }}
          </h3>
          <NRadioGroup
            :size="'large'"
            :value="runTimeInMS === undefined ? 'immediate' : 'scheduled'"
            class="flex! flex-col sm:flex-row gap-2 sm:gap-4"
            @update:value="handleExecutionModeChange"
          >
            <NRadio value="immediate">
              <span>{{ $t("task.run-immediately.self") }}</span>
            </NRadio>
            <NRadio value="scheduled">
              <span>{{ $t("task.schedule-for-later.self") }}</span>
            </NRadio>
          </NRadioGroup>

          <div class="mt-1 text-sm text-control-light">
            {{
              runTimeInMS === undefined
                ? $t("task.run-immediately.description")
                : $t("task.schedule-for-later.description")
            }}
          </div>

          <NDatePicker
            v-if="runTimeInMS !== undefined"
            v-model:value="runTimeInMS"
            type="datetime"
            :placeholder="$t('task.select-scheduled-time')"
            :is-date-disabled="isDateDisabled"
            format="yyyy-MM-dd HH:mm:ss"
            :actions="['clear', 'confirm']"
            clearable
            class="mt-2"
          />
        </div>

        <!-- Comment field (SKIP only) -->
        <div
          v-if="action === 'SKIP' && issue"
          class="flex flex-col gap-y-1 shrink-0"
        >
          <p class="font-medium text-control">
            {{ $t("common.reason") }}
          </p>
          <NInput
            v-model:value="comment"
            type="textarea"
            :placeholder="$t('issue.leave-a-comment')"
            :autosize="{ minRows: 3, maxRows: 10 }"
            :maxlength="1000"
          />
        </div>
      </div>
    </template>
    <template #footer>
      <div class="w-full flex flex-row justify-end items-center gap-x-2">
        <NButton quaternary @click="$emit('close')">
          {{ $t("common.close") }}
        </NButton>

        <NTooltip :disabled="validationErrors.length === 0" placement="top">
          <template #trigger>
            <NButton
              :disabled="validationErrors.length > 0"
              type="primary"
              @click="handleConfirm"
            >
              {{ confirmButtonText }}
            </NButton>
          </template>
          <template #default>
            <ErrorList :errors="validationErrors" />
          </template>
        </NTooltip>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import {
  NAlert,
  NButton,
  NDatePicker,
  NInput,
  NRadio,
  NRadioGroup,
  NScrollbar,
  NTooltip,
  NVirtualList,
} from "naive-ui";
import { computed, ref, watch, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { usePlanContextWithRollout } from "@/components/Plan/logic";
import { projectOfPlan } from "@/components/Plan/logic/utils";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import {
  CANCELABLE_TASK_STATUSES,
  RUNNABLE_TASK_STATUSES,
} from "@/components/RolloutV1/constants/task";
import { EnvironmentV1Name } from "@/components/v2";
import { rolloutServiceClientConnect } from "@/connect";
import {
  pushNotification,
  useCurrentUserV1,
  useEnvironmentV1Store,
} from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type {
  BatchRunTasksRequest,
  Stage,
  Task,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  BatchCancelTaskRunsRequestSchema,
  BatchRunTasksRequestSchema,
  BatchSkipTasksRequestSchema,
  ListTaskRunsRequestSchema,
  Task_Type,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { extractStageUID } from "@/utils";
import CommonDrawer from "./CommonDrawer.vue";
import ErrorList from "./ErrorList.vue";
import TaskDatabaseName from "./TaskDatabaseName.vue";
import { canRolloutTasks } from "./taskPermissions";

// 1 hour default delay for scheduled tasks
const DEFAULT_RUN_DELAY_MS = 60 * 60 * 1000;

export type TargetType = { type: "tasks"; tasks?: Task[]; stage?: Stage };

const props = defineProps<{
  show: boolean;
  action: "RUN" | "SKIP" | "CANCEL";
  target: TargetType;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "confirm"): void;
}>();

const { t } = useI18n();
const { issue, rollout, plan } = usePlanContextWithRollout();
const currentUser = useCurrentUserV1();
const environmentStore = useEnvironmentV1Store();

// State
const loading = ref(false);
const comment = ref("");
const runTimeInMS = ref<number | undefined>(undefined);
const canRolloutPermission = ref(true);

// Task status filters
const isRunnable = (task: Task) => RUNNABLE_TASK_STATUSES.includes(task.status);

const isCancellable = (task: Task) =>
  CANCELABLE_TASK_STATUSES.includes(task.status);

// Rollout type detection
const allRolloutTasks = computed(() =>
  rollout.value.stages.flatMap((stage) => stage.tasks)
);

const rolloutType = computed(() => {
  const tasks = allRolloutTasks.value;
  if (tasks.every((task) => task.type === Task_Type.DATABASE_CREATE)) {
    return "DATABASE_CREATE";
  }
  if (tasks.every((task) => task.type === Task_Type.DATABASE_EXPORT)) {
    return "DATABASE_EXPORT";
  }
  return "DATABASE_CHANGE";
});

const isDatabaseCreateOrExport = computed(
  () =>
    rolloutType.value === "DATABASE_CREATE" ||
    rolloutType.value === "DATABASE_EXPORT"
);

// Eligible tasks based on action type
const eligibleTasks = computed(() => {
  // If specific tasks are provided, use them (e.g., from per-task actions)
  // Otherwise, for DATABASE_CREATE/EXPORT use all rollout tasks,
  // for regular rollouts use stage tasks
  const tasks = props.target.tasks
    ? props.target.tasks
    : isDatabaseCreateOrExport.value
      ? allRolloutTasks.value
      : (props.target.stage?.tasks ?? []);

  if (props.action === "RUN" || props.action === "SKIP") {
    return tasks.filter(isRunnable);
  }
  if (props.action === "CANCEL") {
    return tasks.filter(isCancellable);
  }
  return tasks;
});

// UI visibility
const shouldShowStageInfo = computed(() => !isDatabaseCreateOrExport.value);
const shouldShowTaskInfo = computed(
  () => rolloutType.value !== "DATABASE_CREATE"
);

const taskCountSuffix = computed(() => {
  if (
    eligibleTasks.value.length <= 1 ||
    !shouldShowStageInfo.value ||
    !props.target.stage
  ) {
    return "";
  }
  const total = props.target.stage.tasks.length;
  const eligible = eligibleTasks.value.length;
  return eligible === total ? String(eligible) : `${eligible} / ${total}`;
});

const title = computed(() => {
  const n = eligibleTasks.value.length;
  if (props.action === "RUN") return t("task.run-task", { n });
  if (props.action === "SKIP") return t("task.skip-task", { n });
  if (props.action === "CANCEL") return t("task.cancel-task", { n });
  return "";
});

const confirmButtonText = computed(() => {
  if (props.action === "RUN") return t("common.run");
  if (props.action === "SKIP") return t("common.skip");
  return t("common.cancel");
});

// Validation
const validationErrors = computed(() => {
  const errors: string[] = [];

  if (eligibleTasks.value.length === 0) {
    errors.push(t("common.no-data"));
  }

  if (!canRolloutPermission.value) {
    const isExportByNonCreator =
      issue.value?.type === Issue_Type.DATABASE_EXPORT &&
      issue.value.creator !== `${userNamePrefix}${currentUser.value.email}`;
    errors.push(
      t(
        isExportByNonCreator
          ? "task.data-export-creator-only"
          : "task.no-permission"
      )
    );
  }

  if (
    props.action === "RUN" &&
    runTimeInMS.value !== undefined &&
    runTimeInMS.value <= Date.now()
  ) {
    errors.push(t("task.error.scheduled-time-must-be-in-the-future"));
  }

  return errors;
});

// Cache permission check - only update when panel opens
watch(
  () => props.show,
  (show) => {
    if (show) {
      const tasks = props.target.tasks ?? props.target.stage?.tasks ?? [];
      canRolloutPermission.value = canRolloutTasks(tasks, issue.value);
    }
  },
  { immediate: true }
);

// Initialize run time from existing task schedule
watchEffect(() => {
  const runTimes = new Set(eligibleTasks.value.map((task) => task.runTime));
  if (runTimes.size !== 1) return;

  const runTime = [...runTimes][0];
  if (!runTime) return;

  runTimeInMS.value = Number(runTime.seconds) * 1000 + runTime.nanos / 1000000;
});

// Helpers
const isDateDisabled = (date: number) =>
  date < dayjs().startOf("day").valueOf();

const handleExecutionModeChange = (value: string) => {
  runTimeInMS.value =
    value === "immediate" ? undefined : Date.now() + DEFAULT_RUN_DELAY_MS;
};

const resetState = () => {
  comment.value = "";
  runTimeInMS.value = undefined;
};

const groupTasksByStage = (tasks: Task[]) => {
  const map = new Map<string, Task[]>();
  for (const task of tasks) {
    const stageId = extractStageUID(task.name);
    if (!stageId) continue; // Skip tasks with invalid names
    if (!map.has(stageId)) map.set(stageId, []);
    map.get(stageId)!.push(task);
  }
  return map;
};

const addRunTimeToRequest = (request: BatchRunTasksRequest) => {
  if (runTimeInMS.value === undefined) return;

  const seconds = Math.floor(runTimeInMS.value / 1000);
  const nanos = (runTimeInMS.value % 1000) * 1000000;
  request.runTime = create(TimestampSchema, {
    seconds: BigInt(seconds),
    nanos,
  });
};

// Actions
const runTasks = async () => {
  const tasksByStage = groupTasksByStage(eligibleTasks.value);
  for (const [stageId, tasks] of tasksByStage) {
    const request = create(BatchRunTasksRequestSchema, {
      parent: `${rollout.value.name}/stages/${stageId}`,
      tasks: tasks.map((task) => task.name),
    });
    addRunTimeToRequest(request);
    await rolloutServiceClientConnect.batchRunTasks(request);
  }
};

const skipTasks = async () => {
  const tasksByStage = groupTasksByStage(eligibleTasks.value);
  for (const [stageId, tasks] of tasksByStage) {
    const request = create(BatchSkipTasksRequestSchema, {
      parent: `${rollout.value.name}/stages/${stageId}`,
      tasks: tasks.map((task) => task.name),
      reason: comment.value,
    });
    await rolloutServiceClientConnect.batchSkipTasks(request);
  }
};

const cancelTasks = async () => {
  const tasksByStage = groupTasksByStage(eligibleTasks.value);
  const cancelableRuns = new Map<string, string[]>();

  // Fetch cancelable task runs for each stage
  for (const [stageId, tasks] of tasksByStage) {
    const taskNames = new Set(tasks.map((t) => t.name));
    const parent = `${rollout.value.name}/stages/${stageId}/tasks/-`;
    const response = await rolloutServiceClientConnect.listTaskRuns(
      create(ListTaskRunsRequestSchema, { parent })
    );

    const runs = (response.taskRuns || [])
      .filter((run) => {
        const taskName = run.name.split("/taskRuns/")[0];
        return (
          taskNames.has(taskName) &&
          (run.status === TaskRun_Status.PENDING ||
            run.status === TaskRun_Status.RUNNING)
        );
      })
      .map((run) => run.name);

    if (runs.length > 0) {
      cancelableRuns.set(`${rollout.value.name}/stages/${stageId}`, runs);
    }
  }

  // Cancel in parallel
  await Promise.all(
    Array.from(cancelableRuns.entries()).map(([stageName, runNames]) =>
      rolloutServiceClientConnect.batchCancelTaskRuns(
        create(BatchCancelTaskRunsRequestSchema, {
          parent: `${stageName}/tasks/-`,
          taskRuns: runNames,
        })
      )
    )
  );
};

const handleConfirm = async () => {
  if (loading.value) return;

  loading.value = true;
  try {
    if (props.action === "RUN") await runTasks();
    else if (props.action === "SKIP") await skipTasks();
    else if (props.action === "CANCEL") await cancelTasks();

    emit("confirm");
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: String(error),
    });
  } finally {
    loading.value = false;
    emit("close");
  }
};
</script>
