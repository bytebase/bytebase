<template>
  <NTooltip :disabled="errors.length === 0">
    <template #trigger>
      <ContextMenuButton
        :preference-key="`bb-rollout-action-${action}`"
        :action-list="actionList"
        :default-action-key="`${action}-STAGE`"
      />
    </template>
    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { useCurrentUserV1 } from "@/store";
import {
  StageRolloutAction,
  TaskRolloutAction,
  allPlanChecksPassedForTask,
  isUserAllowedToApplyTaskRolloutAction,
  taskRolloutActionDisplayName,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { RolloutButtonAction } from "./RolloutActionButtonGroup.vue";
import { ErrorList } from "../common";

const props = defineProps<{
  action: TaskRolloutAction;
  stageRolloutActionList: StageRolloutAction[];
}>();

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { issue, activeTask } = useIssueContext();

const errors = computed(() => {
  const errors: string[] = [];
  if (!allPlanChecksPassedForTask(issue.value, activeTask.value)) {
    errors.push("Some checks failed.");
  }
  if (
    !isUserAllowedToApplyTaskRolloutAction(
      issue.value,
      activeTask.value,
      props.action,
      currentUser.value
    )
  ) {
    errors.push("You are not assignee of this issue.");
  }
  return errors;
});

const actionList = computed(() => {
  const { action } = props;

  const text = taskRolloutActionDisplayName(action);
  const actionProps: RolloutButtonAction["props"] = {
    type: "primary",
    size: "large",
    tag: "div",
    disabled: errors.value.length > 0,
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
});
</script>
