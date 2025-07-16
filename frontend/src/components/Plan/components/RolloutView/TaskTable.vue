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
import { flatten } from "lodash-es";
import { CalendarClockIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { semanticTaskType } from "@/components/IssueV1";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1 } from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Task, Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractProjectResourceName,
  extractSchemaVersionFromTask,
  humanizeTs,
} from "@/utils";
import { useRolloutViewContext } from "./context";

const props = withDefaults(
  defineProps<{
    taskStatusFilter: Task_Status[];
    stage: Stage;
    selectedTasks?: Task[];
  }>(),
  {
    selectedTasks: () => [],
  }
);

const emit = defineEmits<{
  (event: "update:selected-tasks", tasks: Task[]): void;
  (event: "refresh"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const { project } = useCurrentProjectV1();
const { rollout, mergedStages } = useRolloutViewContext();

const taskList = computed(() => {
  let allTasks: Task[];
  if (props.stage) {
    // If a specific stage is provided, use only its tasks
    allTasks = props.stage.tasks;
  } else {
    // Otherwise, use all tasks from all stages
    allTasks = flatten(rollout.value.stages.map((stage) => stage.tasks));
  }

  if (props.taskStatusFilter.length === 0) {
    return allTasks;
  }
  return allTasks.filter((task) =>
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

const columnList = computed((): DataTableColumn<Task>[] => {
  return [
    {
      type: "selection",
      width: 50,
      disabled: (task: Task) => {
        return task.status === Task_Status.DONE;
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
      render: (task) => {
        return (
          <div class="flex items-center gap-2">
            <DatabaseDisplay database={task.target} />
            <NTag round size="small">
              {semanticTaskType(task.type)}
            </NTag>
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
    {
      key: "version",
      title: t("common.version"),
      width: 128,
      render: (task) => {
        return extractSchemaVersionFromTask(task) || "-";
      },
    },
  ];
});
</script>
