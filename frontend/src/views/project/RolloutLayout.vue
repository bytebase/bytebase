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
import { computed, ref, toRef, watchEffect } from "vue";
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
import { usePolicyV1Store } from "@/store";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";

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
} = useInitializePlan(toRef(props, "projectId"), toRef(props, "planId"));
const planBaseContext = useBasePlanContext({
  isCreating,
  plan,
  issue,
});
const containerRef = ref<HTMLElement>();
const policyStore = usePolicyV1Store();

const ready = computed(() => !!rollout.value && !isInitializing.value);

// Prefetch rollout policies for all stages
watchEffect(() => {
  if (!rollout.value) return;
  for (const stage of rollout.value.stages) {
    if (stage.environment) {
      policyStore
        .getOrFetchPolicyByParentAndType({
          parentPath: stage.environment,
          policyType: PolicyType.ROLLOUT_POLICY,
        })
        .catch(() => {});
    }
  }
});
const title = computed(() => issue.value?.title || plan.value?.title || "");

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

useBodyLayoutContext().overrideMainContainerClass("py-0! px-0!");

useTitle(
  computed(() => {
    if (ready.value && plan.value?.title) {
      return plan.value.title;
    }
    return ready.value ? t("common.rollout") : t("common.loading");
  })
);
</script>
