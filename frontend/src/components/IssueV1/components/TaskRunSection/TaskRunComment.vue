<template>
  <NEllipsis expand-trigger="click" line-clamp="3" :tooltip="false">
    {{ comment }}
  </NEllipsis>

  <template v-if="commentLink.link">
    <template v-if="commentLink.link.startsWith('/')">
      <router-link class="ml-1 inline normal-link" :to="commentLink.link">
        {{ commentLink.title }}
      </router-link>
    </template>
    <template v-else>
      <a
        class="ml-1 inline normal-link"
        :href="commentLink.link"
        target="_blank"
        rel="noopener noreferrer"
      >
        {{ commentLink.title }}
      </a>
    </template>
  </template>
</template>

<script setup lang="ts">
import { NEllipsis } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { unknownTask, isPostgresFamily } from "@/types";
import type { TaskRun } from "@/types/proto/v1/rollout_service";
import {
  TaskRun_ExecutionStatus,
  TaskRun_Status,
  Task_Type,
} from "@/types/proto/v1/rollout_service";
import {
  databaseV1Url,
  extractChangeHistoryUID,
  extractTaskUID,
  flattenTaskV1List,
  isDatabaseDataExportIssue,
} from "@/utils";
import { databaseForTask, specForTask, useIssueContext } from "../../logic";
import {getProjectIdRolloutUidStageUidTaskUid} from "@/store/modules/v1/common"

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
    if (taskRun.schedulerInfo) {
      const cause = taskRun.schedulerInfo.waitingCause
      if (cause?.connectionLimit) {
        return t("task-run.status.waiting-connection")
      }
      if (cause?.task) {
        return t("task-run.status.waiting-task")
      }
    }
    if (taskRun.executionStatus === TaskRun_ExecutionStatus.PRE_EXECUTING) {
      if (isDatabaseDataExportIssue(issue.value)) {
        return t("task-run.status.preparing-to-export-data");
      }
      return t("task-run.status.dumping-schema-before-executing-sql");
    } else if (taskRun.executionStatus === TaskRun_ExecutionStatus.EXECUTING) {
      if (taskRun.executionDetail) {
        return t("task-run.status.executing-sql-detail", {
          current: taskRun.executionDetail.commandsCompleted + 1,
          total: taskRun.executionDetail.commandsTotal,
          startLine:
            (taskRun.executionDetail.commandStartPosition?.line ?? 0) + 1,
          startColumn:
            (taskRun.executionDetail.commandStartPosition?.column ?? 0) + 1,
          endLine: (taskRun.executionDetail.commandEndPosition?.line ?? 0) + 1,
          endColumn:
            (taskRun.executionDetail.commandEndPosition?.column ?? 0) + 1,
        });
      }
      if (isDatabaseDataExportIssue(issue.value)) {
        return t("task-run.status.exporting-data");
      }
      return t("task-run.status.executing-sql");
    } else if (
      taskRun.executionStatus === TaskRun_ExecutionStatus.POST_EXECUTING
    ) {
      return t("task-run.status.dumping-schema-after-executing-sql");
    }
  } else if (taskRun.status === TaskRun_Status.FAILED) {
    if (
      taskRun.executionDetail?.commandStartPosition &&
      taskRun.executionDetail?.commandEndPosition
    ) {
      return t("task-run.status.failed-sql-detail", {
        startLine:
          (taskRun.executionDetail.commandStartPosition?.line ?? 0) + 1,
        startColumn:
          (taskRun.executionDetail.commandStartPosition?.column ?? 0) + 1,
        endLine: (taskRun.executionDetail.commandEndPosition?.line ?? 0) + 1,
        endColumn:
          (taskRun.executionDetail.commandEndPosition?.column ?? 0) + 1,
        message: taskRun.detail,
      });
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
  if (taskRun.status === TaskRun_Status.RUNNING) {
    const task = taskRun.schedulerInfo?.waitingCause?.task
    if (task) {
      const [, , stageUid, taskUid] = getProjectIdRolloutUidStageUidTaskUid(task.task)
      const link = `/${task.issue}?stage=${stageUid}&task=${taskUid}`
      return {
        title: t("common.blocking-task"),
        link: link,
      }
    }
  } else if (taskRun.status === TaskRun_Status.DONE) {
    switch (task.type) {
      case Task_Type.DATABASE_SCHEMA_BASELINE:
      case Task_Type.DATABASE_SCHEMA_UPDATE:
      case Task_Type.DATABASE_SCHEMA_UPDATE_SDL:
      case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER:
      case Task_Type.DATABASE_DATA_UPDATE: {
        const db = databaseForTask(issue.value, task);
        const link = `${databaseV1Url(
          db
        )}/change-histories/${extractChangeHistoryUID(taskRun.changeHistory)}`;
        return {
          title: t("task.view-change"),
          link,
        };
      }
    }
  } else if (taskRun.status === TaskRun_Status.FAILED) {
    const db = databaseForTask(issue.value, task);
    if (isPostgresFamily(db.instanceResource.engine)) {
      return {
        title: t("common.troubleshoot"),
        link: "https://www.bytebase.com/docs/change-database/troubleshoot/#postgresql",
      };
    }
  }
  return {
    title: "",
    link: "",
  };
});
</script>
