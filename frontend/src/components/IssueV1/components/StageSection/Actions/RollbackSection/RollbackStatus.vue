<template>
  <div class="text-sm font-medium">
    <template v-if="!config?.rollbackEnabled">
      <LogButton />
    </template>
    <template v-else>
      <template v-if="config?.rollbackSqlStatus === RollbackSqlStatus.FAILED">
        <div class="tooltip-wrapper">
          <NTooltip>
            <template #trigger>
              <span class="text-error">{{ $t("task.rollback.failed") }}</span>
            </template>

            <div v-if="config.rollbackError" class="max-w-[20rem]">
              {{ config.rollbackError }}
            </div>
          </NTooltip>
        </div>
      </template>
      <template
        v-else-if="config?.rollbackSqlStatus === RollbackSqlStatus.DONE"
      >
        <template
          v-if="taskRollbackBy && rollbackByIssue.uid !== String(UNKNOWN_ID)"
        >
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
          <NButton
            v-if="allowPreviewRollback"
            size="small"
            @click="tryRollbackTask"
          >
            {{ $t("task.rollback.preview-rollback-issue") }}
          </NButton>
          <NTooltip v-else :disabled="!!config?.rollbackSheet">
            <template #trigger>
              <NButton :disabled="true" size="small" tag="div">
                {{ $t("task.rollback.preview-rollback-issue") }}
              </NButton>
            </template>

            <div v-if="!config?.rollbackSheet" class="whitespace-pre-line">
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
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { NTooltip } from "naive-ui";

import {
  ActivityIssueCommentCreatePayload,
  TaskRollbackBy,
  UNKNOWN_ID,
  unknownIssue,
} from "@/types";
import { extractSheetUID } from "@/utils";
import { LogEntity_Action } from "@/types/proto/v1/logging_service";
import { useActivityV1Store, useSheetV1Store } from "@/store";
import { Task_DatabaseDataUpdate_RollbackSqlStatus as RollbackSqlStatus } from "@/types/proto/v1/rollout_service";
import LogButton from "./LogButton.vue";
import LoggingButton from "./LoggingButton.vue";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { databaseForTask, useIssueContext } from "@/components/IssueV1";
import { useRollbackContext } from "./common";
import IssueStatusIcon from "../../../IssueStatusIcon.vue";

type LocalState = {
  loading: boolean;
};

const router = useRouter();
const sheetV1Store = useSheetV1Store();

const { issue, selectedTask: task } = useIssueContext();
const { allowRollback } = useRollbackContext();

const config = computed(() => task.value.databaseDataUpdate);

const state = reactive<LocalState>({
  loading: false,
});

const allowPreviewRollback = computed(() => {
  if (!allowRollback.value) {
    return false;
  }
  if (!config.value?.rollbackSheet) {
    return false;
  }
  return true;
});

const taskRollbackBy = computed((): TaskRollbackBy | undefined => {
  const activityList = useActivityV1Store().getActivityListByIssue(
    issue.value.uid
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

// const rollbackByIssue = useIssueById(
//   computed(() => taskRollbackBy.value?.rollbackByIssueId ?? UNKNOWN_ID)
// );
const rollbackByIssue = ref(unknownIssue()); // TODO

const tryRollbackTask = async () => {
  // const navigateToRollbackIssue = () => {
  //   if (!state.rollbackIssue) return;
  //   router.push(`/issue/${state.rollbackIssue.uid}`);
  // };

  // if (state.rollbackIssue) {
  //   return navigateToRollbackIssue();
  // }

  if (!config.value) return;
  try {
    state.loading = true;

    const issueName = [
      `Rollback`,
      `#${issue.value.uid}`,
      `${issue.value.title}`,
    ].join(" ");

    const originalSheet = await sheetV1Store.getOrFetchSheetByName(
      config.value.sheet
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
        project: issue.value.projectEntity.uid,
        databaseList: [databaseForTask(issue.value, task.value).uid].join(","),
        rollbackIssueId: issue.value.uid,
        rollbackTaskIdList: [task.value.uid].join(","),
        sheetId: extractSheetUID(config.value.rollbackSheet),
        description,
      },
    });
  } finally {
    state.loading = false;
  }
};
</script>
