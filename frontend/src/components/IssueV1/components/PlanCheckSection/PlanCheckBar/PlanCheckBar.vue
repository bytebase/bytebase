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
      >
        <NTooltip v-if="planCheckRunList.length > 0" trigger="hover">
          <template #trigger>
            <NButton
              strong
              secondary
              circle
              bordered
              size="small"
              type="tertiary"
              @click="state.showPlanCheckDetail = true"
            >
              <template #icon>
                <ChevronsUpDownIcon class="w-4 h-auto" />
              </template>
            </NButton>
          </template>
          {{ $t("common.expand") }}
        </NTooltip>
      </PlanCheckBadgeBar>
    </div>

    <div class="flex justify-end items-center shrink-0">
      <PlanCheckRunButton v-if="allowRunChecks" @run-checks="runChecks" />
    </div>

    <div v-if="state.showPlanCheckDetail" class="w-full mt-2">
      <PlanCheckPanel :plan-check-run-list="planCheckRunList" />
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
import { ChevronsUpDownIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { reactive, ref } from "vue";
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
import PlanCheckPanel from "./PlanCheckPanel.vue";
import PlanCheckRunButton from "./PlanCheckRunButton.vue";

interface LocalState {
  showPlanCheckDetail: boolean;
}

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
const state = reactive<LocalState>({
  showPlanCheckDetail: !issue.value.rollout, // Show plan check detail by default if there is no rollout.
});
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
