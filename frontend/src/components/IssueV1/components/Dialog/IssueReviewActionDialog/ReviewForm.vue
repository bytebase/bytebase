<template>
  <div class="mt-2 flex flex-col gap-y-4">
    <div class="text-sm">
      {{ issue.title }}
    </div>
    <div class="flex flex-col gap-y-1">
      <p class="textlabel">
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
    <div class="py-1 flex justify-end gap-x-3">
      <NButton @click="$emit('cancel')">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton
        :disabled="!allowConfirm"
        v-bind="issueReviewActionButtonProps(action)"
        @click="handleConfirm"
      >
        {{ issueReviewActionDisplayName(action) }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { NButton, NInput } from "naive-ui";

import RequiredStar from "@/components/RequiredStar.vue";
import {
  IssueReviewAction,
  issueReviewActionButtonProps,
  issueReviewActionDisplayName,
  useIssueContext,
} from "@/components/IssueV1";

const props = defineProps<{
  action: IssueReviewAction;
}>();

const emit = defineEmits<{
  (event: "cancel"): void;
  (
    event: "confirm",
    params: {
      action: IssueReviewAction;
      comment?: string;
    },
    onSuccess: () => void
  ): void;
}>();

const { issue } = useIssueContext();
const comment = ref("");

const allowConfirm = computed(() => {
  if (props.action === "SEND_BACK" && comment.value === "") {
    return false;
  }

  return true;
});

const handleConfirm = (e: MouseEvent) => {
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

  emit(
    "confirm",
    {
      action: props.action,
      comment: comment.value,
    },
    onSuccess
  );
};
</script>
