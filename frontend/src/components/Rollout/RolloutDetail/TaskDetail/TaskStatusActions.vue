<template>
  <router-link
    v-if="issue && !issueReviewDone"
    :to="`/${issue.name}`"
    class="shrink-0"
    target="_blank"
  >
    <NButton quaternary type="primary" size="large">
      {{ $t("issue.waiting-for-review") }}
      <ExternalLinkIcon class="ml-1" :size="16" />
    </NButton>
  </router-link>
  <div
    v-else-if="primaryAction || dropdownOptions.length > 0"
    class="flex flex-row justify-end items-center gap-x-2"
  >
    <NButton
      v-if="primaryAction"
      type="primary"
      @click="handleTaskStatusAction(primaryAction)"
    >
      {{ actionDisplayTitle(primaryAction) }}
    </NButton>
    <NDropdown
      v-if="dropdownOptions.length > 0"
      trigger="hover"
      :options="dropdownOptions"
      @select="(action) => handleTaskStatusAction(action)"
    >
      <NButton>
        <template #icon>
          <EllipsisVerticalIcon class="w-5 h-5" />
        </template>
      </NButton>
    </NDropdown>
  </div>
</template>

<script setup lang="ts">
import { EllipsisVerticalIcon, ExternalLinkIcon } from "lucide-vue-next";
import { NButton, NDropdown } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { extractReviewContext } from "@/components/IssueV1";
import { rolloutServiceClient } from "@/grpcweb";
import {
  Issue_Approver_Status,
  IssueStatus,
} from "@/types/proto/v1/issue_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { useRolloutDetailContext } from "../context";
import { useTaskDetailContext } from "./context";
import { stageForTask } from "./utils";

type TaskStatusAction =
  // NOT_STARTED -> PENDING
  | "RUN"
  // FAILED -> PENDING
  | "RETRY"
  // * -> CANCELLED
  | "CANCEL"
  // * -> SKIPPED
  | "SKIP";

const { t } = useI18n();
const { rollout, issue, emmiter } = useRolloutDetailContext();
const { task, taskRuns } = useTaskDetailContext();

const issueReviewDone = computed(() => {
  if (issue.value && issue.value.status === IssueStatus.OPEN) {
    const issueReviewContext = extractReviewContext(issue.value);
    if (issueReviewContext.status.value !== Issue_Approver_Status.APPROVED) {
      return false;
    }
  }
  return true;
});

const primaryAction = computed((): TaskStatusAction | null => {
  if (task.value.status === Task_Status.NOT_STARTED) {
    return "RUN";
  } else if (task.value.status === Task_Status.FAILED) {
    return "RETRY";
  } else {
    return null;
  }
});

const dropdownActions = computed((): TaskStatusAction[] => {
  if (
    [
      Task_Status.NOT_STARTED,
      Task_Status.FAILED,
      Task_Status.CANCELED,
    ].includes(task.value.status)
  ) {
    return ["SKIP"];
  } else if (
    [Task_Status.PENDING, Task_Status.RUNNING].includes(task.value.status)
  ) {
    return ["CANCEL"];
  } else {
    return [];
  }
});

const dropdownOptions = computed(() => {
  return dropdownActions.value.map((action) => {
    return {
      key: action,
      label: actionDisplayTitle(action),
    };
  });
});

const actionDisplayTitle = (action: TaskStatusAction) => {
  if (action === "RUN") {
    return t("common.run");
  } else if (action === "RETRY") {
    return t("common.retry");
  } else if (action === "CANCEL") {
    return t("common.cancel");
  } else if (action === "SKIP") {
    return t("common.skip");
  }
};

const handleTaskStatusAction = async (action: TaskStatusAction) => {
  const stage = stageForTask(rollout.value, task.value);
  if (!stage) return;
  if (action === "RUN" || action === "RETRY") {
    await rolloutServiceClient.batchRunTasks({
      parent: stage.name,
      tasks: [task.value.name],
    });
  } else if (action === "SKIP") {
    await rolloutServiceClient.batchSkipTasks({
      parent: stage.name,
      tasks: [task.value.name],
    });
  } else if (action === "CANCEL") {
    await rolloutServiceClient.batchCancelTaskRuns({
      parent: `${stage.name}/tasks/-`,
      taskRuns: taskRuns.value.map((taskRun) => taskRun.name),
      // TODO(steven): Let user input reason.
      reason: "",
    });
  }
  emmiter.emit("task-status-action");
};
</script>
