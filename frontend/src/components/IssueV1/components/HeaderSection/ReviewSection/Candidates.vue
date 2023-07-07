<template>
  <div>
    <NTooltip v-if="candidates.length === 0" placement="top">
      <template #trigger>
        <heroicons:exclamation-triangle class="w-4 h-4 inline-block" />
      </template>

      <div class="w-[14rem]">
        {{ $t("custom-approval.issue-review.no-one-matches-role") }}
      </div>
    </NTooltip>

    <div
      v-else
      class="min-w-[8rem] max-w-[12rem] max-h-[18rem] flex flex-col text-control-light overflow-y-hidden"
    >
      <div class="flex-1 overflow-auto text-xs">
        <div
          v-for="user in candidates"
          :key="user.name"
          class="flex items-center py-1 gap-x-1"
          :class="[user.name === currentUser.name && 'font-bold']"
        >
          <PrincipalAvatar
            :principal="convertUserToPrincipal(user)"
            size="SMALL"
          />
          <span class="whitespace-nowrap">{{ user.title }}</span>
          <span v-if="user.name === currentUser.name">
            ({{ $t("custom-approval.issue-review.you") }})
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { storeToRefs } from "pinia";
import { NTooltip } from "naive-ui";

import { convertUserToPrincipal, useAuthStore } from "@/store";
import { WrappedReviewStep } from "@/types";
import PrincipalAvatar from "@/components/PrincipalAvatar.vue";

const { currentUser } = storeToRefs(useAuthStore());

const props = defineProps<{
  step: WrappedReviewStep;
}>();

const candidates = computed(() => props.step.candidates);
</script>
