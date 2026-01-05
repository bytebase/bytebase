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

        <!-- Warning Alert -->
        <NAlert
          v-if="validationWarnings.length > 0"
          type="warning"
          :title="$t('rollout.task-execution-notices')"
        >
          <ul class="list-disc list-inside flex flex-col gap-y-1">
            <li
              v-for="(warning, index) in validationWarnings"
              :key="index"
              class="text-sm"
            >
              {{ warning }}
            </li>
          </ul>
        </NAlert>

        <template v-if="stage">
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
              <ul class="text-sm flex flex-col gap-y-2">
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
          label-class="text-base!"
          :allow-run-checks="false"
          :plan-name="issue.plan"
          :plan-check-run-list="planCheckRunList"
          :database="database"
        />

        <!-- Only show comment/reason field for SKIP action -->
        <div v-if="action === 'SKIP'" class="flex flex-col gap-y-1 shrink-0">
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
import dayjs from "dayjs";
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
import { planCheckRunSummaryForCheckRunList } from "@/components/PlanCheckRun/common";
import PlanCheckRunBar from "@/components/PlanCheckRun/PlanCheckRunBar.vue";
import { EnvironmentV1Name } from "@/components/v2";
import { rolloutServiceClientConnect } from "@/connect";
import {
  pushNotification,
  useCurrentProjectV1,
  useEnvironmentV1Store,
} from "@/store";
import { Issue_Approver_Status } from "@/types/proto-es/v1/issue_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  BatchCancelTaskRunsRequestSchema,
  BatchRunTasksRequestSchema,
  BatchSkipTasksRequestSchema,
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

// Check issue approval status
const issueApprovalStatus = computed(() => {
  if (!issue?.value || !props.action) {
    return { isApproved: true, hasIssue: false };
  }

  const currentIssue = issue.value;
  const approvalTemplate = currentIssue.approvalTemplate;

  // Check if issue has approval template
  if (!approvalTemplate || (approvalTemplate.flow?.roles || []).length === 0) {
    return { isApproved: true, hasIssue: true };
  }

  // Simple check based on all approvers being approved (no rejections)
  const hasRejections = currentIssue.approvers.some(
    (app) => app.status === Issue_Approver_Status.REJECTED
  );

  if (hasRejections) {
    return { isApproved: false, hasIssue: true, status: "rejected" };
  }

  // Check if all required approvals are present
  const approvedCount = currentIssue.approvers.filter(
    (app) => app.status === Issue_Approver_Status.APPROVED
  ).length;

  const requiredApprovalsCount = approvalTemplate.flow?.roles?.length || 0;

  const isApproved = approvedCount >= requiredApprovalsCount;

  return {
    isApproved,
    hasIssue: true,
    status: isApproved ? "approved" : "pending",
  };
});

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
  // Only show bypass option when there are warnings but NO errors
  return (
    validationWarnings.value.length > 0 && validationErrors.value.length === 0
  );
});

const planCheckRunList = computed(() => {
  const list = filteredTasks.value.flatMap(getPlanCheckRunsForTask);
  return uniqBy(list, (checkRun) => checkRun.name);
});

// Plan check error validation based on project settings
const planCheckError = computed(() => {
  if (
    !(
      props.action === "ROLLOUT" ||
      props.action === "RETRY" ||
      props.action === "RESTART"
    )
  ) {
    return undefined;
  }

  const summary = planCheckRunSummaryForCheckRunList(planCheckRunList.value);

  // If no enforcement is specified, default to no validation
  if (!project.value.requirePlanCheckNoError) {
    return undefined;
  }

  if (summary.runningCount > 0) {
    return t(
      "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
    );
  }

  if (summary.errorCount > 0) {
    return t(
      "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
    );
  }

  return undefined;
});

// Plan check warning validation for non-blocking plan check results
const planCheckWarning = computed(() => {
  if (
    !(
      props.action === "ROLLOUT" ||
      props.action === "RETRY" ||
      props.action === "RESTART"
    )
  ) {
    return undefined;
  }

  const summary = planCheckRunSummaryForCheckRunList(planCheckRunList.value);
  if (
    summary.successCount +
      summary.warnCount +
      summary.errorCount +
      summary.runningCount ===
    0
  ) {
    return undefined;
  }

  // If enforcement is disabled, show any plan check issues as warnings
  if (!project.value.requirePlanCheckNoError) {
    if (summary.runningCount > 0) {
      return t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
      );
    }
    if (summary.errorCount > 0 || summary.warnCount > 0) {
      return t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
      );
    }
  }

  return undefined;
});

// Collect blocking validation errors
const validationErrors = computed(() => {
  if (!props.action) return [];

  const errors: string[] = [];

  // Permission errors - always block rollout
  if (filteredTasks.value.length > 0) {
    // Check basic task permissions here if needed
  }

  // No active tasks to cancel - blocking error
  if (
    props.action === "CANCEL" &&
    filteredTasks.value.length > 0 &&
    !filteredTasks.value.some(
      (task) =>
        task.status === Task_Status.PENDING ||
        task.status === Task_Status.RUNNING
    )
  ) {
    errors.push(t("rollout.no-active-task-to-cancel"));
  }

  if (
    props.action === "ROLLOUT" ||
    props.action === "RETRY" ||
    props.action === "RESTART"
  ) {
    // Issue approval errors (only if policy requires it) - HARD BLOCK
    const requiresIssueApproval = project.value.requireIssueApproval;

    if (
      requiresIssueApproval &&
      issueApprovalStatus.value.hasIssue &&
      !issueApprovalStatus.value.isApproved
    ) {
      const isRejected = issueApprovalStatus.value.status === "rejected";
      errors.push(
        isRejected
          ? t("issue.approval.rejected-error")
          : t("issue.approval.pending-error")
      );
    }

    // Plan check errors (based on rollout policy) - HARD BLOCK
    if (planCheckError.value) {
      errors.push(planCheckError.value);
    }
  }

  return errors;
});

// Collect validation warnings that don't block rollout
const validationWarnings = computed(() => {
  if (!props.action) return [];

  const warnings: string[] = [];

  // Basic validation warnings
  if (filteredTasks.value.length === 0) {
    warnings.push(t("common.no-data"));
  }

  if (
    props.action === "ROLLOUT" ||
    props.action === "RETRY" ||
    props.action === "RESTART"
  ) {
    // Validate scheduled time if not running immediately
    if (runTimeInMS.value !== undefined && runTimeInMS.value <= Date.now()) {
      warnings.push(t("task.error.scheduled-time-must-be-in-the-future"));
    }

    // Plan check warnings (non-blocking plan check results)
    if (planCheckWarning.value) {
      warnings.push(planCheckWarning.value);
    }

    // Issue approval warnings (when not required by policy but issue is not approved)
    const requiresIssueApproval = project.value.requireIssueApproval;

    if (
      !requiresIssueApproval &&
      issueApprovalStatus.value.hasIssue &&
      !issueApprovalStatus.value.isApproved
    ) {
      const isRejected = issueApprovalStatus.value.status === "rejected";
      warnings.push(
        isRejected
          ? t("issue.approval.rejected-error")
          : t("issue.approval.pending-error")
      );
    }

    // Previous stages incomplete warning
    if (hasPreviousUnrolledStages.value) {
      warnings.push(
        t("issue.status-transition.warning.some-previous-stages-are-not-done")
      );
    }
  }

  return warnings;
});

const confirmErrors = computed(() => {
  const errors: string[] = [];

  // Always block on validation errors
  if (validationErrors.value.length > 0) {
    errors.push(...validationErrors.value);
  }

  // Block on warnings unless bypassed
  if (!performActionAnyway.value && validationWarnings.value.length > 0) {
    errors.push(...validationWarnings.value);
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
