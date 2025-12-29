<template>
  <nav v-if="planID" class="inline-flex items-center text-sm">
    <template v-for="(item, index) in breadcrumbItems" :key="index">
      <span v-if="index > 0" class="mx-2 text-gray-400 font-mono">/</span>
      <RouterLink
        v-if="item.clickable && item.route"
        :to="item.route"
        class="text-gray-500 hover:text-accent transition-colors"
      >
        {{ item.label }}
      </RouterLink>
      <span v-else class="text-gray-900">
        {{ item.label }}
      </span>
    </template>
  </nav>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { type RouteLocationRaw, RouterLink, useRoute } from "vue-router";
import { usePlanContext } from "@/components/Plan/logic";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
} from "@/router/dashboard/projectV1";
import { useEnvironmentV1Store } from "@/store";
import {
  extractIssueUID,
  extractPlanUID,
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
} from "@/utils";

interface BreadcrumbItem {
  label: string;
  route?: RouteLocationRaw;
  clickable: boolean;
}

const { t } = useI18n();
const route = useRoute();
const { plan, issue, rollout } = usePlanContext();
const environmentStore = useEnvironmentV1Store();

const projectId = computed(() => {
  if (!plan.value?.name) return "";
  return extractProjectResourceName(plan.value.name);
});

const planUID = computed(() => {
  if (!plan.value?.name) return "";
  return extractPlanUID(plan.value.name);
});

const issueUID = computed(() => {
  if (!issue.value?.name) return "";
  return extractIssueUID(issue.value.name);
});

const planID = computed(() => {
  if (!rollout.value?.name) return "";
  return extractPlanUIDFromRolloutName(rollout.value.name);
});

// Route params for stage/task views
const stageId = computed(() => route.params.stageId as string | undefined);
const taskId = computed(() => route.params.taskId as string | undefined);

// Stage info for task detail views
const selectedStage = computed(() => {
  if (!stageId.value || !rollout.value) return undefined;
  return rollout.value.stages.find((s) => s.name.endsWith(`/${stageId.value}`));
});

const stageTitle = computed(() => {
  if (!selectedStage.value) return "";
  if (selectedStage.value.environment) {
    const env = environmentStore.getEnvironmentByName(
      selectedStage.value.environment
    );
    if (env?.title) return env.title;
  }
  return selectedStage.value.name.split("/").pop() || "";
});

const breadcrumbItems = computed<BreadcrumbItem[]>(() => {
  const items: BreadcrumbItem[] = [];
  const hasTask = !!taskId.value;

  // Issue or Plan (prefer Issue if exists, but rollout must have a plan)
  if (issueUID.value) {
    items.push({
      label: `${t("common.issue")} #${issueUID.value}`,
      route: {
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
        params: { projectId: projectId.value, issueId: issueUID.value },
      },
      clickable: true,
    });
  } else {
    items.push({
      label: `${t("common.plan")} #${planUID.value}`,
      route: {
        name: PROJECT_V1_ROUTE_PLAN_DETAIL,
        params: { projectId: projectId.value, planId: planUID.value },
      },
      clickable: true,
    });
  }

  // Rollout - clickable when viewing task detail
  items.push({
    label: `${t("common.rollout")} #${planID.value}`,
    route: {
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
      params: { projectId: projectId.value, planId: planID.value },
    },
    clickable: hasTask,
  });

  // Stage (only when viewing a task)
  if (hasTask && stageTitle.value) {
    items.push({
      label: stageTitle.value,
      route: {
        name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
        params: {
          projectId: projectId.value,
          planId: planID.value,
          stageId: stageId.value,
        },
      },
      clickable: true,
    });
  }

  // Task (always last, not clickable - current page)
  if (hasTask) {
    items.push({
      label: `${t("common.task")} #${taskId.value}`,
      clickable: false,
    });
  }

  return items;
});
</script>
