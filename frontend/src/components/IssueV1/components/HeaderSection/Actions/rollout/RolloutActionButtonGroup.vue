<template>
  <div class="flex items-center gap-x-3">
    <RolloutActionButton
      v-for="(action, index) in taskRolloutActionList"
      :key="index"
      :action="action"
      :stage-rollout-action-list="stageRolloutActionList"
      @perform-action="$emit('perform-action', $event)"
    />
  </div>
</template>

<script lang="ts" setup>
import {
  StageRolloutAction,
  TaskRolloutAction,
} from "@/components/IssueV1/logic";
import RolloutActionButton from "./RolloutActionButton.vue";
import { RolloutAction } from "./common";

defineProps<{
  taskRolloutActionList: TaskRolloutAction[];
  stageRolloutActionList: StageRolloutAction[];
}>();

defineEmits<{
  (event: "perform-action", action: RolloutAction): void;
}>();

// const issueLogic = useIssueLogic();
// const issue = issueLogic.issue as Ref<Issue>;

// const currentTask = computed(() => {
//   if (!isDatabaseRelatedIssueType(issue.value.type)) {
//     return undefined;
//   }
//   return issueLogic.activeTaskOfPipeline(issue.value.pipeline!);
// });

// const allowApplyTaskTransition = (transition: TaskStatusTransition) => {
//   if (transition.to === "PENDING") {
//     // "Approve" is disabled when the task checks are not ready.
//     const summary = taskCheckRunSummary(currentTask.value);
//     if (summary.runningCount > 0 || summary.errorCount > 0) {
//       return false;
//     }
//   }
//   return true;
// };

// const allowApplyTaskTransitionToStage = (transition: TaskStatusTransition) => {
//   // Only available for the issue type of schema.update and data.update.
//   const stage = currentTask.value?.stage;
//   if (!stage) return false;

//   // Only available when the stage has multiple tasks.
//   if (stage.taskList.length <= 1) {
//     return false;
//   }

//   // Available to apply a taskStatusTransition to the stage when the transition
//   // type is also applicable to the stage.
//   return (
//     props.stageStatusTransitionList.findIndex(
//       (t) => t.type === transition.type
//     ) >= 0
//   );
// };

// const onClickTaskStatusTransitionActionButton = (
//   action: TaskStatusTransitionButtonAction
// ) => {
//   const { transition, target } = action.params;
//   emit("apply-task-transition", transition, target);
// };
</script>
