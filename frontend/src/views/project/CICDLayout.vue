<template>
  <div ref="containerRef" class="relative overflow-x-hidden h-full">
    <template v-if="ready">
      <PollerProvider>
        <div class="h-full flex flex-col">
          <BannerSection />

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
            />

            <!-- Suffix slot -->
            <template #suffix>
              <div class="pr-3 flex flex-row justify-end items-center gap-4">
                <RefreshIndicator />
              </div>
            </template>
          </NTabs>

          <div class="flex-1 flex">
            <router-view v-slot="{ Component }">
              <keep-alive :max="3">
                <component :is="Component" />
              </keep-alive>
            </router-view>
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
import { NSpin, NTab, NTabs } from "naive-ui";
import { computed, ref, toRef } from "vue";
import { useI18n } from "vue-i18n";
import {
  onBeforeRouteLeave,
  onBeforeRouteUpdate,
  useRoute,
  useRouter,
} from "vue-router";
import {
  providePlanContext,
  useBasePlanContext,
  useInitializePlan,
} from "@/components/Plan";
import { BannerSection, HeaderSection } from "@/components/Plan/components";
import RefreshIndicator from "@/components/Plan/components/RefreshIndicator.vue";
import { provideSidebarContext } from "@/components/Plan/logic/sidebar";
import { useCICDTabNavigation } from "@/components/Plan/logic/useCICDTabNavigation.tsx";
import { useNavigationGuard } from "@/components/Plan/logic/useNavigationGuard";
import PollerProvider from "@/components/Plan/PollerProvider.vue";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { useBodyLayoutContext } from "@/layouts/common";

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
const containerRef = ref<HTMLElement>();

const ready = computed(() => {
  // Ready when we have either an issue or a valid plan, and initialization is complete
  return (!!issue.value || !!plan.value) && !isInitializing.value;
});

const shouldShowNavigation = computed(() => {
  // Or if we have a valid plan with supported specs
  if (!plan.value) {
    return false;
  }
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
  issueLabels: ref<string[]>(issue.value?.labels ?? []),
  ...planBaseContext,
});

provideSidebarContext(containerRef);

// Tab navigation
const { tabKey, availableTabs, tabRender, handleTabChange } =
  useCICDTabNavigation({
    route,
    router,
    plan,
    issue,
    isCreating,
    enabledNewLayout,
    t,
  });

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("py-0! px-0!");

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

// Set up navigation guards to check for unsaved changes
const handleRouteNavigation = async (
  _to: unknown,
  _from: unknown,
  next: (proceed?: boolean) => void
) => {
  const canNavigate = await confirmNavigation();
  next(canNavigate);
};

onBeforeRouteLeave(handleRouteNavigation);
onBeforeRouteUpdate(handleRouteNavigation);
</script>
