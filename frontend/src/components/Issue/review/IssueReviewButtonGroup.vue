<template>
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

    <StandaloneIssueStatusTransitionButtonGroup
      :display-mode="
        allowApprove || allowReject || allowReRequestReview
          ? 'DROPDOWN'
          : 'BUTTON'
      "
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
    <IssueReviewForm
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
import { computed, reactive, Ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBTooltipButton } from "@/bbkit";
import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import {
  candidatesOfApprovalStep,
  useCurrentUserV1,
  useIssueV1Store,
} from "@/store";
import { Issue } from "@/types";
import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import { extractUserUID, taskCheckRunSummary } from "@/utils";
import { StandaloneIssueStatusTransitionButtonGroup } from "../StatusTransitionButtonGroup";
import { useIssueLogic } from "../logic";
import IssueReviewForm from "./IssueReviewForm.vue";

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
const store = useIssueV1Store();
const currentUserV1 = useCurrentUserV1();
const issueContext = useIssueLogic();
const issue = issueContext.issue as Ref<Issue>;
const { flow, ready, status, done } = useIssueReviewContext();

const allowApproveOrReject = computed(() => {
  if (issue.value.status === "CANCELED" || issue.value.status === "DONE") {
    return false;
  }

  if (!ready.value) return false;
  if (done.value) return false;

  const index = flow.value.currentStepIndex;
  const steps = flow.value.template.flow?.steps ?? [];
  const step = steps[index];
  if (!step) return false;
  const candidates = candidatesOfApprovalStep(issue.value, step);
  return candidates.includes(currentUserV1.value.name);
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
    String(issue.value.creator.id) ===
      extractUserUID(currentUserV1.value.name) &&
    status.value === Issue_Approver_Status.REJECTED
  );
});

const allTaskChecksPassed = computed(() => {
  const taskList =
    issue.value.pipeline?.stageList.flatMap((stage) => stage.taskList) ?? [];
  return taskList.every((task) => {
    const summary = taskCheckRunSummary(task);
    return summary.errorCount === 0 && summary.runningCount === 0;
  });
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
    if (status === Issue_Approver_Status.APPROVED) {
      await store.approveIssue(issue.value, comment);
      onSuccess();
    } else if (status === Issue_Approver_Status.PENDING) {
      await store.requestIssue(issue.value, comment);
    } else if (status === Issue_Approver_Status.REJECTED) {
      await store.rejectIssue(issue.value, comment);
    }
    state.modal = undefined;

    // notify the issue logic to update issue status
    issueContext.onStatusChanged(true);
  } finally {
    state.loading = false;
  }
};
</script>
