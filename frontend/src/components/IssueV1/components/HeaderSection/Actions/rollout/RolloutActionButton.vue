<template>
  <NTooltip :disabled="errors.length === 0" placement="top">
    <template #trigger>
      <ContextMenuButton
        :preference-key="`bb-rollout-action-${action}`"
        :action-list="actionList"
        :default-action-key="`${action}-STAGE`"
        size="medium"
        @click="$emit('perform-action', ($event as RolloutButtonAction).params)"
      />
    </template>
    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type {
  StageRolloutAction,
  TaskRolloutAction,
} from "@/components/IssueV1/logic";
import {
  taskRolloutActionButtonProps,
  taskRolloutActionDisplayName,
  useIssueContext,
} from "@/components/IssueV1/logic";
import type { ErrorItem } from "@/components/misc/ErrorList.vue";
import ErrorList from "@/components/misc/ErrorList.vue";
import { canRolloutTasks } from "@/components/RolloutV1/components/taskPermissions";
import { ContextMenuButton } from "@/components/v2";
import type { RolloutAction, RolloutButtonAction } from "./common";

const props = defineProps<{
  action: TaskRolloutAction;
  stageRolloutActionList: StageRolloutAction[];
}>();

defineEmits<{
  (event: "perform-action", action: RolloutAction): void;
}>();

const { t } = useI18n();
const { issue, selectedTask } = useIssueContext();

const errors = computed(() => {
  const errors: ErrorItem[] = [];
  if (!canRolloutTasks([selectedTask.value], issue.value)) {
    errors.push(t("issue.error.you-are-not-allowed-to-perform-this-action"));
  }
  return errors;
});

const actionList = computed(() => {
  const { action } = props;

  const text = taskRolloutActionDisplayName(action, selectedTask.value);
  const actionProps: RolloutButtonAction["props"] = {
    ...taskRolloutActionButtonProps(action),
    tag: "div",
    disabled: errors.value.length > 0,
  };
  if (selectedTask.value.payload?.case === "databaseDataExport") {
    return [
      {
        key: `${action}-STAGE`,
        text,
        props: actionProps,
        params: {
          action,
          target: "STAGE",
        },
      },
    ];
  }

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
  if (props.stageRolloutActionList.includes(action)) {
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
