<template>
  <CommonDrawer
    :show="action !== undefined"
    :title="title"
    :loading="state.loading"
    @show="resetState"
    @close="$emit('close')"
  >
    <template #default>
      <div
        v-if="action"
        class="flex flex-col gap-y-4 h-full overflow-y-hidden px-1"
      >
        <template v-if="stage">
          <NAlert
            v-if="hasPreviousUnrolledStages"
            :title="
              t(
                'issue.status-transition.warning.some-previous-stages-are-not-done'
              )
            "
            type="warning"
          />
          <div
            class="flex flex-row gap-x-2 shrink-0 overflow-y-hidden justify-start items-center"
          >
            <label class="font-medium text-control">
              {{ $t("common.stage") }}
            </label>
            <EnvironmentV1Name
              :environment="
                environmentStore.getEnvironmentByName(stage.environment)
              "
              :link="false"
            />
          </div>
        </template>
        <div
          class="flex flex-col gap-y-1 shrink overflow-y-hidden justify-start"
        >
          <label class="font-medium text-control">
            {{ $t("common.task", filteredTasks.length) }}
            <span class="font-mono opacity-80" v-if="filteredTasks.length > 1"
              >({{ filteredTasks.length }})</span
            >
          </label>
          <div class="flex-1 overflow-y-auto">
            <NScrollbar class="max-h-64">
              <ul class="text-sm space-y-2">
                <li
                  v-for="task in filteredTasks"
                  :key="task.name"
                  class="flex items-center"
                >
                  <NTag
                    v-if="semanticTaskType(task.type)"
                    class="mr-2"
                    size="small"
                  >
                    <span class="inline-block text-center">
                      {{ semanticTaskType(task.type) }}
                    </span>
                  </NTag>
                  <RolloutTaskDatabaseName :task="task" />
                </li>
              </ul>
            </NScrollbar>
          </div>
        </div>

        <PlanCheckRunBar
          v-if="
            (action === 'ROLLOUT' ||
              action === 'RETRY' ||
              action === 'RESTART') &&
            planCheckRunList.length > 0
          "
          class="shrink-0 flex-col gap-y-1"
          label-class="!text-base"
          :allow-run-checks="false"
          :plan-name="issue.plan"
          :plan-check-run-list="planCheckRunList"
          :database="database"
        />

        <div v-if="planCheckErrors.length > 0" class="flex flex-col">
          <ErrorList :errors="planCheckErrors" :bullets="false" class="text-sm">
            <template #prefix>
              <heroicons:exclamation-triangle
                class="text-warning w-4 h-4 inline-block mr-1 mb-px"
              />
            </template>
          </ErrorList>
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
              {{ $t("task.run-immediately.self") }}
            </NCheckbox>
          </div>
          <div v-if="runTimeInMS !== undefined" class="flex flex-col gap-y-1">
            <p class="font-medium text-control">
              {{ $t("task.scheduled-time", filteredTasks.length) }}
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
      </div>
    </template>
    <template #footer>
      <div
        v-if="action"
        class="w-full flex flex-row justify-between items-center gap-x-2"
      >
        <div>
          <NCheckbox
            v-if="showPerformActionAnyway"
            v-model:checked="performActionAnyway"
          >
            {{
              $t("issue.action-anyway", {
                action: taskRolloutActionDialogButtonName(action, taskList),
              })
            }}
          </NCheckbox>
        </div>

        <div class="flex justify-end gap-x-3">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>

          <NTooltip :disabled="confirmErrors.length === 0" placement="top">
            <template #trigger>
              <NButton
                :disabled="confirmErrors.length > 0"
                v-bind="taskRolloutActionButtonProps(action)"
                @click="handleConfirm(action!)"
              >
                {{ taskRolloutActionDialogButtonName(action, taskList) }}
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
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { head, uniqBy } from "lodash-es";
import {
  NAlert,
  NButton,
  NCheckbox,
  NDatePicker,
  NInput,
  NScrollbar,
  NTag,
  NTooltip,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { TaskRolloutAction } from "@/components/IssueV1/logic";
import {
  semanticTaskType,
  stageForTask,
  taskRolloutActionButtonProps,
  taskRolloutActionDialogButtonName,
  taskRolloutActionDisplayName,
  taskRunListForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import PlanCheckRunBar from "@/components/PlanCheckRun/PlanCheckRunBar.vue";
import { planCheckRunSummaryForCheckRunList } from "@/components/PlanCheckRun/common";
import { EnvironmentV1Name } from "@/components/v2";
import { rolloutServiceClientConnect } from "@/grpcweb";
import {
  pushNotification,
  useEnvironmentV1Store,
  useCurrentProjectV1,
} from "@/store";
import {
  BatchRunTasksRequestSchema,
  BatchSkipTasksRequestSchema,
  BatchCancelTaskRunsRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask } from "@/utils";
import { ErrorList } from "../common";
import CommonDrawer from "./CommonDrawer.vue";
import RolloutTaskDatabaseName from "./RolloutTaskDatabaseName.vue";

// Default delay for running tasks if not scheduled immediately.
const DEFAULT_RUN_DELAY_MS = 60000;

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action?: TaskRolloutAction;
  taskList: Task[];
}>();
const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});
const { project } = useCurrentProjectV1();
const { issue, selectedTask, events, getPlanCheckRunsForTask } =
  useIssueContext();
const environmentStore = useEnvironmentV1Store();
const comment = ref("");
const runTimeInMS = ref<number | undefined>(undefined);
const performActionAnyway = ref(false);

const title = computed(() => {
  if (!props.action) return "";

  const action = taskRolloutActionDisplayName(props.action, selectedTask.value);
  if (filteredTasks.value.length > 1) {
    return t("task.action-all-tasks-in-current-stage", { action });
  }
  return action;
});

const database = computed(() =>
  databaseForTask(project.value, selectedTask.value)
);

const stage = computed(() => {
  const firstTask = head(props.taskList);
  if (!firstTask) return undefined;
  return stageForTask(issue.value, firstTask);
});

const showScheduledTimePicker = computed(() => {
  return (
    props.action === "ROLLOUT" ||
    props.action === "RETRY" ||
    props.action === "RESTART"
  );
});

const filteredTasks = computed(() => {
  let filteredTaskList = props.taskList;
  if (props.action === "RETRY") {
    // For RETRY action, we only want to retry tasks that are not done or skipped.
    filteredTaskList = filteredTaskList.filter(
      (task) => task.status === Task_Status.FAILED
    );
  } else if (props.action === "RESTART") {
    // For RESTART action, we only want to restart tasks that are running or pending.
    filteredTaskList = filteredTaskList.filter(
      (task) => task.status === Task_Status.CANCELED
    );
  } else if (props.action === "CANCEL") {
    // For CANCEL action, we only want to cancel tasks that are running or pending.
    filteredTaskList = filteredTaskList.filter((task) =>
      [Task_Status.RUNNING, Task_Status.PENDING].includes(task.status)
    );
  } else if (props.action === "SKIP") {
    // For SKIP action, we only want to skip tasks that are not done or skipped.
    filteredTaskList = filteredTaskList.filter(
      (task) => ![Task_Status.DONE, Task_Status.SKIPPED].includes(task.status)
    );
  } else if (props.action === "ROLLOUT") {
    filteredTaskList = filteredTaskList.filter(
      (task) =>
        ![
          Task_Status.DONE,
          Task_Status.SKIPPED,
          TaskRun_Status.RUNNING,
        ].includes(task.status)
    );
  }
  return filteredTaskList;
});

const hasPreviousUnrolledStages = computed(() => {
  if (!stage.value) return false;
  const stages = issue.value.rolloutEntity?.stages;
  if (!stages) return false;
  const stageIndex = stages.findIndex((s) => s.name === stage.value?.name);
  return stages.slice(0, stageIndex).some((s) =>
    s.tasks.some(
      // Done or skipped tasks are considered as rolled.
      // Otherwise, it's considered as unrolled.
      (t) => ![Task_Status.DONE, Task_Status.SKIPPED].includes(t.status)
    )
  );
});

const showPerformActionAnyway = computed(() => {
  return planCheckErrors.value.length > 0 || hasPreviousUnrolledStages.value;
});

const planCheckRunList = computed(() => {
  const list = filteredTasks.value.flatMap(getPlanCheckRunsForTask);
  return uniqBy(list, (checkRun) => checkRun.name);
});

const planCheckErrors = computed(() => {
  const errors: string[] = [];
  if (
    props.action === "ROLLOUT" ||
    props.action === "RETRY" ||
    props.action === "RESTART"
  ) {
    const summary = planCheckRunSummaryForCheckRunList(planCheckRunList.value);
    if (summary.errorCount > 0 || summary.warnCount) {
      errors.push(
        t(
          "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
        )
      );
    }
    if (summary.runningCount > 0) {
      errors.push(
        t(
          "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
        )
      );
    }
  }

  return errors;
});

const confirmErrors = computed(() => {
  const errors: string[] = [];
  if (!performActionAnyway.value) {
    if (planCheckErrors.value.length > 0) {
      errors.push(...planCheckErrors.value);
    }
    if (hasPreviousUnrolledStages.value) {
      errors.push(
        t("issue.status-transition.warning.some-previous-stages-are-not-done")
      );
    }
  }

  // Validate scheduled time if not running immediately
  if (runTimeInMS.value !== undefined) {
    if (runTimeInMS.value <= Date.now()) {
      errors.push(t("task.error.scheduled-time-must-be-in-the-future"));
    }
  }

  return errors;
});

const handleConfirm = async (action: TaskRolloutAction) => {
  state.loading = true;
  try {
    const stage = stageForTask(issue.value, props.taskList[0]);
    if (!stage) return;
    if (action === "ROLLOUT" || action === "RETRY" || action === "RESTART") {
      // Prepare the request parameters
      const request = create(BatchRunTasksRequestSchema, {
        parent: stage.name,
        tasks: filteredTasks.value.map((task) => task.name),
        reason: comment.value,
      });
      if (runTimeInMS.value !== undefined) {
        // Convert timestamp to protobuf Timestamp format
        const runTimeSeconds = Math.floor(runTimeInMS.value / 1000);
        const runTimeNanos = (runTimeInMS.value % 1000) * 1000000;
        request.runTime = create(TimestampSchema, {
          seconds: BigInt(runTimeSeconds),
          nanos: runTimeNanos,
        });
      }
      await rolloutServiceClientConnect.batchRunTasks(request);
    } else if (action === "SKIP") {
      const request = create(BatchSkipTasksRequestSchema, {
        parent: stage.name,
        tasks: filteredTasks.value.map((task) => task.name),
        reason: comment.value,
      });
      await rolloutServiceClientConnect.batchSkipTasks(request);
    } else if (action === "CANCEL") {
      const taskRunListToCancel = filteredTasks.value
        .map((task) => {
          const taskRunList = taskRunListForTask(issue.value, task);
          const currentRunningTaskRun = taskRunList.find(
            (taskRun) =>
              taskRun.status === TaskRun_Status.RUNNING ||
              taskRun.status === TaskRun_Status.PENDING
          );
          return currentRunningTaskRun;
        })
        .filter((taskRun) => taskRun !== undefined);
      if (taskRunListToCancel.length > 0) {
        const request = create(BatchCancelTaskRunsRequestSchema, {
          parent: `${stage.name}/tasks/-`,
          taskRuns: taskRunListToCancel.map((taskRun) => taskRun.name),
          reason: comment.value,
        });
        await rolloutServiceClientConnect.batchCancelTaskRuns(request);
      }
    }
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: title.value,
    });

    // notify the issue logic to update issue status
    events.emit("status-changed", { eager: true });
  } finally {
    state.loading = false;
    emit("close");
  }
};

const resetState = () => {
  comment.value = "";
  runTimeInMS.value = undefined;
  performActionAnyway.value = false;
};
</script>
