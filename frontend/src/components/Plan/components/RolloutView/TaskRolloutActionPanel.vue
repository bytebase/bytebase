<template>
  <CommonDrawer
    :show="show"
    :title="title"
    :loading="state.loading"
    @show="resetState"
    @close="$emit('close')"
  >
    <template #default>
      <div class="flex flex-col gap-y-4 h-full overflow-y-hidden px-1">
        <!-- Consolidated Alert -->
        <NAlert
          v-if="validationMessages.length > 0"
          type="warning"
          :title="$t('rollout.task-execution-notices')"
        >
          <ul class="list-disc list-inside space-y-1">
            <li
              v-for="(message, index) in validationMessages"
              :key="index"
              class="text-sm"
            >
              {{ message }}
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

        <!-- Stage/Task information - hidden for database creation tasks -->
        <template v-if="shouldShowStageTaskInfo">
          <div
            class="flex flex-row gap-x-2 shrink-0 overflow-y-hidden justify-start items-center"
          >
            <label class="font-medium text-control">
              {{ $t("common.stage") }}
            </label>
            <EnvironmentV1Name
              :environment="
                environmentStore.getEnvironmentByName(targetStage.environment)
              "
              :link="false"
            />
          </div>

          <div
            class="flex flex-col gap-y-1 shrink overflow-y-hidden justify-start"
          >
            <div class="flex items-center justify-between">
              <label class="text-control">
                <span class="font-medium">{{
                  $t("common.task", eligibleTasks.length)
                }}</span>
                <span class="opacity-80" v-if="eligibleTasks.length > 1">
                  ({{
                    eligibleTasks.length === target.stage.tasks.length
                      ? eligibleTasks.length
                      : `${eligibleTasks.length} / ${target.stage.tasks.length}`
                  }})
                </span>
              </label>
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
        </template>

        <div v-if="showScheduledTimePicker" class="flex flex-col">
          <h3 class="font-medium text-control mb-1">
            {{ $t("task.execution-time") }}
          </h3>
          <NRadioGroup
            :size="'large'"
            :value="runTimeInMS === undefined ? 'immediate' : 'scheduled'"
            @update:value="handleExecutionModeChange"
            class="!flex flex-row gap-x-4"
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

        <!-- Only show reason/comment input if issue is available -->
        <div
          v-if="shouldShowComment"
          class="flex flex-col gap-y-1 shrink-0 border-t pt-2"
        >
          <p class="text-control">
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
        <!-- Force rollout checkbox -->
        <div v-if="shouldShowForceRollout" class="flex items-center">
          <NCheckbox v-model:checked="forceRollout" :disabled="state.loading">
            {{ $t("rollout.force-rollout") }}
          </NCheckbox>
        </div>
        <div v-else></div>

        <div class="flex justify-end gap-x-2">
          <NButton quaternary @click="$emit('close')">
            {{ $t("common.close") }}
          </NButton>

          <NTooltip :disabled="validationMessages.length === 0" placement="top">
            <template #trigger>
              <NButton
                :disabled="validationMessages.length > 0 && !forceRollout"
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
              <ErrorList :errors="validationMessages" />
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
import { head } from "lodash-es";
import {
  NAlert,
  NButton,
  NDatePicker,
  NInput,
  NScrollbar,
  NTag,
  NTooltip,
  NVirtualList,
  NCheckbox,
  NRadio,
  NRadioGroup,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { semanticTaskType } from "@/components/IssueV1";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import { ErrorList } from "@/components/IssueV1/components/common";
import { EnvironmentV1Name } from "@/components/v2";
import { rolloutServiceClientConnect } from "@/grpcweb";
import {
  pushNotification,
  useCurrentProjectV1,
  useEnvironmentV1Store,
  usePolicyByParentAndType,
} from "@/store";
import { Issue_Approver_Status } from "@/types/proto-es/v1/issue_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  BatchRunTasksRequestSchema,
  BatchSkipTasksRequestSchema,
  BatchCancelTaskRunsRequestSchema,
  ListTaskRunsRequestSchema,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import {
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  isNullOrUndefined,
} from "@/utils";
import { usePlanContextWithRollout, usePlanCheckStatus } from "../../logic";
import { useIssueReviewContext } from "../../logic/issue-review";
import PlanCheckStatusCount from "../PlanCheckStatusCount.vue";
import TaskDatabaseName from "./TaskDatabaseName.vue";
import { useTaskActionPermissions } from "./taskPermissions";

// Default delay for running tasks if not scheduled immediately.
// 1 hour in milliseconds
const DEFAULT_RUN_DELAY_MS = 60 * 60 * 1000;

type LocalState = {
  loading: boolean;
};

export type TargetType = { type: "tasks"; tasks?: Task[]; stage: Stage };

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
const { project } = useCurrentProjectV1();
const { issue, rollout, plan } = usePlanContextWithRollout();
const reviewContext = useIssueReviewContext();
const state = reactive<LocalState>({
  loading: false,
});
const environmentStore = useEnvironmentV1Store();
const { canPerformTaskAction } = useTaskActionPermissions();
const comment = ref("");
const runTimeInMS = ref<number | undefined>(undefined);
const forceRollout = ref(false);
const { statusSummary: planCheckStatus } = usePlanCheckStatus(plan);

// Check issue approval status using the review context
const issueApprovalStatus = computed(() => {
  if (!issue?.value) {
    return { isApproved: true, hasIssue: false };
  }

  const currentIssue = issue.value;
  const approvalTemplate = head(currentIssue.approvalTemplates);

  // Check if issue has approval template
  if (!approvalTemplate || (approvalTemplate.flow?.steps || []).length === 0) {
    return { isApproved: true, hasIssue: true };
  }

  // Use the review context to determine approval status
  const isApproved = reviewContext.done.value;

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
    isApproved,
    hasIssue: true,
    status,
  };
});

// Check if all previous stages are complete (done or skipped)
const previousStagesStatus = computed(() => {
  if (!rollout.value || !targetStage.value) {
    return { allComplete: true, hasIncomplete: false };
  }

  const stages = rollout.value.stages;
  const currentStageIndex = stages.findIndex(
    (stage) => stage.environment === targetStage.value.environment
  );

  if (currentStageIndex <= 0) {
    // This is the first stage or stage not found
    return { allComplete: true, hasIncomplete: false };
  }

  // Check all previous stages
  for (let i = 0; i < currentStageIndex; i++) {
    const stage = stages[i];
    const hasIncompleteTasks = stage.tasks.some(
      (task) =>
        task.status !== Task_Status.DONE && task.status !== Task_Status.SKIPPED
    );
    if (hasIncompleteTasks) {
      return { allComplete: false, hasIncomplete: true };
    }
  }

  return { allComplete: true, hasIncomplete: false };
});

const shouldShowComment = computed(() => !isNullOrUndefined(issue?.value));

// Plan check error validation
const planCheckError = computed(() => {
  if (props.action === "RUN") {
    // Get plan check runs from the plan check status
    if (planCheckStatus.value.error > 0 || planCheckStatus.value.warning > 0) {
      return t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
      );
    }
    if (planCheckStatus.value.running > 0) {
      return t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
      );
    }
  }
  return undefined;
});

// Collect all validation messages for both display and error checking
const validationMessages = computed(() => {
  const messages: string[] = [];

  // Basic validation errors
  if (runnableTasks.value.length === 0) {
    messages.push(t("common.no-data"));
  }

  if (
    !canPerformTaskAction(
      runnableTasks.value,
      rollout.value,
      project.value,
      issue?.value
    )
  ) {
    messages.push(t("task.no-permission"));
  }

  if (props.action === "RUN") {
    // Validate scheduled time if not running immediately
    if (runTimeInMS.value !== undefined && runTimeInMS.value <= Date.now()) {
      messages.push(t("task.error.scheduled-time-must-be-in-the-future"));
    }

    // Issue approval messages (only if not forcing rollout)
    if (
      issueApprovalStatus.value.hasIssue &&
      !issueApprovalStatus.value.isApproved
    ) {
      const isRejected = issueApprovalStatus.value.status === "rejected";
      messages.push(
        isRejected
          ? t("issue.approval.rejected-error")
          : t("issue.approval.pending-error")
      );
    }

    // Previous stages incomplete (only if not forcing rollout)
    if (previousStagesStatus.value.hasIncomplete) {
      messages.push(
        t("rollout.message.pervious-stages-incomplete.description")
      );
    }

    // Plan check errors (only if not forcing rollout)
    if (planCheckError.value) {
      messages.push(planCheckError.value);
    }

    // Automatic rollout info (always show as informational)
    if (isAutomaticRollout.value) {
      messages.push(t("rollout.automatic-rollout.description"));
    }
  }

  if (
    props.action === "CANCEL" &&
    runnableTasks.value.some(
      (task) =>
        task.status !== Task_Status.PENDING &&
        task.status !== Task_Status.RUNNING
    )
  ) {
    messages.push(t("rollout.no-active-task-to-cancel"));
  }

  return messages;
});

const shouldShowForceRollout = computed(() => {
  // Show force rollout checkbox for RUN action when:
  // 1. Issue approval is not complete, OR
  // 2. Previous stages are not complete, OR
  // 3. Environment has automatic rollout enabled
  return (
    props.action === "RUN" &&
    (hasWorkspacePermissionV2("bb.taskRuns.create") ||
      hasProjectPermissionV2(project.value, "bb.taskRuns.create")) &&
    ((issueApprovalStatus.value.hasIssue &&
      !issueApprovalStatus.value.isApproved) ||
      previousStagesStatus.value.hasIncomplete ||
      planCheckError.value ||
      isAutomaticRollout.value)
  );
});

// Extract stage from target
const targetStage = computed(() => {
  return props.target.stage;
});

// Get rollout policy for target stage environment
const { policy: rolloutPolicy } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: targetStage.value?.environment || "",
    policyType: PolicyType.ROLLOUT_POLICY,
  }))
);

// Check if target stage has automatic rollout policy
const isAutomaticRollout = computed(() => {
  return (
    rolloutPolicy.value?.enforce &&
    rolloutPolicy.value.policy?.case === "rolloutPolicy" &&
    rolloutPolicy.value.policy.value.automatic
  );
});

// Extract tasks if provided directly
const targetTasks = computed(() => {
  if (props.target.type === "tasks") {
    return props.target.tasks;
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

// Get the tasks that will actually be run
const runnableTasks = computed(() => {
  return eligibleTasks.value;
});

// Virtual scroll configuration
const useVirtualScroll = computed(() => eligibleTasks.value.length > 50);
const itemHeight = computed(() => 32); // Height of each task item in pixels

const showScheduledTimePicker = computed(() => {
  return props.action === "RUN";
});

// Check if any of the eligible tasks are database creation tasks
const isDatabaseCreationTask = computed(() => {
  return eligibleTasks.value.every(
    (task) => task.type === Task_Type.DATABASE_CREATE
  );
});

const shouldShowStageTaskInfo = computed(() => !isDatabaseCreationTask.value);

// Handle execution mode change (immediate vs scheduled)
const handleExecutionModeChange = (value: string) => {
  if (value === "immediate") {
    runTimeInMS.value = undefined;
  } else {
    runTimeInMS.value = Date.now() + DEFAULT_RUN_DELAY_MS;
  }
};

const handleConfirm = async () => {
  state.loading = true;
  try {
    if (props.action === "RUN") {
      // Prepare the request parameters
      const request = create(BatchRunTasksRequestSchema, {
        parent: targetStage.value.name,
        tasks: runnableTasks.value.map((task) => task.name),
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
    } else if (props.action === "SKIP") {
      const request = create(BatchSkipTasksRequestSchema, {
        parent: targetStage.value.name,
        tasks: runnableTasks.value.map((task) => task.name),
        reason: comment.value,
      });
      await rolloutServiceClientConnect.batchSkipTasks(request);
    } else if (props.action === "CANCEL") {
      // Fetch task runs for the tasks to be canceled.
      const taskRuns = (
        await Promise.all(
          runnableTasks.value.map(async (task) => {
            const request = create(ListTaskRunsRequestSchema, {
              parent: task.name,
              pageSize: 10,
            });
            return rolloutServiceClientConnect
              .listTaskRuns(request)
              .then((response) => response.taskRuns || []);
          })
        )
      ).flat();
      const cancelableTaskRuns = taskRuns.filter(
        (taskRun) =>
          taskRun.status === TaskRun_Status.PENDING ||
          taskRun.status === TaskRun_Status.RUNNING
      );
      const request = create(BatchCancelTaskRunsRequestSchema, {
        parent: `${targetStage.value.name}/tasks/-`,
        taskRuns: cancelableTaskRuns.map((taskRun) => taskRun.name),
        reason: comment.value,
      });
      await rolloutServiceClientConnect.batchCancelTaskRuns(request);
    }

    emit("confirm");
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

const resetState = () => {
  comment.value = "";
  runTimeInMS.value = undefined;
  forceRollout.value = false;
};
</script>
