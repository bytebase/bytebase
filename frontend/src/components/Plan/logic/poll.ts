import { computed, watch } from "vue";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { planServiceClient } from "@/grpcweb";
import { useCurrentProjectV1 } from "@/store";
import { usePlanStore } from "@/store/modules/v1/plan";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContext } from "./context";

export const usePollPlan = () => {
  const { project } = useCurrentProjectV1();
  const { isCreating, plan, planCheckRunList, events } = usePlanContext();
  const planStore = usePlanStore();

  const shouldPollPlan = computed(() => {
    return !isCreating.value;
  });

  const refreshPlan = async () => {
    if (!shouldPollPlan.value) return;

    const updatedPlan = await planStore.fetchPlanByName(plan.value.name);
    plan.value = updatedPlan;

    if (
      !isCreating.value &&
      hasProjectPermissionV2(project.value, "bb.planCheckRuns.list")
    ) {
      const { planCheckRuns } = await planServiceClient.listPlanCheckRuns({
        parent: updatedPlan.name,
        latestOnly: true,
      });
      planCheckRunList.value = planCheckRuns;
    }
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
