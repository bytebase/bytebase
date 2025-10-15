<template>
  <NTooltip :disabled="errors.length === 0" placement="top">
    <template #trigger>
      <NButton
        size="medium"
        tag="div"
        :disabled="errors.length > 0"
        :type="action === 'SEND_BACK' ? 'default' : 'primary'"
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
import type { IssueReviewAction } from "@/components/IssueV1/logic";
import {
  issueReviewActionDisplayName,
  useIssueContext,
} from "@/components/IssueV1/logic";
import type { ErrorItem } from "@/components/misc/ErrorList.vue";
import ErrorList from "@/components/misc/ErrorList.vue";
import {
  candidatesOfApprovalStepV1,
  extractUserId,
  useCurrentUserV1,
} from "@/store";
import type { ComposedIssue } from "@/types";
import {
  Issue_ApprovalStatus,
  Issue_Approver_Status,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { displayRoleTitle, isUserIncludedInList } from "@/utils";

const props = defineProps<{
  action: IssueReviewAction;
}>();

defineEmits<{
  (event: "perform-action", action: IssueReviewAction): void;
}>();

const { t } = useI18n();
const { issue } = useIssueContext();

const errors = computed(() => {
  if (allowUserToApplyReviewAction(issue.value, props.action)) {
    return [];
  }

  const errors: ErrorItem[] = [
    t("issue.error.you-are-not-allowed-to-perform-this-action"),
  ];

  const { approvalTemplate, approvers } = issue.value;
  if (!approvalTemplate) return errors;

  const rejectedIndex = approvers.findIndex(
    (ap) => ap.status === Issue_Approver_Status.REJECTED
  );
  const currentRoleIndex =
    rejectedIndex >= 0 ? rejectedIndex : approvers.length;

  const roles = approvalTemplate.flow?.roles ?? [];
  const role = roles[currentRoleIndex];
  if (!role) return errors;

  if (role) {
    const roleTitle = displayRoleTitle(role);
    if (roleTitle) {
      errors.push({
        error: roleTitle,
        indent: 1,
      });
    }
  }

  return errors;
});

const allowUserToApplyReviewAction = (
  issue: ComposedIssue,
  action: IssueReviewAction
) => {
  if (
    issue.status === IssueStatus.CANCELED ||
    issue.status === IssueStatus.DONE ||
    issue.approvalStatus === Issue_ApprovalStatus.CHECKING ||
    issue.approvalStatus === Issue_ApprovalStatus.APPROVED ||
    issue.approvalStatus === Issue_ApprovalStatus.SKIPPED
  ) {
    return false;
  }

  const me = useCurrentUserV1();

  if (action === "RE_REQUEST") {
    return me.value.email === extractUserId(issue.creator);
  }

  const { approvalTemplate, approvers } = issue;
  if (!approvalTemplate) return false;

  const rejectedIndex = approvers.findIndex(
    (ap) => ap.status === Issue_Approver_Status.REJECTED
  );
  const currentRoleIndex =
    rejectedIndex >= 0 ? rejectedIndex : approvers.length;

  const roles = approvalTemplate.flow?.roles ?? [];
  const role = roles[currentRoleIndex];
  if (!role) return false;

  const candidates = candidatesOfApprovalStepV1(issue, role);
  return isUserIncludedInList(me.value.email, candidates);
};
</script>
