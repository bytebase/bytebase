<template>
  <CommonDrawer
    :show="action !== undefined"
    :title="title"
    :loading="state.loading"
    @show="comment = ''"
    @close="$emit('close')"
  >
    <template #default>
      <div v-if="action" class="flex flex-col gap-y-4">
        <div class="flex flex-col gap-y-1">
          <div class="font-medium text-control">
            {{ $t("common.issue") }}
          </div>
          <div class="textinfolabel">
            {{ issue.title }}
          </div>
        </div>

        <div class="flex flex-col gap-y-1">
          <p class="font-medium text-control">
            {{ $t("common.comment") }}
            <RequiredStar v-show="props.action === 'SEND_BACK'" />
          </p>
          <NInput
            v-model:value="comment"
            type="textarea"
            :placeholder="$t('issue.leave-a-comment')"
            :autosize="{
              minRows: 3,
              maxRows: 10,
            }"
          />
        </div>
      </div>
    </template>
    <template #footer>
      <div v-if="action" class="flex justify-end gap-x-3">
        <NButton @click="$emit('close')">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          :disabled="!allowConfirm"
          v-bind="issueReviewActionButtonProps(action)"
          @click="handleClickConfirm"
        >
          {{ issueReviewActionDisplayName(action) }}
        </NButton>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  useIssueContext,
  IssueReviewAction,
  targetReviewStatusForReviewAction,
  issueReviewActionButtonProps,
  issueReviewActionDisplayName,
} from "@/components/IssueV1/logic";
import { issueServiceClient } from "@/grpcweb";
import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import CommonDrawer from "./CommonDrawer.vue";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action?: IssueReviewAction;
}>();
const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});
const { events, issue } = useIssueContext();
const comment = ref("");

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

const allowConfirm = computed(() => {
  if (props.action === "SEND_BACK" && comment.value === "") {
    return false;
  }

  return true;
});

const handleClickConfirm = (e: MouseEvent) => {
  const button = e.target as HTMLElement;
  const { left, top, width, height } = button.getBoundingClientRect();
  const { innerWidth: winWidth, innerHeight: winHeight } = window;
  const onSuccess = () => {
    if (props.action !== "APPROVE") {
      return;
    }
    // import the effect lib asynchronously
    import("canvas-confetti").then(({ default: confetti }) => {
      // Create a confetti effect from the position of the LGTM button
      confetti({
        particleCount: 100,
        spread: 70,
        origin: {
          x: (left + width / 2) / winWidth,
          y: (top + height / 2) / winHeight,
        },
      });
    });
  };

  handleConfirm(onSuccess);
};

const handleConfirm = async (onSuccess: () => void) => {
  const { action } = props;
  if (!action) return;
  state.loading = true;
  try {
    const status = targetReviewStatusForReviewAction(action);
    if (status === Issue_Approver_Status.APPROVED) {
      await issueServiceClient.approveIssue({
        name: issue.value.name,
        comment: comment.value,
      });
      onSuccess();
    } else if (status === Issue_Approver_Status.PENDING) {
      await issueServiceClient.requestIssue({
        name: issue.value.name,
        comment: comment.value,
      });
    } else if (status === Issue_Approver_Status.REJECTED) {
      await issueServiceClient.rejectIssue({
        name: issue.value.name,
        comment: comment.value,
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
