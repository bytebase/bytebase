<template>
  <div class="flex items-center gap-x-2">
    <BBContextMenuButton
      v-for="(transition, index) in taskStatusTransitionList"
      :key="index"
      data-label="bb-issue-status-transition-button"
      :default-action-key="`${transition.type}-STAGE`"
      :disabled="!allowApplyTaskTransition(transition)"
      :action-list="getButtonActionListForTransition(transition)"
      @click="(action) => onClickTaskStatusTransitionActionButton(action as TaskStatusTransitionButtonAction)"
    />
  </div>
</template>

<script lang="ts" setup>
import { Ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  default as BBContextMenuButton,
  type ButtonAction,
} from "@/bbkit/BBContextMenuButton.vue";
import { Issue } from "@/types";
import {
  StageStatusTransition,
  TaskStatusTransition,
  isDatabaseRelatedIssueType,
  taskCheckRunSummary,
} from "@/utils";
import { useIssueLogic } from "../logic";
import { IssueContext } from "./common";

export type TaskStatusTransitionButtonAction = ButtonAction<{
  transition: TaskStatusTransition;
  target: "TASK" | "STAGE";
}>;

const props = defineProps<{
  issueContext: IssueContext;
  taskStatusTransitionList: TaskStatusTransition[];
  stageStatusTransitionList: StageStatusTransition[];
}>();

const emit = defineEmits<{
  (
    event: "apply-task-transition",
    transition: TaskStatusTransition,
    target: "TASK" | "STAGE"
  ): void;
}>();

const { t } = useI18n();
const issueLogic = useIssueLogic();
const issue = issueLogic.issue as Ref<Issue>;

const currentTask = computed(() => {
  if (!isDatabaseRelatedIssueType(issue.value.type)) {
    return undefined;
  }
  return issueLogic.activeTaskOfPipeline(issue.value.pipeline!);
});

const allowApplyTaskTransition = (transition: TaskStatusTransition) => {
  if (transition.to === "PENDING") {
    // "Approve" is disabled when the task checks are not ready.
    const summary = taskCheckRunSummary(currentTask.value);
    if (summary.runningCount > 0 || summary.errorCount > 0) {
      return false;
    }
  }
  return true;
};

const getButtonActionListForTransition = (transition: TaskStatusTransition) => {
  const actionList: TaskStatusTransitionButtonAction[] = [];
  const { type, buttonName, buttonType } = transition;
  actionList.push({
    key: `${type}-TASK`,
    text: t(buttonName),
    type: buttonType,
    params: { transition, target: "TASK" },
  });

  if (allowApplyTaskTransitionToStage(transition)) {
    actionList.push({
      key: `${type}-STAGE`,
      text: t("issue.action-to-current-stage", {
        action: t(buttonName),
      }),
      type: buttonType,
      params: { transition, target: "STAGE" },
    });
  }

  return actionList;
};

const allowApplyTaskTransitionToStage = (transition: TaskStatusTransition) => {
  // Only available for the issue type of schema.update and data.update.
  const stage = currentTask.value?.stage;
  if (!stage) return false;

  // Only available when the stage has multiple tasks.
  if (stage.taskList.length <= 1) {
    return false;
  }

  // Available to apply a taskStatusTransition to the stage when the transition
  // type is also applicable to the stage.
  return (
    props.stageStatusTransitionList.findIndex(
      (t) => t.type === transition.type
    ) >= 0
  );
};

const onClickTaskStatusTransitionActionButton = (
  action: TaskStatusTransitionButtonAction
) => {
  const { transition, target } = action.params;
  emit("apply-task-transition", transition, target);
};
</script>
