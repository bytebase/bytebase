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
import Emittery from "emittery";
import { NSpin } from "naive-ui";
import { computed, toRef } from "vue";
import { useI18n } from "vue-i18n";
import {
  providePlanContext,
  useBasePlanContext,
  useInitializePlan,
} from "@/components/Plan";
import PlanDetailPage from "@/components/Plan/PlanDetailPage.vue";
import {
  providePlanCheckRunContext,
  type PlanCheckRunEvents,
} from "@/components/PlanCheckRun/context";
import { useBodyLayoutContext } from "@/layouts/common";
import { isValidPlanName } from "@/utils";

defineOptions({
  inheritAttrs: false,
});

const props = defineProps<{
  projectId: string;
  planSlug: string;
}>();

const { t } = useI18n();

const { isCreating, plan, planCheckRunList, isInitializing } =
  useInitializePlan(toRef(props, "planSlug"), toRef(props, "projectId"));
const ready = computed(() => {
  return !isInitializing.value && !!plan.value;
});
const planBaseContext = useBasePlanContext({
  isCreating,
  ready,
  plan,
});

providePlanContext(
  {
    isCreating,
    plan,
    planCheckRunList,
    ready,
    ...planBaseContext,
  },
  true /* root */
);

providePlanCheckRunContext(
  {
    events: (() => {
      const emittery: PlanCheckRunEvents = new Emittery();
      emittery.on("status-changed", () => {
        // If the status of plan checks changes, trigger a refresh.
        planBaseContext.events?.emit("status-changed", { eager: true });
      });
      return emittery;
    })(),
  },
  true /* root */
);

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("!py-0 !px-0");

const documentTitle = computed(() => {
  if (isCreating.value) {
    return t("issue.new-issue");
  } else {
    if (ready.value && isValidPlanName(plan.value.name)) {
      return plan.value.title;
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>
