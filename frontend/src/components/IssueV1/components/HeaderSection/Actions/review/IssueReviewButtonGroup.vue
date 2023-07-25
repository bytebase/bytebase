<template>
  <div class="issue-debug">
    <div>allowReject: {{ shouldShowReject }}</div>
    <div>allowApprove: {{ shouldShowApprove }}</div>
    <div>allowReRequestReview: {{ shouldShowReRequestReview }}</div>
  </div>
  <div class="flex items-stretch gap-x-3">
    <ReviewActionButton
      v-for="action in issueReviewActionList"
      :key="action"
      :action="action"
      @perform-action="handleApplyAction"
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
      :status="state.modal.status"
      :ok-text="state.modal.okText"
      :button-style="state.modal.buttonStyle"
      :review-type="state.modal.reviewType"
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
  useIssueContext,
} from "@/components/IssueV1";
import { IssueStatusActionButtonGroup } from "../common";
import ReviewActionButton from "./ReviewActionButton.vue";

type LocalState = {
  modal?: {
    title: string;
    status: Issue_Approver_Status;
    okText: string;
    buttonStyle: "PRIMARY" | "ERROR" | "NORMAL";
    reviewType: "APPROVAL" | "SEND_BACK" | "RE_REQUEST_REVIEW";
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

const handleApplyAction = (action: IssueReviewAction) => {
  switch (action) {
    case "APPROVE":
      showModal(Issue_Approver_Status.APPROVED);
      break;
    case "SEND_BACK":
      showModal(Issue_Approver_Status.REJECTED);
      break;
    case "RE_REQUEST":
      showModal(Issue_Approver_Status.PENDING);
      break;
  }
};

const showModal = (status: Issue_Approver_Status) => {
  state.modal = {
    status,
    title: "",
    okText: "",
    buttonStyle: "NORMAL",
    reviewType: "APPROVAL",
  };
  switch (status) {
    case Issue_Approver_Status.APPROVED:
      state.modal.title = t("custom-approval.issue-review.approve-issue");
      state.modal.okText = t("common.approval");
      state.modal.buttonStyle = "PRIMARY";
      state.modal.reviewType = "APPROVAL";
      break;
    case Issue_Approver_Status.REJECTED:
      state.modal.title = t("custom-approval.issue-review.send-back-issue");
      state.modal.okText = t("custom-approval.issue-review.send-back");
      state.modal.buttonStyle = "PRIMARY";
      state.modal.reviewType = "SEND_BACK";
      break;
    case Issue_Approver_Status.PENDING:
      state.modal.title = t(
        "custom-approval.issue-review.re-request-review-issue"
      );
      state.modal.okText = t("custom-approval.issue-review.re-request-review");
      state.modal.buttonStyle = "PRIMARY";
      state.modal.reviewType = "RE_REQUEST_REVIEW";
  }
};

const handleModalConfirm = async (
  {
    status,
    comment,
  }: {
    status: Issue_Approver_Status;
    comment?: string;
  },
  onSuccess: () => void
) => {
  state.loading = true;
  try {
    // TODO
    // if (status === Issue_Approver_Status.APPROVED) {
    //   await store.approveIssue(issue.value, comment);
    //   onSuccess();
    // } else if (status === Issue_Approver_Status.PENDING) {
    //   await store.requestIssue(issue.value, comment);
    // } else if (status === Issue_Approver_Status.REJECTED) {
    //   await store.rejectIssue(issue.value, comment);
    // }
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
