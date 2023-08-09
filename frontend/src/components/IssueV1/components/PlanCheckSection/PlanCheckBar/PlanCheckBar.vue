<template>
  <!-- <div
    v-if="task.taskCheckRunList.length > 0"
    class="flex items-start space-x-4"
  >
    <div class="textlabel h-[26px] inline-flex items-center">
      {{ $t("task.task-checks") }}
    </div>

    <TaskCheckBadgeBar
      :task-check-run-list="task.taskCheckRunList"
      @select-task-check-type="viewCheckRunDetail"
    />

    <RunTaskCheckButton v-if="allowRunTask" @run-checks="runChecks" />

    <BBModal
      v-if="state.showModal"
      :title="$t('task.check-result.title', { name: task.name })"
      class="!w-[56rem]"
      header-class="whitespace-pre-wrap break-all gap-x-1"
      @close="dismissDialog"
    >
      <div class="space-y-4">
        <div>
          <TaskCheckBadgeBar
            :task-check-run-list="task.taskCheckRunList"
            :allow-selection="true"
            :sticky-selection="true"
            :selected-task-check-type="state.selectedTaskCheckType"
            @select-task-check-type="viewCheckRunDetail"
          />
        </div>
        <BBTabFilter
          class="pt-4"
          :tab-item-list="tabItemList"
          :selected-index="state.selectedTabIndex"
          @select-index="
            (index: number) => {
              state.selectedTabIndex = index;
            }
          "
        />
        <TaskCheckRunPanel
          v-if="selectedTaskCheckRun"
          :task-check-run="selectedTaskCheckRun"
          :task="task"
        />
        <div class="pt-4 flex justify-end">
          <button
            type="button"
            class="btn-primary py-2 px-4"
            @click.prevent="dismissDialog"
          >
            {{ $t("common.close") }}
          </button>
        </div>
      </div>
    </BBModal>
  </div> -->

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
