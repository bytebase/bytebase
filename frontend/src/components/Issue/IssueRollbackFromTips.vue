<template>
  <div
    v-if="shouldShowTips"
    class="mt-1 text-sm text-control-light flex flex-row items-center space-x-1"
  >
    <heroicons-outline:arrow-uturn-left class="w-4 h-4" />
    <i18n-t keypath="issue.will-rollback" tag="span">
      <template #link>
        <router-link :to="rollbackIssueLink" class="normal-link">
          <i18n-t keypath="issue.issue-link-with-task">
            <template #issue>#{{ rollbackFromIssue.id }}</template>
            <template #task>[{{ rollbackFromTask.name }}]</template>
          </i18n-t>
        </router-link>
      </template>
    </i18n-t>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vue-router";
import { getRollbackTaskMappingFromQuery } from "@/plugins/issue/logic/initialize/standard";
import { useIssueById } from "@/store";
import {
  RollbackDetail,
  Task,
  TaskCreate,
  TaskDatabaseDataUpdatePayload,
} from "@/types";
import { unknown, UNKNOWN_ID } from "@/types";
import { buildIssueLinkWithTask } from "@/utils";
import { flattenTaskList, useIssueLogic } from "./logic";

const { create, selectedTask } = useIssueLogic();
const route = useRoute();

const payload = computed(() => {
  if (create.value) return undefined;

  const task = selectedTask.value as Task;
  if (task.type === "bb.task.database.data.update") {
    return task.payload as TaskDatabaseDataUpdatePayload | undefined;
  }
  return undefined;
});

const rollbackDetail = computed((): RollbackDetail | undefined => {
  if (!create.value) return undefined;
  // Try to find out the relationship between databaseId and rollback issue/task
  // Id from the URL query.
  const task = selectedTask.value as TaskCreate;
  const { databaseId } = task;
  if (!databaseId || databaseId === UNKNOWN_ID) return undefined;

  const mapping = getRollbackTaskMappingFromQuery(route);
  return mapping.get(databaseId);
});

const rollbackFromIssueId = computed(() => {
  if (create.value) {
    return rollbackDetail.value?.issueId || UNKNOWN_ID;
  }
  return payload.value?.rollbackFromIssueId || UNKNOWN_ID;
});

const rollbackFromTaskId = computed(() => {
  if (create.value) {
    return rollbackDetail.value?.taskId || UNKNOWN_ID;
  }
  return payload.value?.rollbackFromTaskId || UNKNOWN_ID;
});

const rollbackFromIssue = useIssueById(
  rollbackFromIssueId,
  true /* Lazy fetch */
);

const rollbackFromTask = computed((): Task => {
  const issue = rollbackFromIssue.value;
  if (issue.id === UNKNOWN_ID) return unknown("TASK");

  const taskId = rollbackFromTaskId.value;
  if (taskId === UNKNOWN_ID) return unknown("TASK");

  const task = flattenTaskList<Task>(issue).find((t) => t.id === taskId);
  if (task) {
    return task;
  }

  return unknown("TASK");
});

const shouldShowTips = computed(() => {
  // Show the tips when rollbackFromIssue and rollbackFromTask are both ready.
  return (
    rollbackFromIssue.value.id !== UNKNOWN_ID &&
    rollbackFromTask.value.id !== UNKNOWN_ID
  );
});

const rollbackIssueLink = computed(() => {
  if (!shouldShowTips.value) return "";
  return buildIssueLinkWithTask(
    rollbackFromIssue.value,
    rollbackFromTask.value
  );
});
</script>
