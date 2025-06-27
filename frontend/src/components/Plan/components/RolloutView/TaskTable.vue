<template>
  <div class="w-full h-full flex flex-col p-4">
    <NDataTable
      size="small"
      :columns="columnList"
      :data="taskList"
      :striped="true"
      :bordered="true"
      :row-key="(task: Task) => task.name"
      :virtual-scroll="virtualScrollEnabled"
      :max-height="'80vh'"
      :row-height="virtualScrollEnabled ? 40 : undefined"
      :row-props="rowProps"
    />
  </div>
</template>

<script lang="tsx" setup>
import { flatten } from "lodash-es";
import { ChevronRightIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { semanticTaskType } from "@/components/IssueV1";
import TaskStatus from "@/components/Rollout/RolloutDetail/Panels/kits/TaskStatus.vue";
import { databaseForTask } from "@/components/Rollout/RolloutDetail/utils";
import { InstanceV1EngineIcon } from "@/components/v2";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useEnvironmentV1Store } from "@/store";
import type {
  Task,
  Rollout,
  Task_Status,
} from "@/types/proto/v1/rollout_service";
import {
  extractProjectResourceName,
  extractSchemaVersionFromTask,
} from "@/utils";
import { useRolloutViewContext } from "./context";

const props = defineProps<{
  rollout: Rollout;
  taskStatusFilter: Task_Status[];
}>();

const { t } = useI18n();
const router = useRouter();
const { project } = useCurrentProjectV1();
const environmentStore = useEnvironmentV1Store();
const { mergedStages } = useRolloutViewContext();

const taskList = computed(() => {
  const allTasks = flatten(mergedStages.value.map((stage) => stage.tasks));
  if (props.taskStatusFilter.length === 0) {
    return allTasks;
  }
  return allTasks.filter((task) =>
    props.taskStatusFilter.includes(task.status)
  );
});

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

  const rolloutId = props.rollout.name.split("/").pop();
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
      key: "status",
      width: 80,
      title: t("common.status"),
      render: (task) => {
        return <TaskStatus status={task.status} size="small" />;
      },
    },
    {
      key: "stage",
      title: t("common.stage"),
      width: 120,
      render: (task) => {
        const stage = stageMap.value.get(task.name);
        if (stage) {
          const environment = environmentStore.getEnvironmentByName(
            stage.environment
          );
          return environment.title;
        }
        return "-";
      },
    },
    {
      key: "type",
      width: 120,
      title: t("common.type"),
      render: (task) => {
        return semanticTaskType(task.type);
      },
    },
    {
      key: "database",
      title: t("common.database"),
      render: (task) => {
        const database = databaseForTask(project.value, task);
        return (
          <div class="w-auto flex flex-row items-center truncate">
            <InstanceV1EngineIcon
              class="inline-block mr-1"
              instance={database.instanceResource}
            />
            <span class="truncate">{database.instanceResource.title}</span>
            <ChevronRightIcon class="inline opacity-60 w-4 shrink-0 mx-1" />
            <span class="truncate">{database.databaseName}</span>
          </div>
        );
      },
    },
    {
      key: "version",
      title: t("common.version"),
      width: 100,
      render: (task) => {
        return extractSchemaVersionFromTask(task) || "-";
      },
    },
  ];
});
</script>
