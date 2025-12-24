<template>
  <NDataTable
    size="small"
    :row-key="rowKey"
    :columns="columnList"
    :data="sortedTaskRuns"
    :striped="true"
    :bordered="true"
  />

  <Drawer v-model:show="taskRunDetailContext.show">
    <DrawerContent
      :title="$t('common.detail')"
      style="width: calc(100vw - 14rem)"
    >
      <TaskRunDetail
        v-if="taskRunDetailContext.taskRun"
        :key="taskRunDetailContext.taskRun.name"
        :task-run="taskRunDetailContext.taskRun"
        :database="getDatabaseForTaskRun(taskRunDetailContext.taskRun)"
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import type { Duration } from "@bufbuild/protobuf/wkt";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import type { DataTableColumn } from "naive-ui";
import { NButton, NDataTable } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import TaskRunDetail from "@/components/IssueV1/components/TaskRunSection/TaskRunDetail.vue";
import TaskRunStatusIcon from "@/components/IssueV1/components/TaskRunSection/TaskRunStatusIcon.vue";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import { usePlanContextWithRollout } from "@/components/Plan/logic";
import { Drawer, DrawerContent } from "@/components/v2";
import { useCurrentProjectV1 } from "@/store";
import {
  getDateForPbTimestampProtoEs,
  getTimeForPbTimestampProtoEs,
} from "@/types";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask, extractTaskUID, humanizeDurationV1 } from "@/utils";
import TaskRunComment from "./TaskRunComment.vue";

const props = defineProps<{
  taskRuns: TaskRun[];
  showDatabaseColumn?: boolean; // Whether to show database column
}>();

// Sort task runs by creation time descending (newest first)
const sortedTaskRuns = computed(() => {
  return [...props.taskRuns].sort((a, b) => {
    const timeA = a.createTime ? Number(a.createTime.seconds) : 0;
    const timeB = b.createTime ? Number(b.createTime.seconds) : 0;
    return timeB - timeA; // Descending order
  });
});

const { t } = useI18n();
const { project } = useCurrentProjectV1();
const { rollout } = usePlanContextWithRollout();

// Helper function to get task for a given task run
const getTaskForTaskRun = (taskRun: TaskRun): Task | undefined => {
  if (!rollout.value) return undefined;

  // Search through all stages and tasks in the rollout
  for (const stage of rollout.value.stages) {
    for (const task of stage.tasks) {
      if (extractTaskUID(task.name) === extractTaskUID(taskRun.name)) {
        return task;
      }
    }
  }
  return undefined;
};

// Helper function to get database for a task run
const getDatabaseForTaskRun = (taskRun: TaskRun) => {
  const task = getTaskForTaskRun(taskRun);
  return task ? databaseForTask(project.value, task) : undefined;
};

const taskRunDetailContext = ref<{
  show: boolean;
  taskRun?: TaskRun;
}>({
  show: false,
});

const rowKey = (taskRun: TaskRun) => {
  return taskRun.name;
};

const columnList = computed((): DataTableColumn<TaskRun>[] => {
  return [
    {
      key: "status",
      width: 48,
      render: (taskRun: TaskRun) => {
        return <TaskRunStatusIcon status={taskRun.status} />;
      },
    },
    ...(props.showDatabaseColumn
      ? [
          {
            key: "database",
            title: t("common.database"),
            width: 200,
            resizable: true,
            render: (taskRun: TaskRun) => {
              const db = getDatabaseForTaskRun(taskRun);
              return db ? (
                <DatabaseDisplay database={db.name} showEnvironment={true} />
              ) : (
                "-"
              );
            },
          },
        ]
      : []),
    {
      key: "detail",
      title: t("common.detail"),
      ellipsis: true,
      resizable: true,
      render: (taskRun: TaskRun) => {
        return <TaskRunComment taskRun={taskRun} />;
      },
    },
    {
      key: "createTime",
      title: t("task.created"),
      width: 140,
      render: (taskRun: TaskRun) => (
        <HumanizeDate date={getDateForPbTimestampProtoEs(taskRun.createTime)} />
      ),
    },
    {
      key: "startTime",
      title: t("task.started"),
      width: 140,
      render: (taskRun: TaskRun) =>
        taskRun.startTime ? (
          <HumanizeDate
            date={getDateForPbTimestampProtoEs(taskRun.startTime)}
          />
        ) : (
          "-"
        ),
    },
    {
      key: "executionDuration",
      title: t("task.execution-time"),
      width: 120,
      render: (taskRun: TaskRun) => {
        const duration = executionDurationOfTaskRun(taskRun);
        return duration ? humanizeDurationV1(duration) : "-";
      },
    },
    {
      key: "actions",
      title: "",
      width: 80,
      render: (taskRun: TaskRun) =>
        shouldShowDetailButton(taskRun) ? (
          <NButton size="tiny" onClick={() => showDetail(taskRun)}>
            {t("common.detail")}
          </NButton>
        ) : null,
    },
  ];
});

const executionDurationOfTaskRun = (taskRun: TaskRun): Duration | undefined => {
  const { startTime, updateTime } = taskRun;
  if (!startTime || !updateTime) {
    return undefined;
  }
  if (Number(startTime.seconds) === 0) {
    return undefined;
  }
  if (taskRun.status === TaskRun_Status.RUNNING) {
    const elapsedMS = Date.now() - getTimeForPbTimestampProtoEs(startTime);
    return create(DurationSchema, {
      seconds: BigInt(Math.floor(elapsedMS / 1000)),
      nanos: (elapsedMS % 1000) * 1e6,
    });
  }
  const startMS = getTimeForPbTimestampProtoEs(startTime);
  const updateMS = getTimeForPbTimestampProtoEs(updateTime);
  const elapsedMS = updateMS - startMS;
  return create(DurationSchema, {
    seconds: BigInt(Math.floor(elapsedMS / 1000)),
    nanos: (elapsedMS % 1000) * 1e6,
  });
};

const shouldShowDetailButton = (taskRun: TaskRun) => {
  return [
    TaskRun_Status.RUNNING,
    TaskRun_Status.DONE,
    TaskRun_Status.FAILED,
    TaskRun_Status.CANCELED,
  ].includes(taskRun.status);
};

const showDetail = (taskRun: TaskRun) => {
  taskRunDetailContext.value = {
    show: true,
    taskRun,
  };
};
</script>
