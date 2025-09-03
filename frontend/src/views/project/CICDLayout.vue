<template>
  <div ref="containerRef" class="relative overflow-x-hidden h-full">
    <template v-if="ready">
      <PollerProvider>
        <div class="h-full flex flex-col">
          <!-- Banner Section -->
          <div v-if="showBanner" class="banner-section">
            <div
              v-if="showClosedBanner"
              class="h-8 w-full text-base font-medium bg-gray-400 text-white flex justify-center items-center"
            >
              {{ $t("common.closed") }}
            </div>
            <div
              v-else-if="showSuccessBanner"
              class="h-8 w-full text-base font-medium bg-success text-white flex justify-center items-center"
            >
              {{ $t("common.done") }}
            </div>
          </div>

          <HeaderSection />

          <NTabs
            v-if="shouldShowNavigation"
            type="line"
            :value="tabKey"
            tab-class="first:ml-4"
            @update-value="handleTabChange"
          >
            <NTab
              v-for="tab in availableTabs"
              class="select-none"
              :key="tab"
              :name="tab"
              :tab="tabRender(tab)"
              @click="handleTabChange(tab)"
            />

            <!-- Suffix slot -->
            <template #suffix>
              <div class="pr-3 flex flex-row justify-end items-center gap-4">
                <RefreshIndicator />
              </div>
            </template>
          </NTabs>

          <div class="flex-1 flex">
            <router-view />
          </div>
        </div>
      </PollerProvider>
    </template>
    <div v-else class="w-full h-full flex flex-col items-center justify-center">
      <NSpin />
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { useTitle } from "@vueuse/core";
import { CirclePlayIcon, FileDiffIcon, Layers2Icon } from "lucide-vue-next";
import { NSpin, NTab, NTabs, NTag } from "naive-ui";
import { computed, ref, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  useRoute,
  useRouter,
  onBeforeRouteLeave,
  onBeforeRouteUpdate,
} from "vue-router";
import {
  providePlanContext,
  useBasePlanContext,
  useInitializePlan,
} from "@/components/Plan";
import PollerProvider from "@/components/Plan/PollerProvider.vue";
import { HeaderSection } from "@/components/Plan/components";
import RefreshIndicator from "@/components/Plan/components/RefreshIndicator.vue";
import { provideIssueReviewContext } from "@/components/Plan/logic/issue-review";
import { provideSidebarContext } from "@/components/Plan/logic/sidebar";
import { useNavigationGuard } from "@/components/Plan/logic/useNavigationGuard";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { useBodyLayoutContext } from "@/layouts/common";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
} from "@/router/dashboard/projectV1";
import { State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  extractIssueUID,
  extractPlanUID,
  extractProjectResourceName,
  extractRolloutUID,
  issueV1Slug,
} from "@/utils";

enum TabKey {
  Plan = "plan",
  Issue = "issue",
  Rollout = "rollout",
}

const props = defineProps<{
  projectId: string;
  planId?: string;
  issueId?: string;
  rolloutId?: string;
}>();

const { t } = useI18n();
const {
  isCreating,
  plan,
  planCheckRuns,
  issue,
  rollout,
  taskRuns,
  isInitializing,
} = useInitializePlan(
  toRef(props, "projectId"),
  toRef(props, "planId"),
  toRef(props, "issueId"),
  toRef(props, "rolloutId")
);
const planBaseContext = useBasePlanContext({
  isCreating,
  plan,
  issue,
});
const { enabledNewLayout } = useIssueLayoutVersion();
const isLoading = ref(true);
const containerRef = ref<HTMLElement>();

const ready = computed(() => {
  return !isInitializing.value && !!plan.value && !isLoading.value;
});

const shouldShowNavigation = computed(() => {
  return (
    !isCreating.value &&
    plan.value.specs.some(
      (spec) => spec.config.case === "changeDatabaseConfig"
    ) &&
    (plan.value.issue || plan.value.rollout)
  );
});

const route = useRoute();
const router = useRouter();
const { confirmNavigation } = useNavigationGuard();

providePlanContext({
  isCreating,
  plan,
  planCheckRuns,
  issue,
  rollout,
  taskRuns,
  ...planBaseContext,
});

provideIssueReviewContext(computed(() => issue.value));

provideSidebarContext(containerRef);

watch(
  () => isInitializing.value,
  () => {
    if (isInitializing.value) {
      return;
    }

    // Redirect all non-changeDatabaseConfig plans to the legacy issue page.
    // Including export data plans.
    if (
      plan.value.issue &&
      plan.value.specs.some(
        (spec) =>
          spec.config.case !== "changeDatabaseConfig" &&
          spec.config.case !== "createDatabaseConfig"
      )
    ) {
      router.replace({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(plan.value.name),
          issueSlug: issueV1Slug(plan.value.issue),
        },
        query: route.query,
      });
    } else {
      isLoading.value = false;
    }
  },
  { once: true }
);

const tabKey = computed(() => {
  const routeName = route.name?.toString() as string;
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
  // Fallback to Overview if no specific tab is matched.
  return TabKey.Plan;
});

const availableTabs = computed<TabKey[]>(() => {
  const tabs: TabKey[] = [TabKey.Plan];
  if (plan.value.issue && enabledNewLayout.value) {
    tabs.unshift(TabKey.Issue);
  }
  if (plan.value.rollout) {
    tabs.push(TabKey.Rollout);
  }
  return tabs;
});

const tabRender = (tab: TabKey) => {
  switch (tab) {
    case TabKey.Issue:
      return (
        <div class="flex items-center gap-2">
          <Layers2Icon size={18} />
          <span>{t("common.overview")}</span>
        </div>
      );
    case TabKey.Plan:
      return (
        <div class="flex items-center gap-2">
          <FileDiffIcon size={18} />
          <span>{t("plan.navigator.changes")}</span>
          {(isCreating.value || plan.value.specs.length > 1) && (
            <NTag size="tiny" round>
              {plan.value.specs.length}
            </NTag>
          )}
        </div>
      );
    case TabKey.Rollout:
      return (
        <div class="flex items-center gap-2">
          <CirclePlayIcon size={18} />
          <span>{t("plan.navigator.rollout")}</span>
        </div>
      );
    default:
      // Fallback to raw tab name.
      return tab;
  }
};

const handleTabChange = (tab: TabKey) => {
  if (!route || !route.params) {
    console.warn("Route or route.params is undefined");
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
    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
      params: params,
      query: query,
    });
  } else if (tab === TabKey.Plan) {
    router.push({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: params,
      query: query,
    });
  } else if (tab === TabKey.Rollout) {
    router.push({
      name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
      params: params,
      query: query,
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

// Banner conditions
const showClosedBanner = computed(() => {
  return (
    plan.value.state === State.DELETED ||
    (issue.value && issue.value.status === IssueStatus.CANCELED)
  );
});

const showSuccessBanner = computed(() => {
  return issue.value && issue.value.status === IssueStatus.DONE;
});

const showBanner = computed(() => {
  return showClosedBanner.value || showSuccessBanner.value;
});

useTitle(documentTitle);

// Set up navigation guards to check for unsaved changes
onBeforeRouteLeave(async (_to, _from, next) => {
  const canNavigate = await confirmNavigation();
  if (canNavigate) {
    next();
  } else {
    next(false);
  }
});

onBeforeRouteUpdate(async (_to, _from, next) => {
  const canNavigate = await confirmNavigation();
  if (canNavigate) {
    next();
  } else {
    next(false);
  }
});
</script>
