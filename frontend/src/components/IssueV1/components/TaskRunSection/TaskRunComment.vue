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

import {
  TaskRun,
  TaskRun_Status,
  Task_Type,
} from "@/types/proto/v1/rollout_service";
import { unknownTask } from "@/types";
import { extractTaskUID, flattenTaskV1List } from "@/utils";
import { databaseForTask, useIssueContext } from "../../logic";

export type CommentLink = {
  title: string;
  link: string;
};

const props = defineProps<{
  taskRun: TaskRun;
}>();

const { issue } = useIssueContext();
const { t } = useI18n();

const comment = computed(() => {
  const { taskRun } = props;
  if (taskRun.status === TaskRun_Status.FAILED) {
    // return taskRun.
    // TBD
    return "TBD: taskRun.result.detail";
  }
  // Returns result detail if we get the result, otherwise, returns the comment.
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
      case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC:
      case Task_Type.DATABASE_DATA_UPDATE: {
        const db = databaseForTask(issue.value, task);
        // const link = `/${db.name}/changeHistories/${changeHistorySlug(
        //   taskRun.result.migrationId!,
        //   taskRun.result.version!
        // )}`;
        const link = `TBD: make change history link for db ${db.databaseName}`;
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
