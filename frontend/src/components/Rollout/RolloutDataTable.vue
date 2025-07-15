<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="rolloutList"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(rollout: ComposedRollout) => rollout.name"
    :row-props="rowProps"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NTooltip } from "naive-ui";
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

withDefaults(
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

const getStageTaskCount = (stage: any, status: Task_Status) => {
  return stage.tasks.filter((task: any) => task.status === status).length;
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
                    <NTooltip key={stage.name} placement="top">
                      {{
                        trigger: () => (
                          <div class="flex items-center gap-1 cursor-pointer">
                            <TaskStatus
                              status={stageStatus}
                              size="small"
                              disabled
                            />
                            <span class="text-sm font-medium text-gray-700 whitespace-nowrap">
                              {environment.title}
                            </span>
                          </div>
                        ),
                        default: () => {
                          const taskCounts = TASK_STATUS_FILTERS.map(
                            (status) => {
                              const count = getStageTaskCount(stage, status);
                              return { status, count };
                            }
                          ).filter(({ count }) => count > 0);

                          if (taskCounts.length === 0) {
                            return (
                              <span class="text-sm text-gray-400">
                                {t("common.no-data")}
                              </span>
                            );
                          }

                          return (
                            <div class="flex flex-col gap-1">
                              {taskCounts.map(({ status, count }) => (
                                <div
                                  key={status}
                                  class="flex items-center gap-2"
                                >
                                  <TaskStatus status={status} size="small" />
                                  <span class="text-sm">
                                    {stringifyTaskStatus(status)}: {count}
                                  </span>
                                </div>
                              ))}
                            </div>
                          );
                        },
                      }}
                    </NTooltip>
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
        key: "updateTime",
        title: t("common.updated-at"),
        width: 128,
        render: (rollout) =>
          humanizeTs(
            getTimeForPbTimestampProtoEs(rollout.updateTime, 0) / 1000
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
