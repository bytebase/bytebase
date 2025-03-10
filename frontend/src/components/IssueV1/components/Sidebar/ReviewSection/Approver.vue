<template>
  <div :class="approver?.name === currentUser.name && 'font-bold'">
    <slot name="title" :approver="approver">
      <span class="truncate">
        <NPerformantEllipsis>
          {{ approver?.title }}
        </NPerformantEllipsis>
      </span>
    </slot>
    <span
      v-if="approver?.name === currentUser.name"
      class="ml-1 px-1 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
    >
      {{ $t("custom-approval.issue-review.you") }}
    </span>
    <SystemBotTag v-if="approver?.name === SYSTEM_BOT_USER_NAME" />
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { NPerformantEllipsis } from "naive-ui";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import { useCurrentUserV1, useUserStore } from "@/store";
import type { WrappedReviewStep } from "@/types";
import { SYSTEM_BOT_USER_NAME } from "@/types";

const currentUser = useCurrentUserV1();
const userStore = useUserStore();

const props = defineProps<{
  step: WrappedReviewStep;
}>();

const approver = computedAsync(async () => {
  if (!props.step.approver) {
    return;
  }
  const user = await userStore.getOrFetchUserByIdentifier(props.step.approver);
  return user;
});
</script>
