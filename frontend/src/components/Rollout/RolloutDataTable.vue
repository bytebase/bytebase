<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="sortedRolloutList"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(rollout: ComposedRollout) => rollout.name"
    :row-props="rowProps"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, useRouter } from "vue-router";
import { BBAvatar } from "@/bbkit";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/router/dashboard/projectV1";
import { useEnvironmentV1Store, useUserStore } from "@/store";
import {
  getTimeForPbTimestampProtoEs,
  unknownUser,
  type ComposedRollout,
} from "@/types";
import { Task_Status as TaskStatusEnum } from "@/types/proto-es/v1/rollout_service_pb";
import type { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractPlanUID,
  extractProjectResourceName,
  humanizeTs,
  stringifyTaskStatus,
  getStageStatus,
} from "@/utils";

const props = withDefaults(
  defineProps<{
    rolloutList: ComposedRollout[];
    bordered?: boolean;
    loading?: boolean;
    showSelection?: boolean;
  }>(),
  {
    loading: true,
    bordered: false,
    showSelection: false,
  }
);

const { t } = useI18n();
const router = useRouter();
const userStore = useUserStore();
const environmentStore = useEnvironmentV1Store();

const TASK_STATUS_FILTERS: Task_Status[] = [
  TaskStatusEnum.DONE,
  TaskStatusEnum.RUNNING,
  TaskStatusEnum.FAILED,
  TaskStatusEnum.CANCELED,
  TaskStatusEnum.SKIPPED,
  TaskStatusEnum.PENDING,
  TaskStatusEnum.NOT_STARTED,
];

const getTaskCount = (rollout: ComposedRollout, status: Task_Status) => {
  const allTasks = rollout.stages.flatMap((stage) => stage.tasks);
  return allTasks.filter((task) => task.status === status).length;
};

const columnList = computed(
  (): (DataTableColumn<ComposedRollout> & { hide?: boolean })[] => {
    const columns: (DataTableColumn<ComposedRollout> & { hide?: boolean })[] = [
      {
        key: "plan",
        title: t("plan.self"),
        width: 96,
        render: (rollout) => {
          const uid = extractPlanUID(rollout.plan);
          return (
            <RouterLink
              to={{
                name: PROJECT_V1_ROUTE_PLAN_DETAIL,
                params: {
                  projectId: extractProjectResourceName(rollout.plan),
                  planId: uid,
                },
              }}
              custom={true}
            >
              {{
                default: ({ href }: { href: string }) => (
                  <a
                    href={href}
                    class="normal-link"
                    onClick={(e: MouseEvent) => e.stopPropagation()}
                  >
                    #{uid}
                  </a>
                ),
              }}
            </RouterLink>
          );
        },
      },
      {
        key: "stages",
        title: t("rollout.stage.self", 2),
        render: (rollout) => {
          if (rollout.stages.length === 0) {
            return (
              <span class="text-sm text-gray-400 italic">
                {t("common.no-data")}
              </span>
            );
          }
          return (
            <div class="flex items-center gap-2">
              {rollout.stages.map((stage, index) => {
                const environment = environmentStore.getEnvironmentByName(
                  stage.environment
                );
                const stageStatus = getStageStatus(stage);
                return (
                  <>
                    <div key={stage.name} class="flex items-center gap-1">
                      <TaskStatus status={stageStatus} size="small" />
                      <span class="text-sm font-medium text-gray-700">
                        {environment.title}
                      </span>
                    </div>
                    {index < rollout.stages.length - 1 && (
                      <span class="text-gray-400">→</span>
                    )}
                  </>
                );
              })}
            </div>
          );
        },
      },
      {
        key: "tasks",
        title: t("common.tasks"),
        render: (rollout) => {
          return (
            <div class="flex flex-row gap-1 items-center">
              {TASK_STATUS_FILTERS.map((status) => {
                const count = getTaskCount(rollout, status);
                if (count === 0) return null;

                return (
                  <NTag key={status} round>
                    {{
                      avatar: () => <TaskStatus status={status} size="small" />,
                      default: () => (
                        <div class="flex flex-row items-center gap-1">
                          <span class="select-none text-sm">
                            {stringifyTaskStatus(status)}
                          </span>
                          <span class="select-none text-sm font-medium">
                            {count}
                          </span>
                        </div>
                      ),
                    }}
                  </NTag>
                );
              })}
            </div>
          );
        },
      },
      {
        key: "createTime",
        title: t("common.created-at"),
        width: 128,
        render: (rollout) =>
          humanizeTs(
            getTimeForPbTimestampProtoEs(rollout.createTime, 0) / 1000
          ),
      },
      {
        key: "creator",
        title: t("common.creator"),
        width: 128,
        render: (rollout) => {
          const creator =
            userStore.getUserByIdentifier(rollout.creator) || unknownUser();
          return (
            <div class="flex flex-row items-center overflow-hidden gap-x-2">
              <BBAvatar size="SMALL" username={creator.title} />
              <span class="truncate">{creator.title}</span>
            </div>
          );
        },
      },
    ];
    return columns.filter((column) => !column.hide);
  }
);

const sortedRolloutList = computed(() => {
  return props.rolloutList;
});

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
