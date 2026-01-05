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

        <!-- Plan Check Status -->
        <div v-if="planCheckStatus.total > 0" class="flex items-center gap-3">
          <span class="font-medium text-control shrink-0">{{
            $t("plan.navigator.checks")
          }}</span>
          <PlanCheckStatusCount :plan="plan" />
        </div>

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
          <div class="flex items-center justify-between">
            <label class="text-control">
              <span class="font-medium">{{
                $t("common.task", eligibleTasks.length)
              }}</span>
              <span
                class="opacity-80"
                v-if="
                  eligibleTasks.length > 1 &&
                  shouldShowStageInfo &&
                  target.stage
                "
              >
                ({{
                  eligibleTasks.length === target.stage.tasks.length
                    ? eligibleTasks.length
                    : `${eligibleTasks.length} / ${target.stage.tasks.length}`
                }})
              </span>
            </label>
          </div>
          <div>
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
                    class="flex items-center text-sm gap-2"
                    :style="{ height: `${itemHeight}px` }"
                  >
                    <TaskStatus
                      :status="task.status"
                      size="small"
                    />
                    <TaskDatabaseName :task="task" />
                  </div>
                </template>
              </NVirtualList>
            </template>
            <template v-else>
              <NScrollbar class="max-h-64">
                <ul class="text-sm flex flex-col gap-y-2">
                  <li
                    v-for="task in eligibleTasks"
                    :key="task.name"
                    class="flex items-center gap-2"
                  >
                    <TaskStatus
                      :status="task.status"
                      size="small"
                    />
                    <TaskDatabaseName :task="task" />
                  </li>
                </ul>
              </NScrollbar>
            </template>
          </div>
        </div>

        <div v-if="showScheduledTimePicker" class="flex flex-col">
          <h3 class="font-medium text-control mb-1">
            {{ $t("task.execution-time") }}
          </h3>
          <NRadioGroup
            :size="'large'"
            :value="runTimeInMS === undefined ? 'immediate' : 'scheduled'"
            @update:value="handleExecutionModeChange"
            class="flex! flex-row gap-x-4"
          >
            <!-- Run Immediately Option -->
            <NRadio value="immediate">
              <span>{{ $t("task.run-immediately.self") }}</span>
            </NRadio>

            <!-- Schedule for Later Option -->
            <NRadio value="scheduled">
              <span>{{ $t("task.schedule-for-later.self") }}</span>
            </NRadio>
          </NRadioGroup>

          <!-- Description based on selection -->
          <div class="mt-1 text-sm text-control-light">
            <span v-if="runTimeInMS === undefined">
              {{ $t("task.run-immediately.description") }}
            </span>
            <span v-else>
              {{ $t("task.schedule-for-later.description") }}
            </span>
          </div>

          <!-- Scheduled Time Options -->
          <div v-if="runTimeInMS !== undefined" class="flex flex-col">
            <NDatePicker
              v-model:value="runTimeInMS"
              type="datetime"
              :placeholder="$t('task.select-scheduled-time')"
              :is-date-disabled="
                (date: number) => date < dayjs().startOf('day').valueOf()
              "
              format="yyyy-MM-dd HH:mm:ss"
              :actions="['clear', 'confirm']"
              clearable
              class="mt-2"
            />
          </div>
        </div>

        <div
          v-if="shouldShowComment"
          class="flex flex-col gap-y-1 shrink-0"
        >
          <p class="font-medium text-control">
            {{ $t("common.reason") }}
          </p>
          <NInput
            v-model:value="comment"
            type="textarea"
            :placeholder="$t('issue.leave-a-comment')"
            :autosize="{
              minRows: 3,
              maxRows: 10,
            }"
            :maxlength="1000"
          />
        </div>
      </div>
    </template>
    <template #footer>
      <div class="w-full flex flex-row justify-between items-center gap-x-2">
        <!-- Bypass stage requirements checkbox -->
        <div v-if="shouldShowBypassOption" class="flex items-center">
          <NCheckbox v-model:checked="bypassPolicyChecks" :disabled="loading">
            {{ $t("rollout.bypass-stage-requirements") }}
          </NCheckbox>
        </div>
        <div v-else />

        <div class="flex justify-end gap-x-2">
          <NButton quaternary @click="$emit('close')">
            {{ $t("common.close") }}
          </NButton>

          <NTooltip :disabled="validationErrors.length === 0" placement="top">
            <template #trigger>
              <NButton
                :disabled="
                  validationErrors.length > 0 ||
                  (validationWarnings.length > 0 && !bypassPolicyChecks)
                "
                type="primary"
                @click="handleConfirm"
              >
                {{ action === "RUN" ? $t("common.run") : action === "SKIP" ? $t("common.skip") : $t("common.cancel") }}
              </NButton>
            </template>
            <template #default>
              <ErrorList :errors="validationErrors" />
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
import { flatten } from "lodash-es";
import {
  NAlert,
  NButton,
  NCheckbox,
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
import { ErrorList } from "@/components/IssueV1/components/common";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import PlanCheckStatusCount from "@/components/Plan/components/PlanCheckStatusCount.vue";
import {
  usePlanCheckStatus,
  usePlanContextWithRollout,
} from "@/components/Plan/logic";
import { projectOfPlan } from "@/components/Plan/logic/utils";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { EnvironmentV1Name } from "@/components/v2";
import { trackPriorBackupOnTaskRun } from "@/composables/usePriorBackupTelemetry";
import { rolloutServiceClientConnect } from "@/connect";
import {
  pushNotification,
  useCurrentUserV1,
  useEnvironmentV1Store,
  usePolicyByParentAndType,
} from "@/store";
import {
  getProjectIdPlanUidStageUidTaskUidFromRolloutName,
  planNamePrefix,
  projectNamePrefix,
  stageNamePrefix,
  userNamePrefix,
} from "@/store/modules/v1/common";
import {
  Issue_ApprovalStatus,
  Issue_Approver_Status,
  Issue_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
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
  Task_Status,
  Task_Type,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { extractStageUID, isNullOrUndefined } from "@/utils";
import TaskDatabaseName from "./TaskDatabaseName.vue";
import { canRolloutTasks } from "./taskPermissions";

// Default delay for running tasks if not scheduled immediately.
// 1 hour in milliseconds
const DEFAULT_RUN_DELAY_MS = 60 * 60 * 1000;

// Stage is optional - only needed for regular rollouts, not database create/export
export type TargetType = { type: "tasks"; tasks?: Task[]; stage?: Stage };

const props = defineProps<{
  show: boolean;
  action: "RUN" | "SKIP" | "CANCEL";
  target: TargetType;
}>();

// Task status filters for each action type
const isRunnable = (task: Task) =>
  task.status === Task_Status.NOT_STARTED ||
  task.status === Task_Status.FAILED ||
  task.status === Task_Status.CANCELED;

const isCancellable = (task: Task) =>
  task.status === Task_Status.PENDING || task.status === Task_Status.RUNNING;

const emit = defineEmits<{
  (event: "close"): void;
  (event: "confirm"): void;
}>();

const { t } = useI18n();
const { issue, rollout, plan, taskRuns } = usePlanContextWithRollout();
const currentUser = useCurrentUserV1();

const loading = ref(false);
const environmentStore = useEnvironmentV1Store();
const comment = ref("");
const runTimeInMS = ref<number | undefined>(undefined);
const bypassPolicyChecks = ref(false);
const { statusSummary: planCheckStatus } = usePlanCheckStatus(plan);

// Cache permission check result to prevent flickering during poller refetches.
// Re-check only when the panel opens (show changes from false to true).
const canRolloutPermission = ref(true);
watch(
  () => props.show,
  (show) => {
    if (show) {
      // Check permission when panel opens - use provided tasks or stage tasks
      const tasks = props.target.tasks ?? props.target.stage?.tasks ?? [];
      canRolloutPermission.value = canRolloutTasks(tasks, issue.value);
    }
  },
  { immediate: true }
);

// Check issue approval status using the review context
const issueApprovalStatus = computed(() => {
  if (!issue?.value) {
    return { rolloutReady: true, hasIssue: false };
  }

  const currentIssue = issue.value;
  const approvalTemplate = currentIssue.approvalTemplate;

  // Check if issue has approval template
  if (!approvalTemplate || (approvalTemplate.flow?.roles || []).length === 0) {
    return { rolloutReady: true, hasIssue: true };
  }

  // Use the approval status to determine if rollout is ready
  const rolloutReady =
    currentIssue.approvalStatus === Issue_ApprovalStatus.APPROVED ||
    currentIssue.approvalStatus === Issue_ApprovalStatus.SKIPPED;

  // Determine the specific status (rejected vs pending)
  let status = "pending";
  if (
    currentIssue.approvers.some(
      (app) => app.status === Issue_Approver_Status.REJECTED
    )
  ) {
    status = "rejected";
  }

  return {
    rolloutReady,
    hasIssue: true,
    status,
  };
});

// Only show comment/reason field for SKIP action (stored in task.skipped_reason)
// RUN and CANCEL don't store reasons permanently
const shouldShowComment = computed(
  () => props.action === "SKIP" && !isNullOrUndefined(issue?.value)
);

// Plan check warning validation - always show as warning (never error)
const planCheckWarning = computed(() => {
  if (props.action !== "RUN" || planCheckStatus.value.total === 0) {
    return undefined;
  }

  // Always show plan check issues as warnings (regardless of require_* setting)
  if (planCheckStatus.value.running > 0) {
    return t(
      "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
    );
  }
  if (planCheckStatus.value.error > 0 || planCheckStatus.value.warning > 0) {
    return t(
      "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
    );
  }

  return undefined;
});

// Collect blocking validation errors
const validationErrors = computed(() => {
  const errors: string[] = [];

  // Permission errors - always block rollout
  // Use cached permission to prevent flickering during poller refetches
  if (!canRolloutPermission.value) {
    // Special message for data export issues when user is not the creator
    if (
      issue.value &&
      issue.value.type === Issue_Type.DATABASE_EXPORT &&
      issue.value.creator !== `${userNamePrefix}${currentUser.value.email}`
    ) {
      errors.push(t("task.data-export-creator-only"));
    } else {
      errors.push(t("task.no-permission"));
    }
  }

  // No active tasks to cancel - blocking error
  if (
    props.action === "CANCEL" &&
    eligibleTasks.value.length > 0 &&
    !eligibleTasks.value.some(
      (task) =>
        task.status === Task_Status.PENDING ||
        task.status === Task_Status.RUNNING
    )
  ) {
    errors.push(t("rollout.no-active-task-to-cancel"));
  }

  if (props.action === "RUN") {
    // No runnable tasks - blocking error
    if (
      eligibleTasks.value.length > 0 &&
      !eligibleTasks.value.some(
        (task) =>
          task.status === Task_Status.NOT_STARTED ||
          task.status === Task_Status.FAILED ||
          task.status === Task_Status.CANCELED
      )
    ) {
      errors.push(t("rollout.no-runnable-task"));
    }
  }

  return errors;
});

// Collect validation warnings that don't block rollout
const validationWarnings = computed(() => {
  const warnings: string[] = [];

  // Basic validation warnings
  if (eligibleTasks.value.length === 0) {
    warnings.push(t("common.no-data"));
  }

  if (props.action === "RUN") {
    // Validate scheduled time if not running immediately
    if (runTimeInMS.value !== undefined && runTimeInMS.value <= Date.now()) {
      warnings.push(t("task.error.scheduled-time-must-be-in-the-future"));
    }

    // Plan check warnings - always show when checks have issues
    if (planCheckWarning.value) {
      warnings.push(planCheckWarning.value);
    }

    // Issue approval warnings - always show when not approved (regardless of require_* setting)
    if (
      issueApprovalStatus.value.hasIssue &&
      !issueApprovalStatus.value.rolloutReady
    ) {
      const isRejected = issueApprovalStatus.value.status === "rejected";
      warnings.push(
        isRejected
          ? t("issue.approval.rejected-error")
          : t("issue.approval.pending-error")
      );
    }

    // Automatic rollout info (always show as warning for non-export tasks with no task runs)
    if (
      isAutomaticRollout.value &&
      rolloutType.value !== "DATABASE_EXPORT" &&
      !hasTaskRuns.value
    ) {
      warnings.push(t("rollout.automatic-rollout.description"));
    }
  }

  return warnings;
});

const shouldShowBypassOption = computed(() => {
  if (props.action !== "RUN") {
    return false;
  }

  // Only show bypass option when there are warnings but NO errors
  return (
    validationWarnings.value.length > 0 && validationErrors.value.length === 0
  );
});

// All tasks from rollout
const allRolloutTasks = computed(() =>
  flatten(rollout.value.stages.map((stage) => stage.tasks))
);

// Detect rollout type based on tasks
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

// Get eligible tasks based on action type
const eligibleTasks = computed(() => {
  // For database create/export, use all rollout tasks; otherwise use stage tasks
  const tasks = isDatabaseCreateOrExport.value
    ? allRolloutTasks.value
    : (props.target.tasks ?? props.target.stage?.tasks ?? []);

  // Filter by action type (RUN and SKIP use same filter)
  if (props.action === "RUN" || props.action === "SKIP") {
    return tasks.filter(isRunnable);
  }
  if (props.action === "CANCEL") {
    return tasks.filter(isCancellable);
  }
  return tasks;
});

const title = computed(() => {
  const n = eligibleTasks.value.length;
  if (props.action === "RUN") return t("task.run-task", { n });
  if (props.action === "SKIP") return t("task.skip-task", { n });
  if (props.action === "CANCEL") return t("task.cancel-task", { n });
  return "";
});

// Get rollout policy for target stage environment
const { policy: rolloutPolicy } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.target.stage?.environment || "",
    policyType: PolicyType.ROLLOUT_POLICY,
  }))
);

const isAutomaticRollout = computed(
  () =>
    rolloutPolicy.value?.enforce &&
    rolloutPolicy.value.policy?.case === "rolloutPolicy" &&
    rolloutPolicy.value.policy.value.automatic
);

const hasTaskRuns = computed(() =>
  eligibleTasks.value.some((task) =>
    taskRuns.value.some((taskRun) => taskRun.name.startsWith(`${task.name}/`))
  )
);

// Virtual scroll for large task lists
const useVirtualScroll = computed(() => eligibleTasks.value.length > 50);
const itemHeight = 32;

const showScheduledTimePicker = computed(() => props.action === "RUN");

// UI visibility flags
const shouldShowStageInfo = computed(() => !isDatabaseCreateOrExport.value);
const shouldShowTaskInfo = computed(
  () => rolloutType.value !== "DATABASE_CREATE"
);

// Handle execution mode change (immediate vs scheduled)
const handleExecutionModeChange = (value: string) => {
  if (value === "immediate") {
    runTimeInMS.value = undefined;
  } else {
    runTimeInMS.value = Date.now() + DEFAULT_RUN_DELAY_MS;
  }
};

// Helper function to group tasks by their stage (environment) for export tasks
const groupTasksByStage = (tasks: Task[]) => {
  const tasksByStage = new Map<string, Task[]>();
  for (const task of tasks) {
    const stageId = extractStageUID(task.name);
    if (!tasksByStage.has(stageId)) {
      tasksByStage.set(stageId, []);
    }
    tasksByStage.get(stageId)!.push(task);
  }
  return tasksByStage;
};

const addRunTimeToRequest = (request: BatchRunTasksRequest) => {
  if (runTimeInMS.value !== undefined) {
    // Convert timestamp to protobuf Timestamp format
    const runTimeSeconds = Math.floor(runTimeInMS.value / 1000);
    const runTimeNanos = (runTimeInMS.value % 1000) * 1000000;
    request.runTime = create(TimestampSchema, {
      seconds: BigInt(runTimeSeconds),
      nanos: runTimeNanos,
    });
  }
};

// Execute batch operation for tasks grouped by stage
const executeBatchByStage = async (
  operation: (parent: string, tasks: Task[]) => Promise<void>
) => {
  if (!rollout.value) return;

  if (isDatabaseCreateOrExport.value) {
    // For database create/export, group tasks by stage
    const tasksByStage = groupTasksByStage(eligibleTasks.value);
    for (const [stageId, stageTasks] of tasksByStage) {
      await operation(`${rollout.value.name}/stages/${stageId}`, stageTasks);
    }
  } else if (props.target.stage) {
    // For regular rollouts, use the single stage
    await operation(props.target.stage.name, eligibleTasks.value);
  }
};

const handleConfirm = async () => {
  if (loading.value) return;

  loading.value = true;
  try {
    if (props.action === "RUN") {
      await executeBatchByStage(async (parent, tasks) => {
        const request = create(BatchRunTasksRequestSchema, {
          parent,
          tasks: tasks.map((task) => task.name),
        });
        addRunTimeToRequest(request);
        await rolloutServiceClientConnect.batchRunTasks(request);
      });

      // Track prior backup telemetry (async, non-blocking)
      trackPriorBackupOnTaskRun(
        eligibleTasks.value,
        plan.value,
        projectOfPlan(plan.value),
        props.target.stage?.environment ?? ""
      );
    } else if (props.action === "SKIP") {
      await executeBatchByStage(async (parent, tasks) => {
        const request = create(BatchSkipTasksRequestSchema, {
          parent,
          tasks: tasks.map((task) => task.name),
          reason: comment.value,
        });
        await rolloutServiceClientConnect.batchSkipTasks(request);
      });
    } else if (props.action === "CANCEL") {
      await cancelTasks();
    }

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

const resetState = () => {
  comment.value = "";
  runTimeInMS.value = undefined;
  bypassPolicyChecks.value = false;
};

// initial the selected task run time.
watchEffect(() => {
  const runTimes = new Set(eligibleTasks.value.map((task) => task.runTime));
  if (runTimes.size != 1) {
    return;
  }
  const runTime = [...runTimes][0];
  if (!runTime) {
    return;
  }
  runTimeInMS.value = Number(runTime.seconds) * 1000 + runTime.nanos / 1000000;
});

const cancelTasks = async () => {
  // Group tasks by stage first
  const tasksByStage = new Map<string, Task[]>();
  for (const task of eligibleTasks.value) {
    // Extract stage name from task path: projects/{projectId}/plans/{planId}/rollout/stages/{stageId}/tasks/...
    const [projectId, planId, stageId] =
      getProjectIdPlanUidStageUidTaskUidFromRolloutName(task.name);
    if (projectId && planId && stageId) {
      const stageName = `${projectNamePrefix}${projectId}/${planNamePrefix}${planId}/${"rollout"}/${stageNamePrefix}${stageId}`;
      if (!tasksByStage.has(stageName)) {
        tasksByStage.set(stageName, []);
      }
      tasksByStage.get(stageName)!.push(task);
    }
  }

  // Fetch task runs at stage level and filter by eligible tasks
  const cancelableTaskRunsByStage = new Map<string, string[]>();

  for (const [stageName, tasks] of tasksByStage) {
    const taskNames = new Set(tasks.map((t) => t.name));
    const request = create(ListTaskRunsRequestSchema, {
      parent: `${stageName}/tasks/-`,
    });

    const response = await rolloutServiceClientConnect.listTaskRuns(request);
    const stageTaskRuns = (response.taskRuns || [])
      .filter((taskRun) => {
        // Only include task runs for our eligible tasks
        const taskName = taskRun.name.split("/taskRuns/")[0];
        return (
          taskNames.has(taskName) &&
          (taskRun.status === TaskRun_Status.PENDING ||
            taskRun.status === TaskRun_Status.RUNNING)
        );
      })
      .map((taskRun) => taskRun.name);

    if (stageTaskRuns.length > 0) {
      cancelableTaskRunsByStage.set(stageName, stageTaskRuns);
    }
  }

  // Cancel task runs for each stage
  await Promise.all(
    Array.from(cancelableTaskRunsByStage.entries()).map(
      ([stageName, taskRunNames]) => {
        const request = create(BatchCancelTaskRunsRequestSchema, {
          parent: `${stageName}/tasks/-`,
          taskRuns: taskRunNames,
        });
        return rolloutServiceClientConnect.batchCancelTaskRuns(request);
      }
    )
  );
};
</script>
