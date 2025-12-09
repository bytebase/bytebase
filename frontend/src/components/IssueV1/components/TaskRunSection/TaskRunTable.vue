<template>
  <NDataTable
    size="small"
    :row-key="rowKey"
    :columns="columnList"
    :data="taskRunList"
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
        :database="databaseForTask(project, selectedTask)"
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import { type Duration, DurationSchema } from "@bufbuild/protobuf/wkt";
import { type DataTableColumn, NButton, NDataTable } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useCurrentProjectV1 } from "@/store";
import {
  getDateForPbTimestampProtoEs,
  getTimeForPbTimestampProtoEs,
} from "@/types";
import {
  type TaskRun,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask, humanizeDurationV1 } from "@/utils";
import { useIssueContext } from "../../logic";
import TaskRunComment from "./TaskRunComment.vue";
import TaskRunDetail from "./TaskRunDetail.vue";
import TaskRunStatusIcon from "./TaskRunStatusIcon.vue";

defineOptions({
  inheritAttrs: false,
});

defineProps<{
  taskRunList: TaskRun[];
}>();

const { t } = useI18n();
const { project } = useCurrentProjectV1();
const { selectedTask } = useIssueContext();

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
      title: "",
      width: 30,
      render: (taskRun: TaskRun) => (
        <TaskRunStatusIcon status={taskRun.status} />
      ),
    },
    {
      key: "comment",
      title: t("task.comment"),
      width: "60%",
      className: "",
      minWidth: 140,
      resizable: true,
      render: (taskRun: TaskRun) => (
        <div class="flex flex-row justify-start items-center">
          <TaskRunComment taskRun={taskRun} />
        </div>
      ),
    },
    {
      key: "createTime",
      title: t("task.created"),
      width: 100,
      render: (taskRun: TaskRun) => (
        <HumanizeDate date={getDateForPbTimestampProtoEs(taskRun.createTime)} />
      ),
    },
    {
      key: "startTime",
      title: t("task.started"),
      width: 100,
      render: (taskRun: TaskRun) => (
        <HumanizeDate date={getDateForPbTimestampProtoEs(taskRun.startTime)} />
      ),
    },
    {
      key: "executionDuration",
      title: t("task.execution-time"),
      width: 120,
      render: (taskRun: TaskRun) => {
        return humanizeDurationV1(executionDurationOfTaskRun(taskRun));
      },
    },
    {
      key: "actions",
      title: "",
      width: 60,
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
  if (startTime.seconds === 0n) {
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
