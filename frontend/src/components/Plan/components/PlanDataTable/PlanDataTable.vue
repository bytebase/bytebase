<template>
  <div>
    <NDataTable
      key="plan-table"
      size="small"
      :columns="columnList"
      :data="planList"
      :striped="true"
      :bordered="false"
      :loading="loading"
      :scroll-x="scrollX"
      :row-key="(plan: Plan) => plan.name"
      :row-props="rowProps"
    />
  </div>
</template>

<script lang="tsx" setup>
import type { DataTableColumn, TagProps } from "naive-ui";
import { NDataTable, NPerformantEllipsis, NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import Timestamp from "@/components/misc/Timestamp.vue";
import TaskStatus from "@/components/RolloutV1/components/Task/TaskStatus.vue";
import { TASK_STATUS_FILTERS } from "@/components/RolloutV1/constants/task";
import { EnvironmentV1Name } from "@/components/v2";
import { UserNameCell } from "@/components/v2/Model/cells";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/router/dashboard/projectV1";
import { useEnvironmentV1Store, useUserStore } from "@/store";
import { formatEnvironmentName, unknownUser } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { Issue_ApprovalStatus } from "@/types/proto-es/v1/issue_service_pb";
import type {
  Plan,
  Plan_RolloutStageSummary,
} from "@/types/proto-es/v1/plan_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { extractPlanUID, extractProjectResourceName } from "@/utils";
import { extractStageUID } from "@/utils/v1/issue/rollout";
import PlanCheckStatusCount from "../PlanCheckStatusCount.vue";

withDefaults(
  defineProps<{
    planList: Plan[];
    loading?: boolean;
  }>(),
  {}
);

const { t } = useI18n();
const router = useRouter();
const environmentStore = useEnvironmentV1Store();
const userStore = useUserStore();

const TITLE_MIN_WIDTH = 320;
const CHECKS_COLUMN_WIDTH = 200;
const REVIEW_COLUMN_WIDTH = 140;
const STAGES_COLUMN_WIDTH = 260;
const UPDATED_COLUMN_WIDTH = 150;
const CREATOR_COLUMN_WIDTH = 150;

type StatusTag = {
  label: string;
  type?: TagProps["type"];
};

const getApprovalStatusTag = (plan: Plan): StatusTag | undefined => {
  if (plan.issue === "") {
    return undefined;
  }

  switch (plan.approvalStatus) {
    case Issue_ApprovalStatus.CHECKING:
      return {
        label: t("task.checking"),
      };
    case Issue_ApprovalStatus.APPROVED:
      return {
        label: t("issue.table.approved"),
        type: "success",
      };
    case Issue_ApprovalStatus.SKIPPED:
      return {
        label: t("common.skipped"),
      };
    case Issue_ApprovalStatus.REJECTED:
      return {
        label: t("common.rejected"),
        type: "warning",
      };
    case Issue_ApprovalStatus.PENDING:
      return {
        label: t("common.under-review"),
        type: "info",
      };
    default:
      return undefined;
  }
};

const getRolloutStageStatus = (
  summary: Plan_RolloutStageSummary
): Task_Status => {
  for (const status of TASK_STATUS_FILTERS) {
    if (summary.taskStatusCounts.some((item) => item.status === status)) {
      return status;
    }
  }
  return Task_Status.STATUS_UNSPECIFIED;
};

const renderRolloutStages = (plan: Plan) => {
  if (plan.rolloutStageSummaries.length === 0) {
    return <span class="text-control-light">-</span>;
  }

  return (
    <div class="flex items-center gap-1 flex-wrap">
      {plan.rolloutStageSummaries.map((summary, index) => {
        const environment = environmentStore.getEnvironmentByName(
          formatEnvironmentName(extractStageUID(summary.stage))
        );
        return (
          <div key={summary.stage} class="flex items-center gap-1">
            <div class="flex items-center gap-1">
              <TaskStatus
                status={getRolloutStageStatus(summary)}
                size="small"
                disabled
              />
              <EnvironmentV1Name
                environment={environment}
                link={false}
                showIcon={false}
                showColor={false}
                textClass="text-sm"
              />
            </div>
            {index < plan.rolloutStageSummaries.length - 1 && (
              <span class="mx-1 text-control-light">→</span>
            )}
          </div>
        );
      })}
    </div>
  );
};

const columnList = computed((): DataTableColumn<Plan>[] => {
  return [
    {
      key: "title",
      title: t("issue.table.name"),
      minWidth: TITLE_MIN_WIDTH,
      ellipsis: true,
      render: (plan) => {
        const showDraftTag = plan.issue === "" && !plan.hasRollout;
        const isDeleted = plan.state === State.DELETED;
        return (
          <div
            class={`flex items-center overflow-hidden gap-x-2 ${isDeleted ? "opacity-60" : ""}`}
          >
            <div class="whitespace-nowrap text-control opacity-60">
              {extractPlanUID(plan.name)}
            </div>
            {plan.title ? (
              <NPerformantEllipsis class="truncate">
                {{
                  default: () => <span class="normal-nums">{plan.title}</span>,
                  tooltip: () => (
                    <div class="whitespace-pre-wrap wrap-break-word break-all">
                      {plan.title}
                    </div>
                  ),
                }}
              </NPerformantEllipsis>
            ) : (
              <span class="opacity-60 italic">{t("common.untitled")}</span>
            )}
            {isDeleted && (
              <NTag type="warning" round size="small">
                {t("common.closed")}
              </NTag>
            )}
            {showDraftTag && !isDeleted && (
              <NTag round size="small">
                {t("common.draft")}
              </NTag>
            )}
          </div>
        );
      },
    },
    {
      key: "checks",
      title: t("plan.checks.self"),
      width: CHECKS_COLUMN_WIDTH,
      render: (plan) => <PlanCheckStatusCount plan={plan} />,
    },
    {
      key: "approval",
      title: t("plan.navigator.review"),
      width: REVIEW_COLUMN_WIDTH,
      render: (plan) => {
        const statusTag = getApprovalStatusTag(plan);
        if (!statusTag) {
          return <span class="text-control-light">-</span>;
        }
        return (
          <NTag size="small" round type={statusTag.type}>
            {statusTag.label}
          </NTag>
        );
      },
    },
    {
      key: "stages",
      title: t("rollout.stage.self", 2),
      width: STAGES_COLUMN_WIDTH,
      render: (plan) => renderRolloutStages(plan),
    },
    {
      key: "updateTime",
      title: t("issue.table.updated"),
      width: UPDATED_COLUMN_WIDTH,
      render: (plan) => <Timestamp timestamp={plan.updateTime} />,
    },
    {
      key: "creator",
      title: t("issue.table.creator"),
      width: CREATOR_COLUMN_WIDTH,
      render: (plan) => {
        const creator =
          userStore.getUserByIdentifier(plan.creator) ||
          unknownUser(plan.creator);
        return (
          <UserNameCell
            user={creator}
            size="small"
            allowEdit={false}
            showMfaEnabled={false}
            showSource={false}
            showEmail={false}
          />
        );
      },
    },
  ];
});

const scrollX = computed(() => {
  return columnList.value.reduce((sum, column) => {
    return (
      sum +
      ((column as { width?: number; minWidth?: number }).width ??
        (column as { minWidth?: number }).minWidth ??
        100)
    );
  }, 0);
});

const rowProps = (plan: Plan) => {
  const isDeleted = plan.state === State.DELETED;
  return {
    style: isDeleted ? "cursor: pointer; opacity: 0.7;" : "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      const route = router.resolve({
        name: PROJECT_V1_ROUTE_PLAN_DETAIL,
        params: {
          projectId: extractProjectResourceName(plan.name),
          planId: extractPlanUID(plan.name),
        },
      });
      const url = route.fullPath;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
  };
};
</script>
