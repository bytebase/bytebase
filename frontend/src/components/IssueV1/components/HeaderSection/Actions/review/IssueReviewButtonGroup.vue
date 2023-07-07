<template>
  <div class="flex flex-col items-end gap-y-4">
    <div class="flex items-stretch gap-x-4">
      <button
        v-if="allowReject"
        class="btn-normal"
        @click="showModal(Issue_Approver_Status.REJECTED)"
      >
        {{ $t("custom-approval.issue-review.send-back") }}
      </button>

      <BBTooltipButton
        v-if="allowApprove"
        :disabled="disallowApproveReasonList.length > 0"
        :tooltip-props="{
          placement: 'bottom-end',
        }"
        type="primary"
        tooltip-mode="DISABLED-ONLY"
        @click="showModal(Issue_Approver_Status.APPROVED)"
      >
        {{ $t("common.approve") }}

        <template #tooltip>
          <div class="whitespace-pre-line max-w-[20rem]">
            <div v-for="(reason, i) in disallowApproveReasonList" :key="i">
              {{ reason }}
            </div>
          </div>
        </template>
      </BBTooltipButton>

      <button
        v-if="allowReRequestReview"
        class="btn-primary"
        @click="showModal(Issue_Approver_Status.PENDING)"
      >
        {{ $t("custom-approval.issue-review.re-request-review") }}
      </button>

      <!-- <StandaloneIssueStatusTransitionButtonGroup
      :display-mode="
        allowApprove || allowReject || allowReRequestReview
          ? 'DROPDOWN'
          : 'BUTTON'
      "
    /> -->
    </div>

    <ReviewSection />
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

import { candidatesOfApprovalStepV1, useCurrentUserV1 } from "@/store";
import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import { BBTooltipButton } from "@/bbkit";
import { extractUserResourceName } from "@/utils";
import ReviewForm from "./ReviewForm.vue";
import { useIssueContext } from "@/components/IssueV1";
import ReviewSection from "./ReviewSection.vue";
// import { StandaloneIssueStatusTransitionButtonGroup } from "../StatusTransitionButtonGroup";

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
const { flow, ready, status, done } = reviewContext;

const allowApproveOrReject = computed(() => {
  if (phase.value !== "REVIEW") {
    return false;
  }

  if (!ready.value) return false;
  if (done.value) return false;

  const index = flow.value.currentStepIndex;
  const steps = flow.value.template.flow?.steps ?? [];
  const step = steps[index];
  if (!step) return false;
  const candidates = candidatesOfApprovalStepV1(issue.value, step);
  return candidates.includes(currentUser.value.name);
});

const allowApprove = computed(() => {
  if (!allowApproveOrReject.value) return false;

  return status.value === Issue_Approver_Status.PENDING;
});
const allowReject = computed(() => {
  if (!allowApproveOrReject.value) return false;
  return status.value === Issue_Approver_Status.PENDING;
});

const allowReRequestReview = computed(() => {
  return (
    extractUserResourceName(issue.value.creator) === currentUser.value.email &&
    status.value === Issue_Approver_Status.REJECTED
  );
});

const allTaskChecksPassed = computed(() => {
  // TODO
  return true;
  // const taskList =
  //   issue.value.pipeline?.stageList.flatMap((stage) => stage.taskList) ?? [];
  // return taskList.every((task) => {
  //   const summary = taskCheckRunSummary(task);
  //   return summary.errorCount === 0 && summary.runningCount === 0;
  // });
});

const disallowApproveReasonList = computed((): string[] => {
  const reasons: string[] = [];
  if (!allTaskChecksPassed.value) {
    reasons.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
      )
    );
  }
  return reasons;
});

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
</script>
