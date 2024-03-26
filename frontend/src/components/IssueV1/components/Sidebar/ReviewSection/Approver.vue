<template>
  <div :class="step.approver?.name === currentUser.name && 'font-bold'">
    <slot name="title" :approver="step.approver">
      <span class="truncate">
        <NPerformantEllipsis>
          {{ step.approver?.title }}
        </NPerformantEllipsis>
      </span>
    </slot>
    <span
      v-if="step.approver?.name === currentUser.name"
      class="ml-1 px-1 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
    >
      {{ $t("custom-approval.issue-review.you") }}
    </span>
    <SystemBotTag v-if="step.approver?.name === SYSTEM_BOT_USER_NAME" />
  </div>
</template>

<script setup lang="ts">
import { NPerformantEllipsis } from "naive-ui";
import { useCurrentUserV1 } from "@/store";
import type { WrappedReviewStep } from "@/types";
import { SYSTEM_BOT_USER_NAME } from "@/types";

const currentUser = useCurrentUserV1();

defineProps<{
  step: WrappedReviewStep;
}>();
</script>
