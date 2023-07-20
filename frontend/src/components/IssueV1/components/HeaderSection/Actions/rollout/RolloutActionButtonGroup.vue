<template>
  <div class="flex items-center gap-x-2">
    <RolloutActionButton
      v-for="(action, index) in taskRolloutActionList"
      :key="index"
      :action="action"
      :stage-rollout-action-list="stageRolloutActionList"
    />
  </div>
</template>

<script lang="ts" setup>
import {
  StageRolloutAction,
  TaskRolloutAction,
} from "@/components/IssueV1/logic";
import { ContextMenuButtonAction } from "@/components/v2";
import RolloutActionButton from "./RolloutActionButton.vue";

// import { Issue } from "@/types";
// import {
//   StageStatusTransition,
//   TaskStatusTransition,
//   isDatabaseRelatedIssueType,
//   taskCheckRunSummary,
// } from "@/utils";
// import { useIssueLogic } from "../logic";
// import {
//   default as BBContextMenuButton,
//   type ButtonAction,
// } from "@/bbkit/BBContextMenuButton.vue";
// import { IssueContext } from "./common";

// export type TaskStatusTransitionButtonAction = ButtonAction<{
//   transition: TaskStatusTransition;
//   target: "TASK" | "STAGE";
// }>;

export type RolloutAction<T = "TASK" | "STAGE"> = {
  target: T;
  action: T extends "TASK" ? TaskRolloutAction : StageRolloutAction;
};

export type RolloutButtonAction = ContextMenuButtonAction<RolloutAction>;

defineProps<{
  taskRolloutActionList: TaskRolloutAction[];
  stageRolloutActionList: StageRolloutAction[];
}>();

defineEmits<{
  (event: "apply-action", action: RolloutAction): void;
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
