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
import { NDataTable, NPerformantEllipsis, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { TASK_STATUS_FILTERS } from "@/components/Plan/constants/task";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import Timestamp from "@/components/misc/Timestamp.vue";
import { EnvironmentV1Name } from "@/components/v2";
import { useEnvironmentV1Store } from "@/store";
import { type ComposedRollout } from "@/types";
import type { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  stringifyTaskStatus,
  getStageStatus,
  extractRolloutUID,
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
const environmentStore = useEnvironmentV1Store();

const getStageTaskCount = (stage: any, status: Task_Status) => {
  return stage.tasks.filter((task: any) => task.status === status).length;
};

const columnList = computed(
  (): (DataTableColumn<ComposedRollout> & { hide?: boolean })[] => {
    const columns: (DataTableColumn<ComposedRollout> & { hide?: boolean })[] = [
      {
        key: "title",
        title: t("issue.table.name"),
        resizable: true,
        ellipsis: true,
        render: (rollout) => {
          return (
            <div class={`flex items-center overflow-hidden space-x-2`}>
              <div class="whitespace-nowrap text-control opacity-60">
                {extractRolloutUID(rollout.name)}
              </div>
              {rollout.title ? (
                <NPerformantEllipsis class="truncate">
                  {{
                    default: () => <span>{rollout.title}</span>,
                    tooltip: () => (
                      <div class="whitespace-pre-wrap break-words break-all">
                        {rollout.title}
                      </div>
                    ),
                  }}
                </NPerformantEllipsis>
              ) : (
                <span class="opacity-60 italic">{t("common.untitled")}</span>
              )}
            </div>
          );
        },
      },
      {
        key: "stages",
        title: t("rollout.stage.self", 2),
        resizable: true,
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
                            <EnvironmentV1Name
                              environment={environment}
                              link={false}
                            />
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
                                  <TaskStatus
                                    status={status}
                                    size="small"
                                    disabled
                                  />
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
        render: (rollout) => <Timestamp timestamp={rollout.updateTime} />,
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
