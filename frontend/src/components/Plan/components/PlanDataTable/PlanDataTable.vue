<template>
  <NDataTable
    :columns="columnList"
    :data="planList"
    :striped="true"
    :bordered="true"
    :loading="loading"
    :row-key="(plan: ComposedPlan) => plan.name"
    :row-props="rowProps"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NPerformantEllipsis, NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAvatar } from "@/bbkit";
import { ProjectNameCell } from "@/components/v2/Model/DatabaseV1Table/cells";
import { PROJECT_V1_ROUTE_REVIEW_CENTER_DETAIL } from "@/router/dashboard/projectV1";
import { getTimeForPbTimestamp } from "@/types";
import type { ComposedPlan } from "@/types/v1/issue/plan";
import {
  extractPlanUID,
  extractProjectResourceName,
  humanizeTs,
  planV1Slug,
} from "@/utils";
import PlanCheckRunStatusIcon from "../PlanCheckRunStatusIcon.vue";

const props = withDefaults(
  defineProps<{
    planList: ComposedPlan[];
    loading?: boolean;
    showProject: boolean;
  }>(),
  {
    loading: true,
    showProject: true,
  }
);

const router = useRouter();
const { t } = useI18n();

const columnList = computed((): DataTableColumn<ComposedPlan>[] => {
  const columns: (DataTableColumn<ComposedPlan> & { hide?: boolean })[] = [
    {
      key: "status",
      title: "",
      width: "36px",
      render: (plan) => {
        return <PlanCheckRunStatusIcon plan={plan} />;
      },
    },
    {
      key: "title",
      title: t("issue.table.name"),
      resizable: true,
      render: (plan) => {
        return (
          <div class="flex items-center overflow-hidden space-x-2">
            <div class="whitespace-nowrap text-control opacity-60">
              {extractPlanUID(plan.name)}
            </div>
            <NPerformantEllipsis class="flex-1 truncate">
              {{
                default: () => <span>{plan.title}</span>,
                tooltip: () => (
                  <div class="whitespace-pre-wrap break-words break-all">
                    {plan.title}
                  </div>
                ),
              }}
            </NPerformantEllipsis>
          </div>
        );
      },
    },
    {
      key: "project",
      title: t("common.project"),
      width: 144,
      resizable: true,
      hide: !props.showProject,
      render: (plan) => (
        <ProjectNameCell project={plan.projectEntity} mode={"ALL_SHORT"} />
      ),
    },
    {
      key: "updateTime",
      title: t("issue.table.updated"),
      width: 150,
      render: (plan) =>
        humanizeTs(getTimeForPbTimestamp(plan.updateTime, 0) / 1000),
    },
    {
      key: "creator",
      width: 150,
      title: t("issue.table.creator"),
      render: (plan) => (
        <div class="flex flex-row items-center overflow-hidden gap-x-2">
          <BBAvatar size="SMALL" username={plan.creatorEntity.title} />
          <span class="truncate">{plan.creatorEntity.title}</span>
        </div>
      ),
    },
  ];
  return columns.filter((column) => !column.hide);
});

const rowProps = (plan: ComposedPlan) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      const route = router.resolve({
        name: PROJECT_V1_ROUTE_REVIEW_CENTER_DETAIL,
        params: {
          projectId: extractProjectResourceName(plan.project),
          planSlug: planV1Slug(plan),
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
