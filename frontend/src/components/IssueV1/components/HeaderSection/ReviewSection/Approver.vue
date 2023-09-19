<template>
  <div
    class="inline-block"
    :class="step.approver?.name === currentUser.name && 'font-bold'"
  >
    <span>{{ step.approver?.title }}</span>
    <span
      v-if="step.approver?.name === currentUser.name"
      class="ml-1 inline-flex items-center px-1 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
    >
      {{ $t("custom-approval.issue-review.you") }}
    </span>
    <span
      v-if="step.approver?.name === SYSTEM_BOT_USER_NAME"
      class="ml-1 inline-flex items-center px-1 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
    >
      {{ $t("settings.members.system-bot") }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { useCurrentUserV1 } from "@/store";
import { SYSTEM_BOT_USER_NAME, WrappedReviewStep } from "@/types";

const currentUser = useCurrentUserV1();

defineProps<{
  step: WrappedReviewStep;
}>();
</script>
