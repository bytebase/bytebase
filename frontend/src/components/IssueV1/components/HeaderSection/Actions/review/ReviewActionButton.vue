<template>
  <NTooltip :disabled="errors.length === 0" placement="top">
    <template #trigger>
      <NButton
        size="medium"
        tag="div"
        :disabled="errors.length > 0"
        v-bind="issueReviewActionButtonProps(action)"
        @click="$emit('perform-action', action)"
      >
        {{ issueReviewActionDisplayName(action) }}
      </NButton>
    </template>
    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { NButton, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  IssueReviewAction,
  issueReviewActionDisplayName,
  issueReviewActionButtonProps,
  allowUserToApplyReviewAction,
  useIssueContext,
} from "@/components/IssueV1/logic";
import ErrorList, { ErrorItem } from "@/components/misc/ErrorList.vue";
import { useCurrentUserV1 } from "@/store";
import { PresetRoleType } from "@/types";
import {
  ApprovalNode_GroupValue,
  ApprovalNode_Type,
} from "@/types/proto/v1/issue_service";
import { displayRoleTitle } from "@/utils";

const props = defineProps<{
  action: IssueReviewAction;
}>();

defineEmits<{
  (event: "perform-action", action: IssueReviewAction): void;
}>();

const { t } = useI18n();
const { issue, reviewContext } = useIssueContext();
const currentUser = useCurrentUserV1();

const errors = computed(() => {
  const errors: ErrorItem[] = [];

  if (
    !allowUserToApplyReviewAction(
      issue.value,
      reviewContext,
      currentUser.value,
      props.action
    )
  ) {
    errors.push(t("issue.error.you-are-not-allowed-to-perform-this-action"));
    const flow = reviewContext.flow.value;
    const index = flow.currentStepIndex;
    const steps = flow.template.flow?.steps ?? [];
    const step = steps[index];
    if (step) {
      for (let i = 0; i < step.nodes.length; i++) {
        const node = step.nodes[i];
        const {
          type,
          groupValue = ApprovalNode_GroupValue.UNRECOGNIZED,
          role,
        } = node;
        if (type !== ApprovalNode_Type.ANY_IN_GROUP) continue;
        if (groupValue === ApprovalNode_GroupValue.WORKSPACE_OWNER) {
          errors.push({
            error: displayRoleTitle(PresetRoleType.WORKSPACE_ADMIN),
            indent: 1,
          });
        } else if (groupValue === ApprovalNode_GroupValue.WORKSPACE_DBA) {
          errors.push({
            error: displayRoleTitle(PresetRoleType.WORKSPACE_DBA),
            indent: 1,
          });
        } else if (groupValue === ApprovalNode_GroupValue.PROJECT_OWNER) {
          errors.push({
            error: displayRoleTitle(PresetRoleType.PROJECT_OWNER),
            indent: 1,
          });
        } else if (groupValue === ApprovalNode_GroupValue.PROJECT_MEMBER) {
          errors.push({
            error: t("common.project-member"),
            indent: 1,
          });
        } else if (role) {
          errors.push({
            error: displayRoleTitle(role),
            indent: 1,
          });
        }
      }
    }
  }
  return errors;
});
</script>
