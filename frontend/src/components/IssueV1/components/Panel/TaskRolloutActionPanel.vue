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
        <div
          v-if="stage"
          class="flex flex-col gap-y-1 shrink-0 overflow-y-hidden justify-start"
        >
          <label class="font-medium text-control">
            {{ $t("common.stage") }}
          </label>
          <div class="textinfolabel break-all">
            {{ stage.title }}
          </div>
        </div>
        <div
          class="flex flex-col gap-y-1 shrink overflow-y-hidden justify-start"
        >
          <label class="font-medium text-control">
            <template v-if="taskList.length === 1">
              {{ $t("common.task") }}
            </template>
            <template v-else>{{ $t("common.tasks") }}</template>
          </label>
          <div class="flex-1 overflow-y-auto">
            <NScrollbar>
              <ul class="text-sm space-y-2">
                <li
                  v-for="task in taskList"
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
            (action === 'ROLLOUT' || action === 'RETRY') &&
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
          <div>
            <NCheckbox v-model:checked="performActionAnyway">
              {{
                $t("issue.action-anyway", {
                  action: taskRolloutActionDialogButtonName(action, taskList),
                })
              }}
            </NCheckbox>
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
      <div v-if="action" class="flex justify-end gap-x-3">
        <NButton @click="$emit('close')">
          {{ $t("common.cancel") }}
        </NButton>

        <NTooltip :disabled="confirmErrors.length === 0" placement="top">
          <template #trigger>
            <NButton
              :disabled="confirmErrors.length > 0"
              v-bind="taskRolloutActionButtonProps(action)"
              @click="handleConfirm(action!, comment)"
            >
              {{ taskRolloutActionDialogButtonName(action, taskList) }}
            </NButton>
          </template>
          <template #default>
            <ErrorList :errors="confirmErrors" />
          </template>
        </NTooltip>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { head, uniqBy } from "lodash-es";
import {
  NButton,
  NCheckbox,
  NInput,
  NScrollbar,
  NTag,
  NTooltip,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { TaskRolloutAction } from "@/components/IssueV1/logic";
import {
  databaseForTask,
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
import { rolloutServiceClient } from "@/grpcweb";
import { pushNotification } from "@/store";
import type { Task, TaskRun } from "@/types/proto/v1/rollout_service";
import { TaskRun_Status } from "@/types/proto/v1/rollout_service";
import { ErrorList } from "../common";
import CommonDrawer from "./CommonDrawer.vue";
import RolloutTaskDatabaseName from "./RolloutTaskDatabaseName.vue";

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
const { issue, selectedTask, events, getPlanCheckRunsForTask } =
  useIssueContext();
const comment = ref("");
const performActionAnyway = ref(false);

const title = computed(() => {
  if (!props.action) return "";

  const action = taskRolloutActionDisplayName(props.action, selectedTask.value);
  if (props.taskList.length > 1) {
    return t("task.action-all-tasks-in-current-stage", { action });
  }
  return action;
});

const database = computed(() =>
  databaseForTask(issue.value, selectedTask.value)
);

const stage = computed(() => {
  const firstTask = head(props.taskList);
  if (!firstTask) return undefined;
  return stageForTask(issue.value, firstTask);
});

const planCheckRunList = computed(() => {
  const list = props.taskList.flatMap(getPlanCheckRunsForTask);
  return uniqBy(list, (checkRun) => checkRun.name);
});

const planCheckErrors = computed(() => {
  const errors: string[] = [];
  if (props.action === "ROLLOUT" || props.action === "RETRY") {
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
  if (planCheckErrors.value.length > 0 && !performActionAnyway.value) {
    errors.push(...planCheckErrors.value);
  }
  return errors;
});

const handleConfirm = async (
  action: TaskRolloutAction,
  comment: string | undefined
) => {
  state.loading = true;
  try {
    const stage = stageForTask(issue.value, props.taskList[0]);
    if (!stage) return;
    if (action === "ROLLOUT" || action === "RETRY" || action === "RESTART") {
      await rolloutServiceClient.batchRunTasks({
        parent: stage.name,
        tasks: props.taskList.map((task) => task.name),
        reason: comment,
      });
    } else if (action === "SKIP") {
      await rolloutServiceClient.batchSkipTasks({
        parent: stage.name,
        tasks: props.taskList.map((task) => task.name),
        reason: comment,
      });
    } else if (action === "CANCEL") {
      const taskRunListToCancel = props.taskList
        .map((task) => {
          const taskRunList = taskRunListForTask(issue.value, task);
          const currentRunningTaskRun = taskRunList.find(
            (taskRun) =>
              taskRun.status === TaskRun_Status.RUNNING ||
              taskRun.status === TaskRun_Status.PENDING
          );
          return currentRunningTaskRun;
        })
        .filter((taskRun) => taskRun !== undefined) as TaskRun[];
      if (taskRunListToCancel.length > 0) {
        await rolloutServiceClient.batchCancelTaskRuns({
          parent: `${stage.name}/tasks/-`,
          taskRuns: taskRunListToCancel.map((taskRun) => taskRun.name),
          reason: comment,
        });
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
  performActionAnyway.value = false;
};
</script>
