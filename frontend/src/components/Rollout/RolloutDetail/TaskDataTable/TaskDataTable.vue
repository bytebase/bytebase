<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="taskList"
    :striped="true"
    :bordered="true"
    :row-key="(task: Task) => task.name"
    :row-props="rowProps"
  />
</template>

<script lang="tsx" setup>
import { ChevronRightIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { semanticTaskType } from "@/components/IssueV1";
import { EnvironmentV1Name, InstanceV1EngineIcon } from "@/components/v2";
import { type ComposedRollout } from "@/types";
import type { Task } from "@/types/proto/v1/rollout_service";
import { extractSchemaVersionFromTask } from "@/utils";
import TaskStatus from "../Panels/kits/TaskStatus.vue";
import { databaseForTask } from "../utils";
import { extractEnvironmentResourceName } from "@/utils";

const props = withDefaults(
  defineProps<{
    rollout: ComposedRollout;
    taskList: Task[];
  }>(),
  {}
);

const { t } = useI18n();
const router = useRouter();

const stages = computed(() => props.rollout.stages);

const columnList = computed(
  (): (DataTableColumn<Task> & { hide?: boolean })[] => {
    const columns: (DataTableColumn<Task> & { hide?: boolean })[] = [
      {
        key: "status",
        width: 64,
        title: t("common.status"),
        render: (task) => {
          return <TaskStatus status={task.status} size="small" />;
        },
      },
      {
        key: "stage",
        title: t("common.stage"),
        width: 96,
        render: (task) => {
          const stage = stages.value.find(stage => stage.tasks.find((t) => t.name === task.name));
          if (stage) {
            return extractEnvironmentResourceName(stage.environment);
          }
          return "-";
        },
      },
      {
        key: "type",
        width: 96,
        title: t("common.type"),
        render: (task) => {
          return semanticTaskType(task.type);
        },
      },
      {
        key: "database",
        title: t("common.database"),
        render: (task) => {
          return (
            <div class="w-auto flex flex-row items-center truncate">
              <InstanceV1EngineIcon
                class="inline-block mr-1"
                instance={
                  databaseForTask(props.rollout.projectEntity, task)
                    .instanceResource
                }
              />
              <span class="truncate">
                {
                  databaseForTask(props.rollout.projectEntity, task)
                    .instanceResource.title
                }
              </span>
              <ChevronRightIcon class="inline opacity-60 w-4 shrink-0" />
              <EnvironmentV1Name
                class="text-gray-400 mr-1"
                environment={
                  databaseForTask(props.rollout.projectEntity, task)
                    .effectiveEnvironmentEntity
                }
                link={false}
                tag="span"
              />
              <span class="truncate">
                {
                  databaseForTask(props.rollout.projectEntity, task)
                    .databaseName
                }
              </span>
            </div>
          );
        },
      },
      {
        key: "version",
        title: t("common.version"),
        render: (task) => {
          return extractSchemaVersionFromTask(task);
        },
      },
    ];
    return columns.filter((column) => !column.hide);
  }
);

const rowProps = (rollout: ComposedRollout) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      const url = `/${rollout.name}`;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
  };
};
</script>
