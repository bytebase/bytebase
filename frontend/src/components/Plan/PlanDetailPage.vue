<template>
  <div class="h-full flex flex-col">
    <HeaderSection />
    <NTabs
      type="line"
      :value="tabKey"
      tab-class="first:ml-4"
      @update-value="handleTabChange"
    >
      <NTab
        v-for="tab in availableTabs"
        :key="tab"
        :name="tab"
        :tab="tabRender(tab)"
      />

      <!-- Suffix slot for Specifications tab -->
      <template v-if="tabKey === TabKey.Specifications" #suffix>
        <div class="pr-4 flex flex-row justify-end items-center">
          <CurrentSpecSelector />
        </div>
      </template>
    </NTabs>

    <div class="flex-1 flex">
      <Overview v-if="tabKey === TabKey.Overview" />
      <SpecsView v-else-if="tabKey === TabKey.Specifications" />
      <ChecksView v-else-if="tabKey === TabKey.Checks" />
      <IssueReviewView v-else-if="tabKey === TabKey.Review" />
      <RolloutView v-else-if="tabKey === TabKey.Rollout" />
      <div class="pt-4 mx-auto max-w-2xl" v-else>
        <p>Unknown view for {{ tabKey }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="tsx">
import { head } from "lodash-es";
import { NTab, NTabs } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_CHECK_RUNS,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  extractIssueUID,
  extractPlanUID,
  extractRolloutUID,
} from "@/utils";
import { ChecksView, HeaderSection, Overview } from "./components";
import CurrentSpecSelector from "./components/CurrentSpecSelector.vue";
import { IssueReviewView } from "./components/IssueReviewView";
import { RolloutView } from "./components/RolloutView";
import SpecsView from "./components/SpecsView.vue";
import { useSpecsValidation } from "./components/common";
import { gotoSpec } from "./components/common/utils";
import { usePlanContext } from "./logic";

enum TabKey {
  Overview = "overview",
  Specifications = "specifications",
  Checks = "checks",
  Review = "review",
  Rollout = "rollout",
}

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { isCreating, plan } = usePlanContext();
const { isSpecEmpty } = useSpecsValidation(plan.value.specs);

const planCheckRunCount = computed(() =>
  Object.values(plan.value.planCheckRunStatusCount).reduce(
    (sum, count) => sum + count,
    0
  )
);

const tabKey = computed(() => {
  const routeName = route.name?.toString() as string;
  if (
    [
      PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
      PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    ].includes(routeName)
  ) {
    return TabKey.Specifications;
  } else if ([PROJECT_V1_ROUTE_PLAN_DETAIL_CHECK_RUNS].includes(routeName)) {
    return TabKey.Checks;
  } else if (routeName === PROJECT_V1_ROUTE_ISSUE_DETAIL_V1) {
    return TabKey.Review;
  } else if (routeName === PROJECT_V1_ROUTE_ROLLOUT_DETAIL) {
    return TabKey.Rollout;
  }
  // Fallback to Overview if no specific tab is matched.
  return TabKey.Overview;
});

const availableTabs = computed<TabKey[]>(() => {
  const tabs: TabKey[] = [TabKey.Overview, TabKey.Specifications];
  if (!isCreating.value) {
    if (
      plan.value.specs.some(
        (spec) => spec.config?.case === "changeDatabaseConfig"
      )
    ) {
      tabs.push(TabKey.Checks);
    }
    if (plan.value.issue) {
      tabs.push(TabKey.Review);
    }
    if (plan.value.rollout) {
      tabs.push(TabKey.Rollout);
    }
  }
  return tabs;
});

const tabRender = (tab: TabKey) => {
  switch (tab) {
    case TabKey.Overview:
      return t("common.overview");
    case TabKey.Specifications:
      return (
        <div>
          {t("plan.navigator.specifications")}
          {plan.value.specs.some(isSpecEmpty) && (
            <span
              class="text-error ml-0.5"
              title={t("plan.navigator.statement-empty")}
            >
              *
            </span>
          )}
        </div>
      );
    case TabKey.Checks:
      return (
        <div>
          {t("plan.navigator.checks")}
          {planCheckRunCount.value > 0 && (
            <span class="text-gray-500">({planCheckRunCount.value})</span>
          )}
        </div>
      );
    case TabKey.Review:
      return t("plan.navigator.review");
    case TabKey.Rollout:
      return t("plan.navigator.rollout");
    default:
      // Fallback to raw tab name.
      return tab;
  }
};

const handleTabChange = (tab: TabKey) => {
  const params = route.params;
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

  if (tab === TabKey.Overview) {
    router.push({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: params,
      query: route.query,
    });
  } else if (tab === TabKey.Specifications) {
    // Auto select the first spec when switching to Specifications tab.
    const spec = head(plan.value.specs);
    if (spec) {
      gotoSpec(router, spec.id);
    } else {
      router.push({
        name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
        params: params,
        query: route.query,
      });
    }
  } else if (tab === TabKey.Checks) {
    router.push({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL_CHECK_RUNS,
      params: params,
      query: route.query,
    });
  } else if (tab === TabKey.Review) {
    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
      params: params,
      query: route.query,
    });
  } else if (tab === TabKey.Rollout) {
    router.push({
      name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
      params: params,
      query: route.query,
    });
  }
};
</script>
