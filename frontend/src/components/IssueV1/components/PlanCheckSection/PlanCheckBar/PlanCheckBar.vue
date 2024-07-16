<template>
  <div class="w-full flex items-start gap-x-4 flex-wrap">
    <div
      class="textlabel h-[26px] inline-flex items-center"
      :class="labelClass"
    >
      {{ $t("task.task-checks") }}
    </div>

    <div class="flex-1">
      <PlanCheckBadgeBar
        :plan-check-run-list="planCheckRunList"
        @select-type="selectedType = $event"
      />
    </div>

    <div class="flex justify-end items-center shrink-0">
      <PlanCheckRunButton v-if="allowRunChecks" @run-checks="runChecks" />
    </div>

    <PlanCheckModal
      v-if="planCheckRunList.length > 0 && selectedType"
      :selected-type="selectedType"
      :plan-check-run-list="planCheckRunList"
      @close="selectedType = undefined"
    />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import {
  notifyNotEditableLegacyIssue,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { planServiceClient } from "@/grpcweb";
import type {
  PlanCheckRun,
  PlanCheckRun_Type,
} from "@/types/proto/v1/plan_service";
import type { VueClass } from "@/utils";
import PlanCheckBadgeBar from "./PlanCheckBadgeBar.vue";
import PlanCheckModal from "./PlanCheckModal.vue";
import PlanCheckRunButton from "./PlanCheckRunButton.vue";

withDefaults(
  defineProps<{
    allowRunChecks?: boolean;
    labelClass?: VueClass;
    planCheckRunList?: PlanCheckRun[];
  }>(),
  {
    allowRunChecks: true,
    labelClass: "",
    planCheckRunList: () => [],
  }
);

const { issue, events } = useIssueContext();
const selectedType = ref<PlanCheckRun_Type>();

const runChecks = () => {
  const { plan } = issue.value;
  if (!plan) {
    notifyNotEditableLegacyIssue();
    return;
  }

  planServiceClient.runPlanChecks({
    name: plan,
  });
  events.emit("status-changed", { eager: true });
};
</script>
