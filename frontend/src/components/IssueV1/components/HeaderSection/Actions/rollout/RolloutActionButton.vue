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
import { asyncComputed } from "@vueuse/core";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type {
  StageRolloutAction,
  TaskRolloutAction,
} from "@/components/IssueV1/logic";
import {
  allowUserToApplyTaskRolloutAction,
  taskRolloutActionButtonProps,
  taskRolloutActionDisplayName,
  useIssueContext,
} from "@/components/IssueV1/logic";
import type { ErrorItem } from "@/components/misc/ErrorList.vue";
import ErrorList from "@/components/misc/ErrorList.vue";
import { ContextMenuButton } from "@/components/v2";
import { useUserStore } from "@/store";
import { userNamePrefix, roleNamePrefix } from "@/store/modules/v1/common";
import { displayRoleTitle, extractUserResourceName } from "@/utils";
import type { RolloutAction, RolloutButtonAction } from "./common";

const props = defineProps<{
  action: TaskRolloutAction;
  stageRolloutActionList: StageRolloutAction[];
}>();

defineEmits<{
  (event: "perform-action", action: RolloutAction): void;
}>();

const { t } = useI18n();
const { issue, selectedTask, releaserCandidates } = useIssueContext();

const errors = asyncComputed(async () => {
  const errors: ErrorItem[] = [];
  if (
    !(await allowUserToApplyTaskRolloutAction(
      issue.value,
      selectedTask.value,
      props.action,
      releaserCandidates.value
    ))
  ) {
    errors.push(t("issue.error.you-are-not-allowed-to-perform-this-action"));
    const { releasers } = issue.value;
    for (let i = 0; i < releasers.length; i++) {
      const roleOrUser = releasers[i];
      if (roleOrUser.startsWith(roleNamePrefix)) {
        errors.push({
          error: displayRoleTitle(roleOrUser),
          indent: 1,
        });
      }
      if (roleOrUser.startsWith(userNamePrefix)) {
        const email = extractUserResourceName(roleOrUser);
        const user = useUserStore().getUserByEmail(email);
        if (user) {
          errors.push({
            error: `${user.title} (${user.email})`,
            indent: 1,
          });
        }
      }
    }
  }
  return errors;
}, []);

const actionList = computed(() => {
  const { action } = props;

  const text = taskRolloutActionDisplayName(action, selectedTask.value);
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
