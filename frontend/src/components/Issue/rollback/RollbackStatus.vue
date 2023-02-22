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
      <template v-else-if="false && payload?.rollbackSqlStatus === 'DONE'">
        <BBTooltipButton
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
import dayjs from "dayjs";

import type {
  ActivityCreate,
  Issue,
  IssueCreate,
  MigrationContext,
  Task,
  TaskDatabaseDataUpdatePayload,
} from "@/types";
import { BBTooltipButton } from "@/bbkit";
import { useIssueLogic } from "../logic";
import { useActivityStore, useIssueStore } from "@/store";
import { useRollbackLogic } from "./common";
import LogButton from "./LogButton.vue";
import LoggingButton from "./LoggingButton.vue";

type LocalState = {
  loading: boolean;
  rollbackIssue: Issue | undefined;
};

const router = useRouter();
const issueStore = useIssueStore();

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

    const rollbackContext: MigrationContext = {
      detailList: [
        {
          migrationType: "DATA",
          databaseId: task.value.database!.id,
          statement: payload.value!.rollbackStatement!,
          earliestAllowedTs: 0,
          rollbackDetail: {
            issueId: issue.value.id,
            taskId: task.value.id,
          },
        },
      ],
    };

    const datetime = `${dayjs(issue.value.createdTs * 1000).format(
      "MM-DD HH:mm:ss"
    )}`;
    const tz = `UTC${dayjs().format("ZZ")}`;

    const issueName = [
      `[${task.value.database!.name}]`,
      `Rollback ${task.value.name}`,
      `in #${issue.value.id}`,
      `@${datetime} ${tz}`,
    ].join(" ");

    const description = `The original SQL statement:
    ${payload.value!.statement}`;

    const issueCreate: IssueCreate = {
      type: "bb.issue.database.data.update",
      projectId: issue.value.project.id,
      name: issueName,
      description,
      payload: {},
      // Use the same assignee as the original issue
      assigneeId: issue.value.assignee.id,
      createContext: rollbackContext,
    };

    const createdIssue = await issueStore.createIssue(issueCreate);
    state.rollbackIssue = createdIssue;

    await createRollbackCommentActivity(task.value, issue.value, createdIssue);

    navigateToRollbackIssue();
  } finally {
    state.loading = false;
  }
};

const createRollbackCommentActivity = async (
  task: Task,
  issue: Issue,
  newIssue: Issue
) => {
  const comment = [
    "Rollback task",
    `[${task.name}]`,
    `in issue #${newIssue.id}`,
  ].join(" ");

  const createActivity: ActivityCreate = {
    type: "bb.issue.comment.create",
    containerId: issue.id,
    comment,
  };
  try {
    await useActivityStore().createActivity(createActivity);
  } catch {
    // do nothing
    // failing to comment to won't be too bad
  }
};
</script>
