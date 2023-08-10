<template>
  <div class="flex items-start gap-x-4">
    <div class="textlabel h-[26px] inline-flex items-center">
      {{ $t("task.task-checks") }}
    </div>

    <PlanCheckBadgeBar
      :plan-check-run-list="planCheckRunList"
      :task="task"
      @select-type="selectedType = $event"
    />

    <PlanCheckRunButton
      v-if="allowRunChecks"
      :task="task"
      @run-checks="runChecks"
    />

    <PlanCheckPanel
      v-if="planCheckRunList.length > 0 && selectedType"
      :selected-type="selectedType"
      :plan-check-run-list="planCheckRunList"
      :task="task"
      @close="selectedType = undefined"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import {
  notifyNotEditableLegacyIssue,
  planCheckRunListForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { rolloutServiceClient } from "@/grpcweb";
import { PlanCheckRun_Type, Task } from "@/types/proto/v1/rollout_service";
import PlanCheckBadgeBar from "./PlanCheckBadgeBar.vue";
import PlanCheckPanel from "./PlanCheckPanel.vue";
import PlanCheckRunButton from "./PlanCheckRunButton.vue";

const props = defineProps<{
  allowRunChecks?: boolean;
  task: Task;
}>();

const { issue, events } = useIssueContext();
const selectedType = ref<PlanCheckRun_Type>();

const planCheckRunList = computed(() => {
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
