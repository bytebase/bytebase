<template>
  <div class="flex items-center gap-x-2">
    <!-- <BBContextMenuButton
      v-for="(transition, index) in taskStatusTransitionList"
      :key="index"
      data-label="bb-issue-status-transition-button"
      default-action-key="ROLLOUT-STAGE"
      :disabled="!allowApplyTaskTransition(transition)"
      :action-list="getButtonActionListForTransition(transition)"
      @click="(action) => onClickTaskStatusTransitionActionButton(action as TaskStatusTransitionButtonAction)"
    /> -->
    <ContextMenuButton
      v-for="(action, index) in taskRolloutActionList"
      :key="index"
      :preference-key="`bb-rollout-action-${action}`"
      :action-list="getButtonActionListForRolloutAction(action)"
      :default-action-key="`${action}-STAGE`"
    />
  </div>
</template>

<script lang="ts" setup>
import {
  StageRolloutAction,
  TaskRolloutAction,
  taskRolloutActionDisplayName,
} from "@/components/IssueV1/logic";
import { ContextMenuButton, ContextMenuButtonAction } from "@/components/v2";
import { useI18n } from "vue-i18n";

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

const props = defineProps<{
  taskRolloutActionList: TaskRolloutAction[];
  stageRolloutActionList: StageRolloutAction[];
}>();

defineEmits<{
  (event: "apply-action", action: RolloutAction): void;
}>();

const { t } = useI18n();
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

const getButtonActionListForRolloutAction = (action: TaskRolloutAction) => {
  const text = taskRolloutActionDisplayName(action);
  const actionProps: RolloutButtonAction["props"] = {
    type: "primary",
    size: "large",
  };
  const actionList: RolloutButtonAction[] = [
    {
      key: `${action}-TASK`,
      text,
      props: actionProps,
      params: {
        action,
        target: "TASK",
      },
    },
  ];
  if (props.stageRolloutActionList.includes(action as any)) {
    actionList.push({
      key: `${action}-STAGE`,
      text: t("issue.action-to-current-stage", { action: text }),
      props: actionProps,
      params: {
        action,
        target: "STAGE",
      },
    });
  }

  return actionList;
};

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
