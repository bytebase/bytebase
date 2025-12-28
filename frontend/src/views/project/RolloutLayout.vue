<template>
  <div ref="containerRef" class="relative overflow-x-hidden h-full">
    <template v-if="ready">
      <PollerProvider>
        <div class="h-full flex flex-col">
          <div class="flex items-center justify-between pt-2 px-3 sm:px-4 shrink-0">
            <RolloutBreadcrumb />
            <RefreshIndicator />
          </div>

          <h1 class="px-3 sm:px-4 pt-1 text-lg font-medium text-main truncate shrink-0">
            {{ title }}
          </h1>

          <router-view v-slot="{ Component }">
            <keep-alive :max="3">
              <component :is="Component" />
            </keep-alive>
          </router-view>
        </div>
      </PollerProvider>
    </template>
    <div v-else class="w-full h-full flex flex-col items-center justify-center">
      <NSpin />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { NSpin } from "naive-ui";
import { computed, ref, toRef } from "vue";
import { useI18n } from "vue-i18n";
import {
  providePlanContext,
  useBasePlanContext,
  useInitializePlan,
} from "@/components/Plan";
import RefreshIndicator from "@/components/Plan/components/RefreshIndicator.vue";
import { provideSidebarContext } from "@/components/Plan/logic/sidebar";
import PollerProvider from "@/components/Plan/PollerProvider.vue";
import RolloutBreadcrumb from "@/components/RolloutV1/components/RolloutBreadcrumb.vue";
import { useBodyLayoutContext } from "@/layouts/common";

const props = defineProps<{
  projectId: string;
  planId: string;
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
  toRef(props, "planId"), // planId
  undefined, // issueId - not used for rollout routes
  undefined // legacyRolloutId - deprecated, using planId
);
const planBaseContext = useBasePlanContext({
  isCreating,
  plan,
  issue,
});
const containerRef = ref<HTMLElement>();

const ready = computed(() => {
  // Ready when we have a rollout and initialization is complete
  return !!rollout.value && !isInitializing.value;
});

const title = computed(() => {
  // Use issue title if available, otherwise plan title
  if (issue.value?.title) {
    return issue.value.title;
  }
  if (plan.value?.title) {
    return plan.value.title;
  }
  return "";
});

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
  if (ready.value && rollout.value) {
    // Use plan title if available, otherwise rollout name
    if (plan.value?.title) {
      return plan.value.title;
    }
    return t("common.rollout");
  }
  return t("common.loading");
});

useTitle(documentTitle);
</script>
```
