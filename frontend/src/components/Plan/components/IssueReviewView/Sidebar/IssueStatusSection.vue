<template>
  <div v-if="issueStatusText" class="w-full flex flex-row justify-between items-center gap-2">
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

const isRejected = computed(() =>
  props.issue.approvers.some(
    (app) => app.status === Issue_Approver_Status.REJECTED
  )
);

const issueStatusText = computed(() => {
  const issueValue = props.issue;
  if (issueValue.status !== IssueStatus.OPEN) {
    return "";
  }

  // Issue is rollout-ready if approved, skipped, or has no approval required (empty flow)
  const roles = issueValue.approvalTemplate?.flow?.roles ?? [];
  const noApprovalRequired = roles.length === 0;
  const rolloutReady =
    issueValue.approvalStatus === Issue_ApprovalStatus.APPROVED ||
    issueValue.approvalStatus === Issue_ApprovalStatus.SKIPPED ||
    noApprovalRequired;
  if (rolloutReady) {
    return t("issue.review.approved");
  }
  if (isRejected.value) {
    return t("issue.review.rejected");
  }
  return t("issue.review.under-review");
});

const issueStatusTagType = computed(() => {
  const issueValue = props.issue;
  if (issueValue.status !== IssueStatus.OPEN) {
    return "default";
  }

  if (isRejected.value) {
    return "error";
  }
  return "success";
});
</script>
