import { CirclePlayIcon, FileDiffIcon, Layers2Icon } from "lucide-vue-next";
import { NTag } from "naive-ui";
import type { Ref } from "vue";
import { computed } from "vue";
import type { ComposerTranslation } from "vue-i18n";
import type { RouteLocationNormalizedLoaded, Router } from "vue-router";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
} from "@/router/dashboard/projectV1";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { EMPTY_PLAN_NAME } from "@/types/v1/issue/plan";
import { extractIssueUID, extractPlanUID, extractRolloutUID } from "@/utils";

export enum TabKey {
  Plan = "plan",
  Issue = "issue",
  Rollout = "rollout",
}

export interface UseCICDTabNavigationOptions {
  route: RouteLocationNormalizedLoaded;
  router: Router;
  plan: Ref<Plan>;
  issue: Ref<Issue | undefined>;
  isCreating: Ref<boolean>;
  enabledNewLayout: Ref<boolean>;
  t: ComposerTranslation;
}

export const useCICDTabNavigation = (options: UseCICDTabNavigationOptions) => {
  const { route, router, plan, issue, isCreating, enabledNewLayout, t } =
    options;

  const tabKey = computed(() => {
    const routeName = route.name?.toString();
    if (!routeName) {
      // Default to Issue if no valid plan, otherwise Plan
      return plan.value.name === EMPTY_PLAN_NAME ? TabKey.Issue : TabKey.Plan;
    }

    if (
      [
        PROJECT_V1_ROUTE_PLAN_DETAIL,
        PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
        PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      ].includes(routeName)
    ) {
      return TabKey.Plan;
    } else if (routeName === PROJECT_V1_ROUTE_ISSUE_DETAIL_V1) {
      return TabKey.Issue;
    } else if (
      [
        PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
        PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
        PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
      ].includes(routeName)
    ) {
      return TabKey.Rollout;
    }
    // Fallback: Default to Issue if no valid plan, otherwise Plan
    return plan.value.name === EMPTY_PLAN_NAME ? TabKey.Issue : TabKey.Plan;
  });

  const availableTabs = computed<TabKey[]>(() => {
    const tabs: TabKey[] = [];

    // Show Issue tab if issue exists and new layout is enabled
    if (plan.value.issue && enabledNewLayout.value) {
      tabs.push(TabKey.Issue);
    }

    // Only show Plan tab if we have a valid plan with specs
    // Don't show Plan tab for grant requests (they don't have plan specs)
    const isGrantRequest = issue.value?.type === Issue_Type.GRANT_REQUEST;
    const hasValidPlan =
      plan.value.name !== EMPTY_PLAN_NAME && plan.value.specs.length > 0;
    if (hasValidPlan && !isGrantRequest) {
      tabs.push(TabKey.Plan);
    }

    // Show Rollout tab if rollout exists
    if (plan.value.rollout) {
      tabs.push(TabKey.Rollout);
    }

    return tabs;
  });

  const issueTabContent = computed(() => (
    <div class="flex items-center gap-2">
      <Layers2Icon size={18} />
      <span>{t("common.overview")}</span>
    </div>
  ));

  const planTabContent = computed(() => (
    <div class="flex items-center gap-2">
      <FileDiffIcon size={18} />
      <span>{t("plan.navigator.changes")}</span>
      {(isCreating.value || plan.value.specs.length > 1) && (
        <NTag size="tiny" round>
          {plan.value.specs.length}
        </NTag>
      )}
    </div>
  ));

  const rolloutTabContent = computed(() => (
    <div class="flex items-center gap-2">
      <CirclePlayIcon size={18} />
      <span>{t("plan.navigator.rollout")}</span>
    </div>
  ));

  const tabRender = (tab: TabKey) => {
    switch (tab) {
      case TabKey.Issue:
        return issueTabContent.value;
      case TabKey.Plan:
        return planTabContent.value;
      case TabKey.Rollout:
        return rolloutTabContent.value;
      default:
        return tab;
    }
  };

  const handleTabChange = (tab: TabKey) => {
    if (!route?.params) {
      console.error("Cannot navigate: route params are undefined");
      return;
    }

    const params = { ...route.params };
    if (isCreating.value) {
      params.planId = "create";
    } else {
      params.planId = extractPlanUID(plan.value.name);
      if (plan.value.issue) {
        params.issueId = extractIssueUID(plan.value.issue);
      }
      if (plan.value.rollout) {
        params.rolloutId = extractRolloutUID(plan.value.rollout);
      }
    }

    const query = route.query || {};

    if (tab === TabKey.Issue) {
      router
        .push({
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
          params: params,
          query: query,
        })
        .catch((error) => {
          console.error("Failed to navigate to Issue tab:", error);
        });
    } else if (tab === TabKey.Plan) {
      router
        .push({
          name: PROJECT_V1_ROUTE_PLAN_DETAIL,
          params: params,
          query: query,
        })
        .catch((error) => {
          console.error("Failed to navigate to Plan tab:", error);
        });
    } else if (tab === TabKey.Rollout) {
      router
        .push({
          name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
          params: params,
          query: query,
        })
        .catch((error) => {
          console.error("Failed to navigate to Rollout tab:", error);
        });
    }
  };

  return {
    tabKey,
    availableTabs,
    tabRender,
    handleTabChange,
  };
};
