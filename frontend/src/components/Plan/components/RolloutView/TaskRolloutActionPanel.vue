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
        <!-- Issue Approval Alert -->
        <NAlert
          v-if="shouldShowForceRollout"
          type="warning"
          :title="
            issueApprovalStatus.status === 'rejected'
              ? $t('issue.approval.rejected-title')
              : $t('issue.approval.pending-title')
          "
        >
          {{
            issueApprovalStatus.status === "rejected"
              ? $t("issue.approval.rejected-description")
              : $t("issue.approval.pending-description")
          }}
        </NAlert>

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
                ({{ eligibleTasks.length }})
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
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { semanticTaskType } from "@/components/IssueV1";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import { ErrorList } from "@/components/IssueV1/components/common";
import { rolloutServiceClientConnect } from "@/grpcweb";
import {
  pushNotification,
  useCurrentProjectV1,
  useEnvironmentV1Store,
} from "@/store";
import { Issue_Approver_Status } from "@/types/proto-es/v1/issue_service_pb";
import {
  BatchRunTasksRequestSchema,
  BatchSkipTasksRequestSchema,
  BatchCancelTaskRunsRequestSchema,
  ListTaskRunsRequestSchema,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { usePlanContextWithRollout } from "../../logic";
import { useIssueReviewContext } from "../../logic/issue-review";
import TaskDatabaseName from "./TaskDatabaseName.vue";
import { useTaskActionPermissions } from "./taskPermissions";

// Default delay for running tasks if not scheduled immediately.
const DEFAULT_RUN_DELAY_MS = 60000;

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
const { issue, rollout } = usePlanContextWithRollout();
const reviewContext = useIssueReviewContext();
const state = reactive<LocalState>({
  loading: false,
});
const environmentStore = useEnvironmentV1Store();
const { canPerformTaskAction } = useTaskActionPermissions();
const comment = ref("");
const runTimeInMS = ref<number | undefined>(undefined);
const forceRollout = ref(false);

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

const shouldShowForceRollout = computed(() => {
  // Show force rollout checkbox only for RUN action with issue approval
  return (
    props.action === "RUN" &&
    issueApprovalStatus.value.hasIssue &&
    !issueApprovalStatus.value.isApproved &&
    hasWorkspacePermissionV2("bb.taskRuns.create")
  );
});

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

const confirmErrors = computed(() => {
  const errors: string[] = [];

  if (runnableTasks.value.length === 0) {
    errors.push(t("common.no-data"));
  }

  if (
    !canPerformTaskAction(
      runnableTasks.value,
      rollout.value,
      project.value,
      issue?.value
    ) &&
    !hasWorkspacePermissionV2("bb.taskRuns.create")
  ) {
    errors.push(t("task.no-permission"));
  }

  // Validate scheduled time if not running immediately (only for RUN)
  if (props.action === "RUN" && runTimeInMS.value !== undefined) {
    if (runTimeInMS.value <= Date.now()) {
      errors.push(t("task.error.scheduled-time-must-be-in-the-future"));
    }
  }

  // Check issue approval for RUN action
  if (
    props.action === "RUN" &&
    !issueApprovalStatus.value.isApproved &&
    issueApprovalStatus.value.hasIssue &&
    !forceRollout.value
  ) {
    if (issueApprovalStatus.value.status === "rejected") {
      errors.push(t("issue.approval.rejected-error"));
    } else {
      errors.push(t("issue.approval.pending-error"));
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
    // Check task status.
    errors.push("No active task to cancel");
  }

  return errors;
});

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
