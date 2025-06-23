import { computed, watch } from "vue";
import { create } from "@bufbuild/protobuf";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { planServiceClientConnect } from "@/grpcweb";
import { ListPlanCheckRunsRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { convertNewPlanCheckRunToOld } from "@/utils/v1/plan-conversions";
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
      const request = create(ListPlanCheckRunsRequestSchema, {
        parent: updatedPlan.name,
        latestOnly: true,
      });
      const response = await planServiceClientConnect.listPlanCheckRuns(request);
      planCheckRunList.value = response.planCheckRuns.map(convertNewPlanCheckRunToOld);
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
