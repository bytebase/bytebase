<template>
  <BBModal
    :title="title"
    class="relative overflow-hidden !w-[30rem] !max-w-[30rem]"
    header-class="overflow-hidden"
    @close="$emit('close')"
  >
    <ReviewForm
      :action="action"
      @cancel="$emit('close')"
      @confirm="handleModalConfirm"
    />
    <div
      v-if="state.loading"
      class="absolute inset-0 flex items-center justify-center bg-white/50"
    >
      <BBSpin />
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
const { t } = useI18n();

import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import ReviewForm from "./ReviewForm.vue";
import {
  IssueReviewAction,
  targetReviewStatusForReviewAction,
  useIssueContext,
} from "@/components/IssueV1";
import { useI18n } from "vue-i18n";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action: IssueReviewAction;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const state = reactive<LocalState>({
  loading: false,
});

const { events } = useIssueContext();

const title = computed(() => {
  switch (props.action) {
    case "APPROVE":
      return t("custom-approval.issue-review.approve-issue");
    case "SEND_BACK":
      return t("custom-approval.issue-review.send-back-issue");
    case "RE_REQUEST":
      return t("custom-approval.issue-review.re-request-review-issue");
  }
  return ""; // Make linter happy
});

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
    emit("close");

    // notify the issue logic to update issue status
    events.emit("status-changed", { eager: true });
  } finally {
    state.loading = false;
  }
};
</script>
