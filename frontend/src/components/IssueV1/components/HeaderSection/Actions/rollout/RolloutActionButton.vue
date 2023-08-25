<template>
  <NTooltip :disabled="errors.length === 0" placement="top">
    <template #trigger>
      <ContextMenuButton
        :preference-key="`bb-rollout-action-${action}`"
        :action-list="actionList"
        :default-action-key="`${action}-STAGE`"
        @click="$emit('perform-action', ($event as RolloutButtonAction).params)"
      />
    </template>
    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ErrorList } from "@/components/IssueV1/components/common";
import {
  StageRolloutAction,
  TaskRolloutAction,
  allPlanChecksPassedForTask,
  allowUserToApplyTaskRolloutAction,
  taskRolloutActionButtonProps,
  taskRolloutActionDisplayName,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { ContextMenuButton } from "@/components/v2";
import { useCurrentUserV1 } from "@/store";
import { RolloutAction, RolloutButtonAction } from "./common";

const props = defineProps<{
  action: TaskRolloutAction;
  stageRolloutActionList: StageRolloutAction[];
}>();

defineEmits<{
  (event: "perform-action", action: RolloutAction): void;
}>();

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { issue, activeTask } = useIssueContext();

const errors = asyncComputed(async () => {
  const errors: string[] = [];
  if (!allPlanChecksPassedForTask(issue.value, activeTask.value)) {
    errors.push("Some checks failed");
  }
  if (
    !(await allowUserToApplyTaskRolloutAction(
      issue.value,
      activeTask.value,
      currentUser.value,
      props.action
    ))
  ) {
    errors.push("You are not the assignee of this issue");
  }
  return errors;
}, []);

const actionList = computed(() => {
  const { action } = props;

  const text = taskRolloutActionDisplayName(action);
  const actionProps: RolloutButtonAction["props"] = {
    ...taskRolloutActionButtonProps(action),
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
