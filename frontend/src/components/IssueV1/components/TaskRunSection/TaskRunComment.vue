<template>
  {{ comment }}

  <template v-if="commentLink.link">
    <router-link
      class="bb-comment-link ml-1 normal-link"
      :to="commentLink.link"
    >
      {{ commentLink.title }}
    </router-link>
  </template>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { unknownTask } from "@/types";
import {
  TaskRun,
  TaskRun_ExecutionStatus,
  TaskRun_Status,
  Task_Type,
} from "@/types/proto/v1/rollout_service";
import {
  changeHistoryLinkRaw,
  extractChangeHistoryUID,
  extractTaskUID,
  flattenTaskV1List,
} from "@/utils";
import { databaseForTask, specForTask, useIssueContext } from "../../logic";

export type CommentLink = {
  title: string;
  link: string;
};

const props = defineProps<{
  taskRun: TaskRun;
}>();

const { issue } = useIssueContext();
const { t } = useI18n();

const task = computed(() => {
  const taskUID = extractTaskUID(props.taskRun.name);
  const task =
    flattenTaskV1List(issue.value.rolloutEntity).find(
      (task) => task.uid === taskUID
    ) ?? unknownTask();
  return task;
});

const earliestAllowedTime = computed(() => {
  const spec = specForTask(issue.value.planEntity, task.value);
  return spec?.earliestAllowedTime ? spec.earliestAllowedTime.getTime() : null;
});

const comment = computed(() => {
  const { taskRun } = props;
  if (taskRun.status === TaskRun_Status.PENDING) {
    if (earliestAllowedTime.value) {
      return t("task-run.status.enqueued-with-rollout-time", {
        time: new Date(earliestAllowedTime.value).toLocaleString(),
      });
    }
    return t("task-run.status.enqueued");
  } else if (taskRun.status === TaskRun_Status.RUNNING) {
    if (taskRun.executionStatus === TaskRun_ExecutionStatus.PRE_EXECUTING) {
      return t("task-run.status.dumping-schema-before-executing-sql");
    } else if (taskRun.executionStatus === TaskRun_ExecutionStatus.EXECUTING) {
      return t("task-run.status.executing-sql");
    } else if (
      taskRun.executionStatus === TaskRun_ExecutionStatus.POST_EXECUTING
    ) {
      return t("task-run.status.dumping-schema-after-executing-sql");
    }
  }
  return taskRun.detail;
});

const commentLink = computed((): CommentLink => {
  const { taskRun } = props;
  const taskUID = extractTaskUID(taskRun.name);
  const task =
    flattenTaskV1List(issue.value.rolloutEntity).find(
      (task) => task.uid === taskUID
    ) ?? unknownTask();
  if (taskRun.status === TaskRun_Status.DONE) {
    switch (task.type) {
      case Task_Type.DATABASE_SCHEMA_BASELINE:
      case Task_Type.DATABASE_SCHEMA_UPDATE:
      case Task_Type.DATABASE_SCHEMA_UPDATE_SDL:
      case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER:
      case Task_Type.DATABASE_DATA_UPDATE: {
        const db = databaseForTask(issue.value, task);
        const link = changeHistoryLinkRaw(
          db.name,
          extractChangeHistoryUID(taskRun.changeHistory),
          taskRun.schemaVersion
        );
        return {
          title: t("task.view-change"),
          link,
        };
      }
    }
  } else if (taskRun.status === TaskRun_Status.FAILED) {
    // TBD: taskRun.code is not available
    // if (taskRun.code == MigrationErrorCode.MIGRATION_SCHEMA_MISSING) {
    //   return {
    //     title: "Check instance",
    //     link: `/instance/${instanceSlug(task.instance)}`,
    //   };
    // } else if (
    //   task.database &&
    //   (taskRun.code == MigrationErrorCode.MIGRATION_ALREADY_APPLIED ||
    //     taskRun.code == MigrationErrorCode.MIGRATION_OUT_OF_ORDER ||
    //     taskRun.code == MigrationErrorCode.MIGRATION_BASELINE_MISSING)
    // ) {
    //   return {
    //     title: t("task.view-change-history"),
    //     link: `/db/${databaseSlug(task.database!)}#change-history`,
    //   };
    // }
  }
  return {
    title: "",
    link: "",
  };
});
</script>
