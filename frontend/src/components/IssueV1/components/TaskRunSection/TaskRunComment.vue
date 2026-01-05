<template>
  <div class="flex flex-col xl:flex-row xl:items-center xl:gap-x-1">
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
import { useTaskRunLogStore } from "@/store/modules/v1/taskRunLog";
import {
  getDateForPbTimestampProtoEs,
  getTimeForPbTimestampProtoEs,
  unknownTask,
} from "@/types";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { isPostgresFamily } from "@/types/v1/instance";
import { databaseForTask, extractTaskUID, flattenTaskV1List } from "@/utils";
import { displayTaskRunLogEntryType } from "@/utils/v1/taskRunLog";
import { useIssueContext } from "../../logic";

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
    return t("task-run.status.enqueued");
  } else if (taskRun.status === TaskRun_Status.RUNNING) {
    if (taskRun.schedulerInfo) {
      const cause = taskRun.schedulerInfo.waitingCause;
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
  if (taskRun.status === TaskRun_Status.FAILED) {
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
