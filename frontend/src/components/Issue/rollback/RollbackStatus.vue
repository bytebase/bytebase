<template>
  <div class="text-sm font-medium">
    <template v-if="!payload?.rollbackEnabled">
      <button
        :disabled="!allowRollback"
        class="btn-normal !px-3 !py-1"
        @click="toggleRollback(true)"
      >
        {{ $t("task.rollback.log") }}
      </button>
    </template>
    <template v-else>
      <template v-if="payload?.rollbackError">
        <div class="tooltip-wrapper">
          <NTooltip>
            <template #trigger>
              <span class="text-error">{{ $t("task.rollback.failed") }}</span>
            </template>

            <div class="max-w-[20rem]">{{ payload.rollbackError }}</div>
          </NTooltip>
        </div>
      </template>
      <template v-else-if="payload?.rollbackStatement">
        <button
          :disabled="!allowRollback"
          class="btn-normal !px-3 !py-1"
          @click="tryRollbackTask"
        >
          {{ $t("task.rollback.preview-rollback") }}
        </button>
      </template>
      <template v-else>
        <div
          class="select-none inline-flex items-center border border-control-border rounded-md text-control bg-white text-sm leading-5 font-medium focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
          :class="[
            !allowRollback && 'cursor-not-allowed bg-control-bg opacity-50 ',
          ]"
        >
          <span class="pl-3 pr-0.5">{{ $t("task.rollback.logging") }}</span>
          <span
            class="h-[28px] px-1.5 flex items-center rounded-r-md hover:bg-control-bg-hover cursor-pointer"
            :class="[!allowRollback && 'pointer-events-none']"
            @click="toggleRollback(false)"
          >
            <heroicons-outline:x-mark class="w-3 h-3" />
          </span>
        </div>
      </template>
    </template>
  </div>
</template>
<script lang="ts" setup>
import { computed, reactive, type Ref } from "vue";
import { NTooltip } from "naive-ui";
import dayjs from "dayjs";

import type {
  ActivityCreate,
  Issue,
  IssueCreate,
  RollbackContext,
  Task,
  TaskDatabaseDataUpdatePayload,
} from "@/types";
import { useIssueLogic } from "../logic";
import { useRouter } from "vue-router";
import { useActivityStore, useIssueStore } from "@/store";
import { absolutifyLink, buildIssueLinkWithTask } from "@/utils";
import { useRollbackLogic } from "./common";

type LocalState = {
  loading: boolean;
  rollbackIssue: Issue | undefined;
};

const router = useRouter();
const issueStore = useIssueStore();

const context = useIssueLogic();
const { allowRollback, toggleRollback } = useRollbackLogic();

const issue = context.issue as Ref<Issue>;
const task = context.selectedTask as Ref<Task>;
const payload = computed(
  () => task.value.payload as TaskDatabaseDataUpdatePayload | undefined
);

const state = reactive<LocalState>({
  loading: false,
  rollbackIssue: undefined,
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

    const rollbackContext: RollbackContext = {
      issueId: issue.value.id,
      taskIdList: [task.value.id],
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

    const originalIssueTaskLink = buildIssueLinkWithTask(
      issue.value,
      task.value,
      true /* Keep the URL short and clean */
    );
    const description = [
      `The original task: ${absolutifyLink(originalIssueTaskLink)}`,
      `The original SQL statement: ${payload.value!.statement}`,
    ].join("\n");

    const issueCreate: IssueCreate = {
      type: "bb.issue.database.rollback",
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
