<template>
  <div class="relative overflow-x-hidden h-full">
    <template v-if="ready">
      <PlanDetailPage />
    </template>
    <div v-else class="w-full h-full flex flex-col items-center justify-center">
      <NSpin />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { NSpin } from "naive-ui";
import { computed, toRef } from "vue";
import { useI18n } from "vue-i18n";
import {
  providePlanContext,
  useBasePlanContext,
  useInitializePlan,
} from "@/components/Plan";
import PlanDetailPage from "@/components/Plan/PlanDetailPage.vue";
import { useBodyLayoutContext } from "@/layouts/common";
import { isValidPlanName } from "@/utils";

defineOptions({
  inheritAttrs: false,
});

const props = defineProps<{
  projectId: string;
  planId: string;
}>();

const { t } = useI18n();
const { isCreating, plan, planCheckRunList, isInitializing } =
  useInitializePlan(toRef(props, "planId"), toRef(props, "projectId"));
const planBaseContext = useBasePlanContext({
  isCreating,
  plan,
});

const ready = computed(() => {
  return !isInitializing.value && !!plan.value;
});

providePlanContext(
  {
    isCreating,
    plan,
    planCheckRunList,
    ...planBaseContext,
  },
  true /* root */
);

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("!py-0 !px-0");

const documentTitle = computed(() => {
  if (isCreating.value) {
    return t("plan.new-plan");
  } else {
    if (ready.value && isValidPlanName(plan.value.name)) {
      return plan.value.title;
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>
