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
        <template v-if="taskRollbackBy && rollbackByIssue.id !== UNKNOWN_ID">
          <router-link
            :to="`/issue/${taskRollbackBy.rollbackByIssueId}`"
            class="text-accent inline-flex gap-x-1"
          >
            <span>{{ $t("common.issue") }}</span>
            <span>#{{ taskRollbackBy.rollbackByIssueId }}</span>
            <IssueStatusIcon :issue-status="rollbackByIssue.status" />
          </router-link>
        </template>
        <template v-else>
          <button
            v-if="allowPreviewRollback"
            type="button"
            class="btn-normal !px-3 !py-1"
            @click.prevent="tryRollbackTask"
          >
            {{ $t("task.rollback.preview-rollback-issue") }}
          </button>
          <NTooltip v-else :disabled="!!payload?.rollbackSheetId">
            <template #trigger>
              <div
                class="select-none inline-flex border border-control-border rounded-md bg-control-bg opacity-50 cursor-not-allowed px-3 py-1 text-sm leading-5 font-medium"
              >
                {{ $t("task.rollback.preview-rollback-issue") }}
              </div>
            </template>

            <div v-if="!payload?.rollbackSheetId" class="whitespace-pre-line">
              {{ $t("task.rollback.empty-rollback-statement") }}
              <LearnMoreLink
                url="https://www.bytebase.com/docs/change-database/rollback-data-changes?source=console#why-i-get-the-rollback-sheet-is-empty"
                color="light"
                class="ml-1"
              />
            </div>
          </NTooltip>
        </template>
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
import { useIssueLogic } from "../logic";
import { useRollbackLogic } from "./common";
import IssueStatusIcon from "../IssueStatusIcon.vue";
import LogButton from "./LogButton.vue";
import LoggingButton from "./LoggingButton.vue";
import { useActivityV1Store, useIssueById, useSheetV1Store } from "@/store";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { LogEntity_Action } from "@/types/proto/v1/logging_service";

type LocalState = {
  loading: boolean;
  rollbackIssue: Issue | undefined;
};

const router = useRouter();
const sheetV1Store = useSheetV1Store();

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
  if (!payload.value?.rollbackSheetId) {
    return false;
  }
  return true;
});

const taskRollbackBy = computed((): TaskRollbackBy | undefined => {
  const activityList = useActivityV1Store().getActivityListByIssue(
    issue.value.id
  );
  // Find the latest comment activity with TaskRollbackBy struct if possible.
  for (let i = activityList.length - 1; i >= 0; i--) {
    const activity = activityList[i];
    if (activity.action !== LogEntity_Action.ACTION_ISSUE_COMMENT_CREATE)
      continue;
    const payload = JSON.parse(activity.payload) as
      | ActivityIssueCommentCreatePayload
      | undefined;
    if (!payload) continue;
    const taskRollbackBy = payload.taskRollbackBy;
    if (!taskRollbackBy) continue;
    return taskRollbackBy;
  }
  return undefined;
});

const rollbackByIssue = useIssueById(
  computed(() => taskRollbackBy.value?.rollbackByIssueId ?? UNKNOWN_ID)
);

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

    const originalSheetUID = payload.value!.sheetId!;
    const rollbackSheetUID = payload.value!.rollbackSheetId!;
    const originalSheet = await sheetV1Store.getOrFetchSheetByUID(
      String(originalSheetUID)
    );
    const description = [
      "The original SQL statement:",
      `${new TextDecoder().decode(originalSheet?.content)}`,
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
        sheetId: rollbackSheetUID,
        description,
      },
    });
  } finally {
    state.loading = false;
  }
};
</script>
