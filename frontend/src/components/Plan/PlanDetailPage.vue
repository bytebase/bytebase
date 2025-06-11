<template>
  <div class="h-full flex flex-col">
    <HeaderSection />
    <NTabs
      type="line"
      :value="tabKey"
      tab-class="first:ml-4"
      @update-value="handleTabChange"
    >
      <NTabPane
        v-for="tab in availableTabs"
        :key="tab"
        :name="tab"
        :tab="tabRender(tab)"
      >
        <Overview v-if="tab === TabKey.Overview" />
        <SpecsView v-else-if="tab === TabKey.Specifications" />
        <ChecksView v-else-if="tab === TabKey.Checks" />
      </NTabPane>

      <!-- Suffix slot for Specifications tab -->
      <template v-if="tabKey === TabKey.Specifications" #suffix>
        <div class="pr-4 flex flex-row justify-end items-center">
          <CurrentSpecSelector />
        </div>
      </template>
    </NTabs>
  </div>
</template>

<script setup lang="tsx">
import { head } from "lodash-es";
import { NTabPane, NTabs } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import {
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_CHECK_RUNS,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
} from "@/router/dashboard/projectV1";
import { ChecksView, HeaderSection, Overview } from "./components";
import CurrentSpecSelector from "./components/CurrentSpecSelector.vue";
import SpecsView from "./components/SpecsView.vue";
import { useSpecsValidation } from "./components/common";
import { gotoSpec } from "./components/common/utils";
import { usePlanContext, usePollPlan } from "./logic";

enum TabKey {
  Overview = "overview",
  Specifications = "specifications",
  Checks = "checks",
}

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { isCreating, plan, planCheckRunList } = usePlanContext();
const { isSpecEmpty } = useSpecsValidation(plan.value.specs);

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
  }
  return TabKey.Overview;
});

const availableTabs = computed<TabKey[]>(() => {
  const tabs: TabKey[] = [TabKey.Overview, TabKey.Specifications];
  if (!isCreating.value) {
    tabs.push(TabKey.Checks);
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
          {planCheckRunList.value.length > 0 && (
            <span>({planCheckRunList.value.length})</span>
          )}
        </div>
      );
    default:
      return "";
  }
};

const handleTabChange = (tab: TabKey) => {
  if (tab === TabKey.Overview) {
    router.push({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: route.params,
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
        params: route.params,
        query: route.query,
      });
    }
  } else if (tab === TabKey.Checks) {
    router.push({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL_CHECK_RUNS,
      params: route.params,
      query: route.query,
    });
  }
};

usePollPlan();
</script>
