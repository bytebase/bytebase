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
        v-if="!state.showPlanCheckDetail"
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
      <PlanCheckRunButton
        v-if="allowRunChecks && task"
        :task="task"
        @run-checks="runChecks"
      />
    </div>

    <div v-if="state.showPlanCheckDetail" class="w-full mt-2">
      <PlanCheckPanel :plan-check-run-list="planCheckRunList" :task="task" />
    </div>

    <PlanCheckModal
      v-if="planCheckRunList.length > 0 && selectedType"
      :selected-type="selectedType"
      :plan-check-run-list="planCheckRunList"
      :task="task"
      @close="selectedType = undefined"
    />
  </div>
</template>

<script lang="ts" setup>
import { ChevronsUpDownIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive, ref } from "vue";
import {
  notifyNotEditableLegacyIssue,
  planCheckRunListForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { rolloutServiceClient } from "@/grpcweb";
import type {
  PlanCheckRun,
  PlanCheckRun_Type,
  Task,
} from "@/types/proto/v1/rollout_service";
import type { VueClass } from "@/utils";
import PlanCheckBadgeBar from "./PlanCheckBadgeBar.vue";
import PlanCheckModal from "./PlanCheckModal.vue";
import PlanCheckPanel from "./PlanCheckPanel.vue";
import PlanCheckRunButton from "./PlanCheckRunButton.vue";

interface LocalState {
  showPlanCheckDetail: boolean;
}

const props = defineProps<{
  allowRunChecks?: boolean;
  task?: Task;
  labelClass?: VueClass;
  planCheckRunList?: PlanCheckRun[];
}>();

const { issue, events } = useIssueContext();
const state = reactive<LocalState>({
  showPlanCheckDetail: false,
});
const selectedType = ref<PlanCheckRun_Type>();

const planCheckRunList = computed(() => {
  if (!props.task) {
    return props.planCheckRunList ?? issue.value.planCheckRunList;
  }
  return planCheckRunListForTask(issue.value, props.task);
});

const runChecks = (taskList: Task[]) => {
  const { plan } = issue.value;
  if (!plan) {
    notifyNotEditableLegacyIssue();
    return;
  }

  rolloutServiceClient.runPlanChecks({
    name: plan,
  });
  events.emit("status-changed", { eager: true });
};
</script>
