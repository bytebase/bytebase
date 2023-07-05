<template>
  <div class="mt-2 flex flex-col gap-y-4">
    <div class="text-sm">
      {{ issue.name }}
    </div>
    <div class="flex flex-col gap-y-1">
      <p class="textlabel">
        {{ $t("common.comment") }}
        <RequiredStar v-show="props.reviewType === 'SEND_BACK'" />
      </p>
      <AutoHeightTextarea
        v-model:value="comment"
        :placeholder="$t('issue.leave-a-comment')"
        :max-height="160"
        class="w-full"
      />
    </div>
    <div class="py-1 flex justify-end gap-x-3">
      <button class="btn-normal" @click="$emit('cancel')">
        {{ $t("common.cancel") }}
      </button>
      <button
        :class="
          buttonStyle === 'PRIMARY'
            ? 'btn-primary'
            : buttonStyle === 'ERROR'
            ? 'btn-danger'
            : 'btn-normal'
        "
        :disabled="!allowConfirm"
        @click="handleConfirm"
      >
        {{ okText }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Ref, computed, ref } from "vue";

import { useIssueLogic } from "../logic";
import { Issue } from "@/types";
import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import AutoHeightTextarea from "@/components/misc/AutoHeightTextarea.vue";
import RequiredStar from "@/components/RequiredStar.vue";

const props = defineProps<{
  okText: string;
  status: Issue_Approver_Status;
  buttonStyle: "PRIMARY" | "ERROR" | "NORMAL";
  reviewType: "APPROVAL" | "SEND_BACK" | "RE_REQUEST_REVIEW";
}>();

const emit = defineEmits<{
  (event: "cancel"): void;
  (
    event: "confirm",
    params: {
      status: Issue_Approver_Status;
      comment?: string;
    },
    onSuccess: () => void
  ): void;
}>();

const issueContext = useIssueLogic();
const issue = issueContext.issue as Ref<Issue>;
const comment = ref("");

const allowConfirm = computed(() => {
  if (props.reviewType === "SEND_BACK" && comment.value === "") {
    return false;
  }

  return true;
});

const handleConfirm = (e: MouseEvent) => {
  const button = e.target as HTMLElement;
  const { left, top, width, height } = button.getBoundingClientRect();
  const { innerWidth: winWidth, innerHeight: winHeight } = window;
  const onSuccess = () => {
    if (props.status !== Issue_Approver_Status.APPROVED) {
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
      status: props.status,
      comment: comment.value,
    },
    onSuccess
  );
};
</script>
