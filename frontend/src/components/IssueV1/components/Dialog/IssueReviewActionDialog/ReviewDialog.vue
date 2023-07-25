<template>
  <CommonDialog :title="title" :loading="state.loading" @close="$emit('close')">
    <ReviewForm
      :action="action"
      @cancel="$emit('close')"
      @confirm="handleConfirm"
    />
  </CommonDialog>
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";

import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import {
  IssueReviewAction,
  targetReviewStatusForReviewAction,
  useIssueContext,
} from "@/components/IssueV1";
import CommonDialog from "../CommonDialog.vue";
import ReviewForm from "./ReviewForm.vue";
import { issueServiceClient } from "@/grpcweb";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action: IssueReviewAction;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});

const { issue, events } = useIssueContext();

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

const handleConfirm = async (
  params: {
    action: IssueReviewAction;
    comment?: string;
  },
  onSuccess: () => void
) => {
  state.loading = true;
  try {
    const { action, comment = "" } = params;
    const status = targetReviewStatusForReviewAction(action);
    if (status === Issue_Approver_Status.APPROVED) {
      // await store.approveIssue(issue.value, comment);
      onSuccess();
    } else if (status === Issue_Approver_Status.PENDING) {
      await issueServiceClient.requestIssue({
        name: issue.value.name,
        comment,
      });
      //
    } else if (status === Issue_Approver_Status.REJECTED) {
      await issueServiceClient.rejectIssue({
        name: issue.value.name,
        comment,
      });
    }

    // notify the issue logic to update issue status
    events.emit("status-changed", { eager: true });
  } finally {
    state.loading = false;
    emit("close");
  }
};
</script>
