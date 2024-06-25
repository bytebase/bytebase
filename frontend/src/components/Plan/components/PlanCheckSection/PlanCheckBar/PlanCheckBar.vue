<template>
  <div class="w-full flex flex-col items-start gap-4">
    <div class="w-full flex flex-row justify-between items-center">
      <div class="textlabel h-[26px] inline-flex items-center">
        {{ $t("task.task-checks") }}
      </div>

      <div class="flex justify-end items-center shrink-0">
        <PlanCheckRunButton v-if="allowRunChecks" @run-checks="runChecks" />
      </div>
    </div>

    <PlanCheckPanel :plan-check-run-list="planCheckRunList" />
  </div>
</template>

<script lang="ts" setup>
import { usePlanContext } from "@/components/Plan/logic";
import { planServiceClient } from "@/grpcweb";
import type { PlanCheckRun } from "@/types/proto/v1/plan_service";
import PlanCheckPanel from "./PlanCheckPanel.vue";
import PlanCheckRunButton from "./PlanCheckRunButton.vue";

withDefaults(
  defineProps<{
    allowRunChecks?: boolean;
    planCheckRunList?: PlanCheckRun[];
  }>(),
  {
    allowRunChecks: true,
    planCheckRunList: () => [],
  }
);

const { plan } = usePlanContext();

const runChecks = () => {
  planServiceClient.runPlanChecks({
    name: plan.value.name,
  });
};
</script>
