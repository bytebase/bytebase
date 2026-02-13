<template>
  <div class="flex flex-col xl:flex-row xl:items-center xl:gap-x-1">
    <NEllipsis
      :expand-trigger="expandTrigger || undefined"
      :line-clamp="lineClamp"
      :tooltip="false"
    >
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
import { NEllipsis } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  getDateForPbTimestampProtoEs,
  getTimeForPbTimestampProtoEs,
} from "@/types";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";

export type CommentLink = {
  title: string;
  link: string;
};

const props = withDefaults(
  defineProps<{
    taskRun: TaskRun;
    expandTrigger?: false | "click";
    lineClamp?: number;
  }>(),
  {
    expandTrigger: "click",
    lineClamp: 2,
  }
);

const { t } = useI18n();

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
  }
  return taskRun.detail || "-";
});

const commentLink = computed((): CommentLink => {
  const { taskRun } = props;
  if (taskRun.status === TaskRun_Status.FAILED) {
    if (comment.value.includes("version")) {
      return {
        title: t("common.troubleshoot"),
        link: "https://docs.bytebase.com/change-database/troubleshoot/?source=console#duplicate-version",
      };
    }
  }
  return {
    title: "",
    link: "",
  };
});
</script>
