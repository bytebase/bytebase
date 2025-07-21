<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="taskList"
    :striped="true"
    :bordered="false"
    :row-key="(task: Task) => task.name"
    :virtual-scroll="virtualScrollEnabled"
    :max-height="'80vh'"
    :row-height="virtualScrollEnabled ? 40 : undefined"
    :row-props="rowProps"
    :checked-row-keys="selectedTaskNames"
    @update:checked-row-keys="handleSelectionChange"
  />
</template>

<script lang="tsx" setup>
import { head } from "lodash-es";
import { CalendarClockIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NTag, NTooltip } from "naive-ui";
import { computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { semanticTaskType } from "@/components/IssueV1";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, batchGetOrFetchDatabases } from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractProjectResourceName,
  extractSchemaVersionFromTask,
  humanizeTs,
} from "@/utils";
import { usePlanContextWithRollout } from "../../logic";
import { useRolloutViewContext } from "./context";

const props = withDefaults(
  defineProps<{
    taskStatusFilter: Task_Status[];
    tasks: Task[];
    selectedTasks?: Task[];
    taskSelectable?: (task: Task) => boolean;
  }>(),
  {
    selectedTasks: () => [],
    taskSelectable: () => () => true,
  }
);

const emit = defineEmits<{
  (event: "update:selected-tasks", tasks: Task[]): void;
}>();

const { t } = useI18n();
const router = useRouter();
const { project } = useCurrentProjectV1();
const { rollout, mergedStages } = useRolloutViewContext();
const { taskRuns } = usePlanContextWithRollout();

const taskList = computed(() => {
  if (props.taskStatusFilter.length === 0) {
    return props.tasks;
  }
  return props.tasks.filter((task) =>
    props.taskStatusFilter.includes(task.status)
  );
});

// Selection handling
const selectedTaskNames = computed(() => {
  return props.selectedTasks.map((task) => task.name);
});

const handleSelectionChange = (selectedKeys: Array<string | number>) => {
  const selectedTasks = taskList.value.filter((task) =>
    selectedKeys.includes(task.name)
  );
  emit("update:selected-tasks", selectedTasks);
};

const stages = computed(() => mergedStages.value);

// Virtual scroll configuration
const virtualScrollEnabled = computed(() => taskList.value.length > 100);

// Memoized stage lookup for better performance
const stageMap = computed(() => {
  const map = new Map<string, (typeof stages.value)[0]>();
  stages.value.forEach((stage) => {
    stage.tasks.forEach((task) => {
      map.set(task.name, stage);
    });
  });
  return map;
});

const prepareDatabases = async () => {
  if (taskList.value.length > 0) {
    try {
      await batchGetOrFetchDatabases(taskList.value.map((task) => task.target));
    } catch {
      // Ignore errors - this is just for pre-loading data
    }
  }
};

// Watch for task list changes and load database data
watch(
  taskList,
  () => {
    prepareDatabases();
  },
  { immediate: true }
);

// Helper function to extract IDs from task and stage names
const getTaskRouteParams = (task: Task) => {
  const stage = stageMap.value.get(task.name);
  if (!stage) return null;

  const rolloutId = rollout.value.name.split("/").pop();
  const stageId = stage.name.split("/").pop();
  const taskId = task.name.split("/").pop();

  return { rolloutId, stageId, taskId };
};

// Row click handler
const handleRowClick = (task: Task) => {
  const params = getTaskRouteParams(task);
  if (params) {
    router.push({
      name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
      params: {
        projectId: extractProjectResourceName(project.value.name),
        rolloutId: params.rolloutId,
        stageId: params.stageId,
        taskId: params.taskId,
      },
    });
  }
};

// Row props for click handling
const rowProps = (task: Task) => {
  return {
    style: "cursor: pointer;",
    onClick: () => handleRowClick(task),
  };
};

// Get the latest TaskRun for a given task
const getLatestTaskRun = (task: Task) => {
  const relatedTaskRuns = taskRuns.value.filter((taskRun) =>
    taskRun.name.startsWith(task.name + "/taskRuns/")
  );
  return head(relatedTaskRuns);
};

const columnList = computed((): DataTableColumn<Task>[] => {
  return [
    {
      type: "selection",
      width: 50,
      disabled: (task: Task) => {
        return !props.taskSelectable(task);
      },
      cellProps: () => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
    },
    {
      key: "status",
      title: "",
      width: "36px",
      render: (task) => <TaskStatus status={task.status} size="small" />,
    },
    {
      key: "database",
      title: t("common.database"),
      resizable: true,
      render: (task) => {
        const schemaVersion = extractSchemaVersionFromTask(task);

        return (
          <div class="flex items-center gap-2">
            <DatabaseDisplay database={task.target} />
            <NTag round size="small">
              {semanticTaskType(task.type)}
            </NTag>
            {schemaVersion && (
              <NTag round size="small">
                {schemaVersion}
              </NTag>
            )}
            {task.runTime && (
              <NTooltip>
                {{
                  trigger: () => (
                    <NTag round size="small">
                      <div class="flex items-center gap-1">
                        <CalendarClockIcon class="w-3.5 h-3.5" />
                        {humanizeTs(
                          getTimeForPbTimestampProtoEs(task.runTime, 0) / 1000
                        )}
                      </div>
                    </NTag>
                  ),
                  default: () => t("task.scheduled-time"),
                }}
              </NTooltip>
            )}
          </div>
        );
      },
    },
    {
      key: "detail",
      title: t("common.detail"),
      ellipsis: true,
      resizable: true,
      render: (task) => {
        const latestTaskRun = getLatestTaskRun(task);
        if (!latestTaskRun || !latestTaskRun.detail) {
          return "-";
        }
        return latestTaskRun.detail;
      },
    },
    {
      key: "updateTime",
      title: t("common.updated-at"),
      width: 128,
      render: (task) => {
        if (!task.updateTime) {
          return "-";
        }
        return humanizeTs(
          getTimeForPbTimestampProtoEs(task.updateTime, 0) / 1000
        );
      },
    },
  ];
});
</script>
