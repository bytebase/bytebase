<template>
  <button
    v-if="showApproveButton"
    class="btn-primary"
    @click="state.modal = true"
  >
    {{ $t("common.approve") }}
  </button>

  <BBModal
    v-if="state.modal"
    :title="$t('custom-approval.issue-review.approve-issue')"
    class="relative overflow-hidden !w-[30rem] !max-w-[30rem]"
    header-class="overflow-hidden"
    @close="state.modal = false"
  >
    <div
      v-if="state.loading"
      class="absolute inset-0 flex items-center justify-center bg-white/50"
    >
      <BBSpin />
    </div>
    <IssueReviewForm
      @cancel="state.modal = false"
      @confirm="handleConfirmApprove"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, reactive, Ref } from "vue";

import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import {
  candidatesOfApprovalStep,
  useCurrentUserV1,
  useReviewStore,
} from "@/store";
import { Issue } from "@/types";
import { useIssueLogic } from "../logic";
import IssueReviewForm from "./IssueReviewForm.vue";

type LocalState = {
  modal: boolean;
  loading: boolean;
};

const state = reactive<LocalState>({
  modal: false,
  loading: false,
});

const store = useReviewStore();
const currentUserV1 = useCurrentUserV1();
const issueContext = useIssueLogic();
const issue = issueContext.issue as Ref<Issue>;
const { flow, ready, done } = useIssueReviewContext();

const showApproveButton = computed(() => {
  if (issue.value.status === "CANCELED" || issue.value.status === "DONE") {
    return false;
  }

  if (!ready.value) return false;
  if (done.value) return false;

  const index = flow.value.currentStepIndex;
  const steps = flow.value.template.flow?.steps ?? [];
  const step = steps[index];
  if (!step) return [];
  const candidates = candidatesOfApprovalStep(issue.value, step);
  return candidates.includes(currentUserV1.value.name);
});

const handleConfirmApprove = async (onSuccess: () => void) => {
  state.loading = true;
  try {
    await store.approveReview(issue.value);
    onSuccess();
    state.modal = false;

    // notify the issue logic to update issue status
    issueContext.onStatusChanged(true);
  } finally {
    state.loading = false;
  }
};
</script>
