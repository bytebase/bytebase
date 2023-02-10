<template>
  <button
    v-if="showRollbackButton"
    :disabled="!allowRollback || state.buttonState === ButtonState.LOADING"
    type="button"
    class="mt-0.5 px-3 inline-flex items-center gap-x-2 border border-control-border rounded-sm text-control bg-control-bg hover:bg-control-bg-hover text-sm leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2 relative disabled:cursor-not-allowed disabled:text-gray-400 disabled:hover:bg-control-bg"
    @click="tryRollbackTask"
  >
    {{ $t("common.rollback") }}
    <template v-if="state.buttonState === ButtonState.LOADING">
      <BBSpin class="w-4 h-4 -mr-1.5" />
    </template>
  </button>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import dayjs from "dayjs";

import type {
  ActivityCreate,
  Issue,
  IssueCreate,
  RollbackContext,
  Task,
  TaskDatabaseDataUpdatePayload,
} from "@/types";
import { useIssueLogic } from "./logic";
import { useActivityStore, useCurrentUser, useIssueStore } from "@/store";
import { hasProjectPermission, hasWorkspacePermission, isDev } from "@/utils";

enum ButtonState {
  DEFAULT,
  LOADING,
  GENERATED,
}

type LocalState = {
  buttonState: ButtonState;
  rollbackIssue: Issue | undefined;
};

const state = reactive<LocalState>({
  buttonState: ButtonState.DEFAULT,
  rollbackIssue: undefined,
});

const router = useRouter();
const issueStore = useIssueStore();
const currentUser = useCurrentUser();

const { issue, create, selectedTask } = useIssueLogic();

const showRollbackButton = computed(() => {
  if (!isDev()) return false;

  if (create.value) return false;

  const issueEntity = issue.value as Issue;
  const task = selectedTask.value as Task;

  return (
    issueEntity.type === "bb.issue.database.data.update" &&
    task.type === "bb.task.database.data.update" &&
    task.status === "DONE" &&
    task.database?.instance.engine === "MYSQL"
  );
});

const allowRollback = computed(() => {
  if (!showRollbackButton.value) return false;

  const issueEntity = issue.value as Issue;
  const user = currentUser.value;

  if (user.id === issueEntity.creator.id) {
    // Allowed to the issue creator
    return true;
  }

  if (user.id === issueEntity.assignee.id) {
    // Allowed to the issue assignee
    return true;
  }

  const memberInProject = issueEntity.project.memberList.find(
    (member) => member.principal.id === user.id
  );
  if (
    memberInProject?.role &&
    hasProjectPermission(
      "bb.permission.project.admin-database",
      memberInProject.role
    )
  ) {
    // Allowed to the project owner
    return true;
  }

  if (
    hasWorkspacePermission("bb.permission.workspace.manage-issue", user.role)
  ) {
    // Allowed to DBAs and workspace owners
    return true;
  }
  return false;
});

const tryRollbackTask = async () => {
  if (!showRollbackButton.value) return;

  const navigateToRollbackIssue = () => {
    if (!state.rollbackIssue) return;
    router.push(`/issue/${state.rollbackIssue.id}`);
  };

  if (state.buttonState === ButtonState.GENERATED && state.rollbackIssue) {
    return navigateToRollbackIssue();
  }

  try {
    state.buttonState = ButtonState.LOADING;

    const issueEntity = issue.value as Issue;
    const task = selectedTask.value as Task;

    const rollbackContext: RollbackContext = {
      issueId: issueEntity.id,
      taskIdList: [task.id],
    };

    const datetime = `${dayjs(issueEntity.createdTs * 1000).format(
      "MM-DD HH:mm:ss"
    )}`;
    const tz = `UTC${dayjs().format("ZZ")}`;

    const issueName = [
      `[${task.database!.name}]`,
      `Rollback ${task.name}`,
      `in #${issueEntity.id}`,
      `@${datetime} ${tz}`,
    ].join(" ");

    const description = [
      `The original SQL statement:\n${
        (task.payload as TaskDatabaseDataUpdatePayload).statement
      }`,
    ].join("\n");

    const issueCreate: IssueCreate = {
      type: "bb.issue.database.rollback",
      projectId: issueEntity.project.id,
      name: issueName,
      description,
      payload: {},
      // Use the same assignee as the original issue
      assigneeId: issueEntity.assignee.id,
      createContext: rollbackContext,
    };

    const createdIssue = await issueStore.createIssue(issueCreate);
    state.rollbackIssue = createdIssue;

    state.buttonState = ButtonState.GENERATED;

    await createRollbackCommentActivity(task, issueEntity, createdIssue);

    navigateToRollbackIssue();
  } catch {
    state.buttonState = ButtonState.DEFAULT;
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
