import { computed, watch } from "vue";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { usePlanStore } from "@/store/modules/v1/plan";
import { extractProjectResourceName } from "@/utils";
import { usePlanContext } from "./context";

export const usePollPlan = () => {
  const { isCreating, ready, plan } = usePlanContext();
  const planStore = usePlanStore();

  const shouldPollPlan = computed(() => {
    return !isCreating.value && ready.value;
  });

  const refreshPlan = () => {
    if (!shouldPollPlan.value) return;

    planStore
      .fetchPlanByUID(
        plan.value.uid,
        extractProjectResourceName(plan.value.name)
      )
      .then((updatedPlan) => (plan.value = updatedPlan));
  };

  const poller = useProgressivePoll(refreshPlan, {
    interval: {
      min: 500,
      max: 10000,
      growth: 2,
      jitter: 500,
    },
  });

  watch(
    shouldPollPlan,
    () => {
      if (shouldPollPlan.value) {
        poller.start();
      } else {
        poller.stop();
      }
    },
    {
      immediate: true,
    }
  );
};
