<template>
  <div class="flex flex-col xl:flex-row xl:items-center xl:space-x-1">
    <NEllipsis expand-trigger="click" line-clamp="2" :tooltip="false">
      <span>{{ comment }}</span>
    </NEllipsis>
    <template v-if="commentLink.link">
      <router-link
        v-if="commentLink.link.startsWith('/')"
        class="inline normal-link shrink-0"
        :to="commentLink.link"
      >
        {{ commentLink.title }}
      </router-link>
      <a
        v-else
        class="inline normal-link shrink-0"
        :href="commentLink.link"
        target="_blank"
        rel="noopener noreferrer"
      >
        {{ commentLink.title }}
      </a>
    </template>
  </div>
</template>

<script setup lang="tsx">
import { last } from "lodash-es";
import { NEllipsis } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { getProjectIdRolloutUidStageUidTaskUid } from "@/store/modules/v1/common";
import {
  unknownTask,
  isPostgresFamily,
  type ComposedTaskRun,
  getTimeForPbTimestamp,
} from "@/types";
import { TaskRun_Status, Task_Type } from "@/types/proto/v1/rollout_service";
import { databaseV1Url, extractTaskUID, flattenTaskV1List } from "@/utils";
import { extractChangelogUID } from "@/utils/v1/changelog";
import { databaseForTask, specForTask, useIssueContext } from "../../logic";
import { displayTaskRunLogEntryType } from "./TaskRunLogTable/common";

export type CommentLink = {
  title: string;
  link: string;
};

const props = defineProps<{
  taskRun: ComposedTaskRun;
}>();

const { issue } = useIssueContext();
const { t } = useI18n();

const task = computed(() => {
  const taskUID = extractTaskUID(props.taskRun.name);
  const task =
    flattenTaskV1List(issue.value.rolloutEntity).find(
      (task) => extractTaskUID(task.name) === taskUID
    ) ?? unknownTask();
  return task;
});

const earliestAllowedTime = computed(() => {
  const spec = specForTask(issue.value.planEntity, task.value);
  return spec?.earliestAllowedTime
    ? getTimeForPbTimestamp(spec.earliestAllowedTime)
    : null;
});

const comment = computed(() => {
  const { taskRun } = props;
  if (taskRun.status === TaskRun_Status.PENDING) {
    if (earliestAllowedTime.value) {
      return t("task-run.status.enqueued-with-rollout-time", {
        time: new Date(earliestAllowedTime.value).toLocaleString(),
      });
    }
    if (taskRun.schedulerInfo) {
      const cause = taskRun.schedulerInfo.waitingCause;
      if (cause?.task) {
        return t("task-run.status.waiting-task");
      }
    }
    return t("task-run.status.enqueued");
  } else if (taskRun.status === TaskRun_Status.RUNNING) {
    if (taskRun.schedulerInfo) {
      const cause = taskRun.schedulerInfo.waitingCause;
      if (cause?.connectionLimit) {
        return t("task-run.status.waiting-connection");
      }
      if (cause?.task) {
        return t("task-run.status.waiting-task");
      }
    }

    const lastLogEntry = last(taskRun.taskRunLog.entries);
    if (!lastLogEntry) {
      return "-";
    }
    return displayTaskRunLogEntryType(lastLogEntry.type);
  }
  return taskRun.detail;
});

const commentLink = computed((): CommentLink => {
  const { taskRun } = props;
  const taskUID = extractTaskUID(taskRun.name);
  const task =
    flattenTaskV1List(issue.value.rolloutEntity).find(
      (task) => extractTaskUID(task.name) === taskUID
    ) ?? unknownTask();
  if (
    taskRun.status === TaskRun_Status.PENDING ||
    taskRun.status === TaskRun_Status.RUNNING
  ) {
    const task = taskRun.schedulerInfo?.waitingCause?.task;
    if (task) {
      const [, , stageUid, taskUid] = getProjectIdRolloutUidStageUidTaskUid(
        task.task
      );
      const link =
        task.issue !== ""
          ? `/${task.issue}?stage=${stageUid}&task=${taskUid}`
          : `/${task.task}`;
      return {
        title: t("common.blocking-task"),
        link: link,
      };
    }
  } else if (taskRun.status === TaskRun_Status.DONE) {
    switch (task.type) {
      case Task_Type.DATABASE_SCHEMA_BASELINE:
      case Task_Type.DATABASE_SCHEMA_UPDATE:
      case Task_Type.DATABASE_SCHEMA_UPDATE_SDL:
      case Task_Type.DATABASE_DATA_UPDATE: {
        if (taskRun.changelog === "") {
          return {
            title: "",
            link: "",
          };
        }
        const db = databaseForTask(issue.value, task);
        const link = `${databaseV1Url(
          db
        )}/changelogs/${extractChangelogUID(taskRun.changelog)}`;
        return {
          title: t("task.view-change"),
          link,
        };
      }
    }
  } else if (taskRun.status === TaskRun_Status.FAILED) {
    const db = databaseForTask(issue.value, task);
    // Cast a wide net to catch migration version error
    if (comment.value.includes("version")) {
      return {
        title: t("common.troubleshoot"),
        link: "https://www.bytebase.com/docs/change-database/troubleshoot/?source=console#duplicate-version",
      };
    } else if (isPostgresFamily(db.instanceResource.engine)) {
      return {
        title: t("common.troubleshoot"),
        link: "https://www.bytebase.com/docs/change-database/troubleshoot/?source=console#postgresql",
      };
    }
  }
  return {
    title: "",
    link: "",
  };
});
</script>
