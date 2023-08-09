<template>
  <div class="flex items-stretch gap-x-3">
    <ReviewActionButton
      v-for="action in issueReviewActionList"
      :key="action"
      :action="action"
      @perform-action="
        (action) => events.emit('perform-issue-review-action', { action })
      "
    />

    <IssueStatusActionButtonGroup
      :display-mode="issueReviewActionList.length > 0 ? 'DROPDOWN' : 'BUTTON'"
      :issue-status-action-list="issueStatusActionList"
      :extra-action-list="[]"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  IssueReviewAction,
  getApplicableIssueStatusActionList,
  useIssueContext,
} from "@/components/IssueV1";
import { useCurrentUserV1 } from "@/store";
import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import { extractUserResourceName } from "@/utils";
import { IssueStatusActionButtonGroup } from "../common";
import ReviewActionButton from "./ReviewActionButton.vue";

const currentUser = useCurrentUserV1();
const { issue, phase, reviewContext, events } = useIssueContext();
const { ready, status, done } = reviewContext;

const shouldShowApproveOrReject = computed(() => {
  if (phase.value !== "REVIEW") {
    return false;
  }

  if (!ready.value) return false;
  if (done.value) return false;

  return true;
});
const shouldShowApprove = computed(() => {
  if (!shouldShowApproveOrReject.value) return false;

  return status.value === Issue_Approver_Status.PENDING;
});
const shouldShowReject = computed(() => {
  if (!shouldShowApproveOrReject.value) return false;
  return status.value === Issue_Approver_Status.PENDING;
});
const shouldShowReRequestReview = computed(() => {
  return (
    extractUserResourceName(issue.value.creator) === currentUser.value.email &&
    status.value === Issue_Approver_Status.REJECTED
  );
});
const issueReviewActionList = computed(() => {
  const actionList: IssueReviewAction[] = [];
  if (shouldShowReject.value) {
    actionList.push("SEND_BACK");
  }
  if (shouldShowApprove.value) {
    actionList.push("APPROVE");
  }
  if (shouldShowReRequestReview.value) {
    actionList.push("RE_REQUEST");
  }

  return actionList;
});

const issueStatusActionList = computed(() => {
  return getApplicableIssueStatusActionList(issue.value);
});
</script>
