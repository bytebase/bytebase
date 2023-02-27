<template>
  <div class="text-sm font-medium">
    <template v-if="!payload?.rollbackEnabled">
      <LogButton />
    </template>
    <template v-else>
      <template v-if="payload?.rollbackSqlStatus === 'FAILED'">
        <div class="tooltip-wrapper">
          <NTooltip>
            <template #trigger>
              <span class="text-error">{{ $t("task.rollback.failed") }}</span>
            </template>

            <div v-if="payload.rollbackError" class="max-w-[20rem]">
              {{ payload.rollbackError }}
            </div>
          </NTooltip>
        </div>
      </template>
      <template v-else-if="payload?.rollbackSqlStatus === 'DONE'">
        <template
          v-if="
            taskRollbackBy && taskRollbackBy.rollbackByIssueId !== UNKNOWN_ID
          "
        >
          <router-link
            :to="`/issue/${taskRollbackBy.rollbackByIssueId}`"
            class="text-accent inline-flex gap-x-1"
          >
            <span>{{ $t("common.issue") }}</span>
            <span>#{{ taskRollbackBy.rollbackByIssueId }}</span>
          </router-link>
        </template>
        <BBTooltipButton
          v-else
          :disabled="!allowPreviewRollback"
          tooltip-mode="DISABLED-ONLY"
          class="btn-normal !px-3 !py-1"
          @click="tryRollbackTask"
        >
          {{ $t("task.rollback.preview-rollback-issue") }}
          <template v-if="!payload?.rollbackStatement" #tooltip>
            <div class="whitespace-pre-line">
              {{ $t("task.rollback.empty-rollback-statement") }}
            </div>
          </template>
        </BBTooltipButton>
      </template>
      <template v-else>
        <LoggingButton />
      </template>
    </template>
  </div>
</template>
<script lang="ts" setup>
import { computed, reactive, type Ref } from "vue";
import { useRouter } from "vue-router";
import { NTooltip } from "naive-ui";

import {
  ActivityIssueCommentCreatePayload,
  Issue,
  Task,
  TaskDatabaseDataUpdatePayload,
  TaskRollbackBy,
  UNKNOWN_ID,
} from "@/types";
import { BBTooltipButton } from "@/bbkit";
import { useIssueLogic } from "../logic";
import { useRollbackLogic } from "./common";
import LogButton from "./LogButton.vue";
import LoggingButton from "./LoggingButton.vue";
import { useActivityStore } from "@/store";

type LocalState = {
  loading: boolean;
  rollbackIssue: Issue | undefined;
};

const router = useRouter();

const context = useIssueLogic();
const { allowRollback } = useRollbackLogic();

const issue = context.issue as Ref<Issue>;
const task = context.selectedTask as Ref<Task>;
const payload = computed(
  () => task.value.payload as TaskDatabaseDataUpdatePayload | undefined
);

const state = reactive<LocalState>({
  loading: false,
  rollbackIssue: undefined,
});

const allowPreviewRollback = computed(() => {
  if (!allowRollback.value) {
    return false;
  }
  if (!payload.value?.rollbackStatement) {
    return false;
  }
  return true;
});

const taskRollbackBy = computed((): TaskRollbackBy | undefined => {
  const activityList = useActivityStore().getActivityListByIssue(
    issue.value.id
  );
  // Find the latest comment activity with TaskRollbackBy struct if possible.
  for (let i = activityList.length - 1; i >= 0; i--) {
    const activity = activityList[i];
    if (activity.type !== "bb.issue.comment.create") continue;
    const payload = activity.payload as
      | ActivityIssueCommentCreatePayload
      | undefined;
    if (!payload) continue;
    const taskRollbackBy = payload.taskRollbackBy;
    if (!taskRollbackBy) continue;
    return taskRollbackBy;
  }
  return undefined;
});

const tryRollbackTask = async () => {
  const navigateToRollbackIssue = () => {
    if (!state.rollbackIssue) return;
    router.push(`/issue/${state.rollbackIssue.id}`);
  };

  if (state.rollbackIssue) {
    return navigateToRollbackIssue();
  }

  try {
    state.loading = true;

    const issueName = [
      `Rollback`,
      `#${issue.value.id}`,
      `${issue.value.name}`,
    ].join(" ");

    const description = [
      "The original SQL statement:",
      `${payload.value!.statement}`,
    ].join("\n");

    router.push({
      name: "workspace.issue.detail",
      params: {
        issueSlug: "new",
      },
      query: {
        template: "bb.issue.database.data.update",
        name: issueName,
        project: issue.value.project.id,
        databaseList: [task.value.database!.id].join(","),
        rollbackIssueId: issue.value.id,
        rollbackTaskIdList: [task.value.id].join(","),
        sql: payload.value!.rollbackStatement!,
        description,
      },
    });
  } finally {
    state.loading = false;
  }
};
</script>
