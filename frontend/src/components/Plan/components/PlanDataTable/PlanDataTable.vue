<template>
  <NDataTable
    key="plan-table"
    size="small"
    :columns="columnList"
    :data="planList"
    :striped="true"
    :bordered="true"
    :loading="loading"
    :row-key="(plan: Plan) => plan.name"
    :row-props="rowProps"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NPerformantEllipsis, NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import Timestamp from "@/components/misc/Timestamp.vue";
import { UserNameCell } from "@/components/v2/Model/cells";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/router/dashboard/projectV1";
import { useUserStore } from "@/store";
import { unknownUser } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { extractPlanUID, extractProjectResourceName } from "@/utils";
import PlanCheckRunStatusIcon from "../PlanCheckRunStatusIcon.vue";

withDefaults(
  defineProps<{
    planList: Plan[];
    loading?: boolean;
  }>(),
  {}
);

const { t } = useI18n();
const router = useRouter();
const userStore = useUserStore();
const { enabledNewLayout } = useIssueLayoutVersion();

const columnList = computed((): DataTableColumn<Plan>[] => {
  const columns: (DataTableColumn<Plan> & { hide?: boolean })[] = [
    {
      key: "title",
      title: t("issue.table.name"),
      ellipsis: true,
      render: (plan) => {
        const showDraftTag =
          enabledNewLayout.value && plan.issue === "" && !plan.hasRollout;
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
                  default: () => <span>{plan.title}</span>,
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
            <PlanCheckRunStatusIcon plan={plan} size="small" />
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
      key: "updateTime",
      title: t("issue.table.updated"),
      width: 128,
      render: (plan) => <Timestamp timestamp={plan.updateTime} />,
    },
    {
      key: "creator",
      title: t("issue.table.creator"),
      width: 128,
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
  return columns.filter((column) => !column.hide);
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
