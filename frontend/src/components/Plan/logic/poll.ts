import { computed, watch } from "vue";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { usePlanStore } from "@/store/modules/v1/plan";
import { usePlanContext } from "./context";

export const usePollPlan = () => {
  const { isCreating, plan, events } = usePlanContext();
  const planStore = usePlanStore();

  const shouldPollPlan = computed(() => {
    return !isCreating.value;
  });

  const refreshPlan = () => {
    if (!shouldPollPlan.value) return;

    planStore
      .fetchPlanByName(plan.value.name)
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

  events.on("status-changed", ({ eager }) => {
    if (eager) {
      refreshPlan();
      poller.restart();
    }
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
