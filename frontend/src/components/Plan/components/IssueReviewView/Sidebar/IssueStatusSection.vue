<template>
  <div class="w-full flex flex-row justify-between items-center gap-2">
    <h3 class="textlabel">
      {{ $t("common.status") }}
    </h3>
    <NTag :type="issueStatusTagType" size="medium" round>
      {{ issueStatusText }}
    </NTag>
  </div>
</template>

<script setup lang="ts">
import { NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_ApprovalStatus,
  Issue_Approver_Status,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";

const props = defineProps<{
  issue: Issue;
}>();

const { t } = useI18n();

const issueStatusText = computed(() => {
  const issueValue = props.issue;

  // Show review status instead of just "Open"
  if (issueValue.approvalStatus === Issue_ApprovalStatus.CHECKING) {
    return t("issue.table.open");
  }

  const rolloutReady =
    issueValue.approvalStatus === Issue_ApprovalStatus.APPROVED ||
    issueValue.approvalStatus === Issue_ApprovalStatus.SKIPPED;
  const reviewStatus = rolloutReady
    ? Issue_Approver_Status.APPROVED
    : issueValue.approvers.some(
          (app) => app.status === Issue_Approver_Status.REJECTED
        )
      ? Issue_Approver_Status.REJECTED
      : Issue_Approver_Status.PENDING;

  switch (reviewStatus) {
    case Issue_Approver_Status.APPROVED:
      return t("issue.review.approved");
    case Issue_Approver_Status.REJECTED:
      return t("issue.review.rejected");
    default:
      return t("issue.review.under-review");
  }
});

const issueStatusTagType = computed(() => {
  const issueValue = props.issue;

  switch (issueValue.status) {
    case IssueStatus.OPEN:
      // Show different colors based on review status
      if (issueValue.approvalStatus === Issue_ApprovalStatus.CHECKING) {
        return "info";
      }

      const rolloutReady =
        issueValue.approvalStatus === Issue_ApprovalStatus.APPROVED ||
        issueValue.approvalStatus === Issue_ApprovalStatus.SKIPPED;
      const reviewStatus = rolloutReady
        ? Issue_Approver_Status.APPROVED
        : issueValue.approvers.some(
              (app) => app.status === Issue_Approver_Status.REJECTED
            )
          ? Issue_Approver_Status.REJECTED
          : Issue_Approver_Status.PENDING;

      switch (reviewStatus) {
        case Issue_Approver_Status.APPROVED:
          return "success";
        case Issue_Approver_Status.REJECTED:
          return "error";
        case Issue_Approver_Status.PENDING:
          return "warning";
        default:
          return "info";
      }
    case IssueStatus.DONE:
      return "success";
    case IssueStatus.CANCELED:
      return "default";
    default:
      return "default";
  }
});
</script>
