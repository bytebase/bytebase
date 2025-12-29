<template>
  <div v-if="shouldShowSection" class="w-full flex flex-col gap-y-2 py-3">
    <div class="text-sm text-control">
      {{ $t("task-run.failed-runs") }}
    </div>
    <div v-if="isLoading" class="flex items-center justify-center py-4">
      <div class="text-sm text-control-light">Loading task runs...</div>
    </div>
    <NDataTable
      v-else-if="taskRunList.length > 0"
      size="small"
      :row-key="rowKey"
      :columns="columnList"
      :data="taskRunList"
      :max-height="400"
      :row-props="rowProps"
      :pagination="
        taskRunList.length > DEFAULT_TASK_RUNS_PER_PAGE
          ? {
              page: currentPage,
              pageSize: DEFAULT_TASK_RUNS_PER_PAGE,
              showSizePicker: false,
              itemCount: taskRunList.length,
              simple: false,
              size: 'small',
              onUpdatePage: (page: number) => {
                currentPage = page;
              },
            }
          : false
      "
    />
    <div v-else class="text-sm text-control-light py-4">
      No task runs found.
    </div>
    <Drawer v-model:show="taskRunDetailContext.show">
      <DrawerContent
        :title="$t('common.detail')"
        style="width: calc(100vw - 14rem)"
      >
        <TaskRunDetail
          v-if="taskRunDetailContext.taskRun && selectedDatabase"
          :key="taskRunDetailContext.taskRun.name"
          :task-run="taskRunDetailContext.taskRun"
          :database="selectedDatabase"
        />
      </DrawerContent>
    </Drawer>
  </div>
</template>

<script lang="tsx" setup>
import { ExternalLinkIcon } from "lucide-vue-next";
import { type DataTableColumn, NButton, NDataTable } from "naive-ui";
import { computed, reactive, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import TaskRunDetail from "@/components/IssueV1/components/TaskRunSection/TaskRunDetail.vue";
import TaskRunStatusIcon from "@/components/IssueV1/components/TaskRunSection/TaskRunStatusIcon.vue";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import TaskRunComment from "@/components/RolloutV1/components/TaskRunComment.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useDatabaseV1Store } from "@/store";
import { useTaskRunLogStore } from "@/store/modules/v1/taskRunLog";
import { getDateForPbTimestampProtoEs } from "@/types";
import {
  Task_Status,
  type TaskRun,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  extractTaskUID,
  flattenTaskV1List,
} from "@/utils";
import { usePlanContextWithRollout } from "../../logic";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";
import { useSelectedSpec } from "./context";

const DEFAULT_TASK_RUNS_PER_PAGE = 5;

const { t } = useI18n();
const router = useRouter();
const { project } = useCurrentProjectV1();
const { rollout, taskRuns } = usePlanContextWithRollout();
const { selectedSpec } = useSelectedSpec();
const databaseStore = useDatabaseV1Store();
const taskRunLogStore = useTaskRunLogStore();

// Pagination state
const currentPage = ref(1);

const taskRunDetailContext = reactive<{
  show: boolean;
  taskRun?: TaskRun;
}>({
  show: false,
});

const failedSpecTasks = computed(() => {
  const tasks = flattenTaskV1List(rollout.value);
  return tasks.filter(
    (task) =>
      task.specId === selectedSpec.value.id &&
      task.status === Task_Status.FAILED
  );
});

const taskRunList = computed(() => {
  if (failedSpecTasks.value.length === 0) {
    return [];
  }

  const latestFailedTaskRuns: TaskRun[] = [];

  // For each failed task, find its latest task run
  for (const failedTask of failedSpecTasks.value) {
    const taskUID = extractTaskUID(failedTask.name);

    // Get all task runs for this failed task
    const taskRunsForTask = taskRuns.value.filter(
      (taskRun) => extractTaskUID(taskRun.name) === taskUID
    ) as TaskRun[];

    if (taskRunsForTask.length > 0) {
      // Sort by creation time to get the latest first
      const sortedTaskRuns = taskRunsForTask.sort((a, b) => {
        const aTime = a.createTime ? Number(a.createTime.seconds) : 0;
        const bTime = b.createTime ? Number(b.createTime.seconds) : 0;
        return bTime - aTime; // Latest first
      });

      // Get the latest task run (first after sorting)
      const latestTaskRun = sortedTaskRuns[0];
      latestFailedTaskRuns.push(latestTaskRun);
    }
  }

  return latestFailedTaskRuns;
});

// Determine if the section should be shown at all
const shouldShowSection = computed(() => {
  return failedSpecTasks.value.length > 0 && taskRuns.value.length > 0;
});

// Helper function to get the task for a task run
const getTaskForTaskRun = (taskRun: TaskRun) => {
  const taskUID = extractTaskUID(taskRun.name);
  return failedSpecTasks.value.find(
    (task) => extractTaskUID(task.name) === taskUID
  );
};

// Determine if we're in a loading state
const isLoading = computed(() => {
  // We're loading if we have failed tasks, rollout exists, but no task runs loaded yet
  const hasFailedTasks = failedSpecTasks.value.length > 0;
  return (
    hasFailedTasks && rollout.value !== undefined && taskRuns.value.length === 0
  );
});

watchEffect(async () => {
  for (const taskRun of taskRunList.value) {
    if (taskRun.status === TaskRun_Status.RUNNING) {
      await taskRunLogStore.fetchTaskRunLog(taskRun.name);
    }
  }
});

const rowKey = (taskRun: TaskRun) => {
  return taskRun.name;
};

const rowProps = (taskRun: TaskRun) => {
  return {
    style: "cursor: pointer;",
    onClick: () => {
      if (shouldShowDetailButton(taskRun)) {
        showDetail(taskRun);
      }
    },
  };
};

const columnList = computed((): DataTableColumn<TaskRun>[] => {
  return [
    {
      key: "status",
      title: "",
      width: "30px",
      render: (taskRun: TaskRun) => (
        <TaskRunStatusIcon status={taskRun.status} />
      ),
    },
    {
      key: "database",
      title: t("common.database"),
      width: "256px",
      resizable: true,
      render: (taskRun: TaskRun) => {
        const task = getTaskForTaskRun(taskRun);
        return task?.target ? (
          <DatabaseDisplay showEnvironment database={task.target} />
        ) : (
          "-"
        );
      },
    },
    {
      key: "comment",
      title: t("task.comment"),
      resizable: true,
      render: (taskRun: TaskRun) => (
        <div class="flex flex-row justify-start items-center">
          <TaskRunComment
            taskRun={taskRun}
            expandTrigger={false}
            lineClamp={1}
          />
        </div>
      ),
    },
    {
      key: "startTime",
      title: t("task.started"),
      width: "100px",
      render: (taskRun: TaskRun) => (
        <HumanizeDate date={getDateForPbTimestampProtoEs(taskRun.startTime)} />
      ),
    },
    {
      key: "actions",
      title: "",
      width: "100px",
      render: (taskRun: TaskRun) =>
        shouldShowDetailButton(taskRun) ? (
          <NButton
            size="tiny"
            onClick={() => navigateToTaskDetail(taskRun)}
            iconPlacement="right"
          >
            {{
              default: () => t("common.detail"),
              icon: () => <ExternalLinkIcon class="w-3 h-3" />,
            }}
          </NButton>
        ) : null,
    },
  ];
});

const shouldShowDetailButton = (taskRun: TaskRun) => {
  return [
    TaskRun_Status.RUNNING,
    TaskRun_Status.DONE,
    TaskRun_Status.FAILED,
    TaskRun_Status.CANCELED,
  ].includes(taskRun.status);
};

// Helper function to get route params for a task run
const getTaskRouteParams = (taskRun: TaskRun) => {
  const task = getTaskForTaskRun(taskRun);
  if (!task) return null;

  // Extract IDs from task name (format: projects/xxx/plans/yyy/rollout/stages/zzz/tasks/aaa)

  const taskParts = task.name.split("/");

  const planId = extractPlanUIDFromRolloutName(rollout.value.name);

  const stageIndex = taskParts.indexOf("stages");

  const taskIndex = taskParts.indexOf("tasks");

  if (stageIndex === -1 || taskIndex === -1) return null;

  const stageId = taskParts[stageIndex + 1];
  const taskId = taskParts[taskIndex + 1];

  return { planId, stageId, taskId };
};

// Get database for selected task run
const selectedDatabase = computed(() => {
  if (!taskRunDetailContext.taskRun) return undefined;
  const task = getTaskForTaskRun(taskRunDetailContext.taskRun);
  if (!task?.target) return undefined;
  return databaseStore.getDatabaseByName(task.target);
});

// Navigate to task detail page
const navigateToTaskDetail = (taskRun: TaskRun) => {
  const params = getTaskRouteParams(taskRun);
  if (params) {
    router.push({
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      params: {
        projectId: extractProjectResourceName(project.value.name),
        planId: params.planId,
        stageId: params.stageId || "_", // Use placeholder for empty stageId
        taskId: params.taskId,
      },
    });
  }
};

// Show task run detail in drawer
const showDetail = (taskRun: TaskRun) => {
  taskRunDetailContext.taskRun = taskRun;
  taskRunDetailContext.show = true;
};
</script>
