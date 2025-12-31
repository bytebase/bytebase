<template>
  <div ref="containerRef" class="relative overflow-x-hidden h-full">
    <template v-if="ready">
      <PollerProvider>
        <div class="h-full flex flex-col">
          <HeaderSection />

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
import { NSpin } from "naive-ui";
import { computed, ref, toRef } from "vue";
import { useI18n } from "vue-i18n";
import { onBeforeRouteLeave, onBeforeRouteUpdate } from "vue-router";
import {
  providePlanContext,
  useBasePlanContext,
  useInitializePlan,
} from "@/components/Plan";
import { HeaderSection } from "@/components/Plan/components";
import { provideSidebarContext } from "@/components/Plan/logic/sidebar";
import { useNavigationGuard } from "@/components/Plan/logic/useNavigationGuard";
import PollerProvider from "@/components/Plan/PollerProvider.vue";
import { useBodyLayoutContext } from "@/layouts/common";

const props = defineProps<{
  projectId: string;
  planId?: string;
  issueId?: string;
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
  toRef(props, "issueId")
);
const planBaseContext = useBasePlanContext({
  isCreating,
  plan,
  issue,
});
const containerRef = ref<HTMLElement>();

const ready = computed(() => {
  // Ready when we have either an issue or a valid plan, and initialization is complete
  return (!!issue.value || !!plan.value) && !isInitializing.value;
});

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

provideSidebarContext(containerRef);

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
