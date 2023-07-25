<template>
  <div class="flex items-stretch gap-x-3">
    <ReviewActionButton
      v-for="action in issueReviewActionList"
      :key="action"
      :action="action"
      debugger
      @perform-action="showModal"
    />

    <IssueStatusActionButtonGroup
      :display-mode="issueReviewActionList.length > 0 ? 'DROPDOWN' : 'BUTTON'"
      :issue-status-action-list="issueStatusActionList"
      :extra-action-list="[]"
    />
  </div>

  <BBModal
    v-if="state.modal"
    :title="state.modal.title"
    class="relative overflow-hidden !w-[30rem] !max-w-[30rem]"
    header-class="overflow-hidden"
    @close="state.modal = undefined"
  >
    <div
      v-if="state.loading"
      class="absolute inset-0 flex items-center justify-center bg-white/50"
    >
      <BBSpin />
    </div>
    <ReviewForm
      :action="state.modal.action"
      @cancel="state.modal = undefined"
      @confirm="handleModalConfirm"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";

import { useCurrentUserV1 } from "@/store";
import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import { extractUserResourceName } from "@/utils";
import ReviewForm from "./ReviewForm.vue";
import {
  IssueReviewAction,
  getApplicableIssueStatusActionList,
  targetReviewStatusForReviewAction,
  useIssueContext,
} from "@/components/IssueV1";
import { IssueStatusActionButtonGroup } from "../common";
import ReviewActionButton from "./ReviewActionButton.vue";

type LocalState = {
  modal?: {
    title: string;
    action: IssueReviewAction;
  };
  loading: boolean;
};

const state = reactive<LocalState>({
  modal: undefined,
  loading: false,
});

const { t } = useI18n();
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

const showModal = (action: IssueReviewAction) => {
  state.modal = {
    title: "",
    action,
  };
  switch (action) {
    case "APPROVE":
      state.modal.title = t("custom-approval.issue-review.approve-issue");
      break;
    case "SEND_BACK":
      state.modal.title = t("custom-approval.issue-review.send-back-issue");
      break;
    case "RE_REQUEST":
      state.modal.title = t(
        "custom-approval.issue-review.re-request-review-issue"
      );
  }
};

const handleModalConfirm = async (
  params: {
    action: IssueReviewAction;
    comment?: string;
  },
  onSuccess: () => void
) => {
  state.loading = true;
  try {
    // TODO
    const { action } = params;
    const status = targetReviewStatusForReviewAction(action);
    if (status === Issue_Approver_Status.APPROVED) {
      // await store.approveIssue(issue.value, comment);
      onSuccess();
    } else if (status === Issue_Approver_Status.PENDING) {
      // await store.requestIssue(issue.value, comment);
    } else if (status === Issue_Approver_Status.REJECTED) {
      // await store.rejectIssue(issue.value, comment);
    }
    state.modal = undefined;

    // notify the issue logic to update issue status
    events.emit("status-changed", { eager: true });
  } finally {
    state.loading = false;
  }
};

const issueStatusActionList = computed(() => {
  return getApplicableIssueStatusActionList(issue.value);
});
</script>
