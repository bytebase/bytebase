<template>
  <div class="-mx-4 relative overflow-x-hidden">
    <template v-if="ready">
      <PlanDetailPage />
    </template>
    <div v-else class="w-full h-full flex flex-col items-center justify-center">
      <NSpin />
    </div>
  </div>
  <FeatureModal
    :open="state.showFeatureModal"
    feature="bb.feature.multi-tenancy"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import Emittery from "emittery";
import { NSpin } from "naive-ui";
import { computed, reactive, toRef } from "vue";
import { useI18n } from "vue-i18n";
import { FeatureModal } from "@/components/FeatureGuard";
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

interface LocalState {
  showFeatureModal: boolean;
}

defineOptions({
  inheritAttrs: false,
});

const props = defineProps<{
  projectId: string;
  planSlug: string;
}>();

const { t } = useI18n();

const state = reactive<LocalState>({
  showFeatureModal: false,
});

const { isCreating, plan, isInitializing, reInitialize } = useInitializePlan(
  toRef(props, "planSlug"),
  toRef(props, "projectId")
);
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
    ready,
    reInitialize,
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

overrideMainContainerClass("!py-0");

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
