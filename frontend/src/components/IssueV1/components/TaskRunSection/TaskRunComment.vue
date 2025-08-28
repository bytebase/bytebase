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
import { useCurrentProjectV1 } from "@/store";
import { getProjectIdRolloutUidStageUidTaskUid } from "@/store/modules/v1/common";
import { useTaskRunLogStore } from "@/store/modules/v1/taskRunLog";
import {
  unknownTask,
  getTimeForPbTimestampProtoEs,
  getDateForPbTimestampProtoEs,
} from "@/types";
import {
  TaskRun_Status,
  Task_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { isPostgresFamily } from "@/types/v1/instance";
import { databaseForTask } from "@/utils";
import { databaseV1Url, extractTaskUID, flattenTaskV1List } from "@/utils";
import { extractChangelogUID } from "@/utils/v1/changelog";
import { useIssueContext } from "../../logic";
import { displayTaskRunLogEntryType } from "./TaskRunLogTable/common";

export type CommentLink = {
  title: string;
  link: string;
};

const props = defineProps<{
  taskRun: TaskRun;
}>();

const { issue } = useIssueContext();
const { project } = useCurrentProjectV1();
const { t } = useI18n();
const taskRunLogStore = useTaskRunLogStore();

const earliestAllowedTime = computed(() => {
  return props.taskRun.runTime
    ? getTimeForPbTimestampProtoEs(props.taskRun.runTime)
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
      if (cause?.cause?.case === "task") {
        return t("task-run.status.waiting-task", {
          time: getDateForPbTimestampProtoEs(
            taskRun.schedulerInfo.reportTime
          )?.toLocaleString(),
        });
      }
    }
    return t("task-run.status.enqueued");
  } else if (taskRun.status === TaskRun_Status.RUNNING) {
    if (taskRun.schedulerInfo) {
      const cause = taskRun.schedulerInfo.waitingCause;
      if (cause?.cause?.case === "connectionLimit") {
        return t("task-run.status.waiting-connection", {
          time: getDateForPbTimestampProtoEs(
            taskRun.schedulerInfo.reportTime
          )?.toLocaleString(),
        });
      }
      if (cause?.cause?.case === "task") {
        return t("task-run.status.waiting-task", {
          time: getDateForPbTimestampProtoEs(
            taskRun.schedulerInfo.reportTime
          )?.toLocaleString(),
        });
      }
      if (cause?.cause?.case === "parallelTasksLimit") {
        return t("task-run.status.waiting-max-tasks-per-rollout", {
          time: getDateForPbTimestampProtoEs(
            taskRun.schedulerInfo.reportTime
          )?.toLocaleString(),
        });
      }
    }

    const taskRunLog = taskRunLogStore.getTaskRunLog(taskRun.name);
    const lastLogEntry = last(taskRunLog.entries);
    if (lastLogEntry) {
      return displayTaskRunLogEntryType(lastLogEntry.type);
    }
  }
  return taskRun.detail || "-";
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
    const waitingCause = taskRun.schedulerInfo?.waitingCause;
    if (waitingCause?.cause?.case === "task") {
      const task = waitingCause.cause.value;
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
      case Task_Type.DATABASE_SCHEMA_UPDATE:
      case Task_Type.DATABASE_SCHEMA_UPDATE_SDL:
      case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST:
      case Task_Type.DATABASE_DATA_UPDATE: {
        if (taskRun.changelog === "") {
          return {
            title: "",
            link: "",
          };
        }
        const db = databaseForTask(project.value, task);
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
    const db = databaseForTask(project.value, task);
    // Cast a wide net to catch migration version error
    if (comment.value.includes("version")) {
      return {
        title: t("common.troubleshoot"),
        link: "https://docs.bytebase.com/change-database/troubleshoot/?source=console#duplicate-version",
      };
    } else if (isPostgresFamily(db.instanceResource.engine)) {
      return {
        title: t("common.troubleshoot"),
        link: "https://docs.bytebase.com/change-database/troubleshoot/?source=console#postgresql",
      };
    }
  }
  return {
    title: "",
    link: "",
  };
});
</script>
