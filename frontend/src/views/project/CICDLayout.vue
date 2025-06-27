<template>
  <div class="relative overflow-x-hidden h-full">
    <template v-if="ready">
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
            @click="handleTabChange(tab)"
          />

          <!-- Suffix slot for Specifications tab -->
          <template v-if="tabKey === TabKey.Specifications" #suffix>
            <div class="pr-4 flex flex-row justify-end items-center">
              <CurrentSpecSelector />
            </div>
          </template>
        </NTabs>

        <div class="flex-1 flex">
          <router-view />
        </div>
      </div>
    </template>
    <div v-else class="w-full h-full flex flex-col items-center justify-center">
      <NSpin />
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { useTitle } from "@vueuse/core";
import { head } from "lodash-es";
import { NSpin, NTab, NTabs } from "naive-ui";
import { computed, toRef } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import {
  providePlanContext,
  useBasePlanContext,
  useInitializePlan,
} from "@/components/Plan";
import { HeaderSection } from "@/components/Plan/components";
import CurrentSpecSelector from "@/components/Plan/components/CurrentSpecSelector.vue";
import { useSpecsValidation } from "@/components/Plan/components/common";
import { gotoSpec } from "@/components/Plan/components/common/utils";
import { provideIssueReviewContext } from "@/components/Plan/logic/issue-review";
import { useBodyLayoutContext } from "@/layouts/common";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_CHECK_RUNS,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  extractIssueUID,
  extractPlanUID,
  extractRolloutUID,
  isNullOrUndefined,
} from "@/utils";

enum TabKey {
  Overview = "overview",
  Specifications = "specifications",
  Checks = "checks",
  Review = "review",
  Rollout = "rollout",
}

const props = defineProps<{
  projectId: string;
  planId?: string;
  issueId?: string;
  rolloutId?: string;
}>();

const { t } = useI18n();
const { isCreating, plan, planCheckRunList, issue, rollout, isInitializing } =
  useInitializePlan(
    toRef(props, "projectId"),
    toRef(props, "planId"),
    toRef(props, "issueId"),
    toRef(props, "rolloutId")
  );
const planBaseContext = useBasePlanContext({
  isCreating,
  plan,
});

const ready = computed(() => {
  return !isInitializing.value && !!plan.value;
});

const route = useRoute();
const router = useRouter();

providePlanContext(
  {
    isCreating,
    plan,
    planCheckRunList,
    issue,
    rollout,
    ...planBaseContext,
  },
  true /* root */
);

provideIssueReviewContext(issue);

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
  } else if (
    [
      PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
      PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
    ].includes(routeName)
  ) {
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
        (spec) => !isNullOrUndefined(spec.changeDatabaseConfig)
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

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("!py-0 !px-0");

const documentTitle = computed(() => {
  if (isCreating.value) {
    return t("plan.new-plan");
  } else {
    if (ready.value) {
      if (issue.value) {
        return issue.value.title;
      }
      if (plan.value) {
        return plan.value.title;
      }
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>
