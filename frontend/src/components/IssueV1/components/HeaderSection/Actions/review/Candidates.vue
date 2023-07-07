<template>
  <NTooltip v-if="candidates.length === 0" placement="top">
    <template #trigger>
      <heroicons:exclamation-triangle class="w-4 h-4 inline-block" />
    </template>

    <div class="w-[14rem]">
      {{ $t("custom-approval.issue-review.no-one-matches-role") }}
    </div>
  </NTooltip>

  <NEllipsis
    v-else
    class="flex-1 truncate"
    :tooltip="{
      raw: true,
      showArrow: false,
    }"
    :theme-overrides="{
      peers: {
        Tooltip: {
          boxShadow: 'none',
        },
      },
    }"
  >
    <div
      v-for="(user, i) in candidates"
      :key="user.name"
      class="inline-flex flex-nowrap truncate"
    >
      <span
        :class="user.name === currentUser.name && 'font-bold'"
        class="truncate"
      >
        {{ user.title }}
      </span>
      <span v-if="user.name === currentUser.name" class="font-bold ml-1">
        ({{ $t("custom-approval.issue-review.you") }})
      </span>
      <span v-if="i < candidates.length - 1" class="mr-1">,</span>
    </div>

    <template #tooltip>
      <div
        class="w-[12rem] max-h-[18rem] flex flex-col border rounded bg-white shadow-md text-control-light overflow-y-hidden"
      >
        <div class="whitespace-nowrap pt-3 pb-2 px-2 border-b textlabel">
          {{ approvalNodeText(step.step.nodes[0]) }}
        </div>
        <div class="flex-1 overflow-auto text-xs">
          <div
            v-for="user in candidates"
            :key="user.name"
            class="flex items-center py-1.5 px-2"
            :class="[user.name === currentUser.name && 'font-bold']"
          >
            <PrincipalAvatar
              :principal="convertUserToPrincipal(user)"
              size="SMALL"
              class="mr-2"
            />
            <span class="whitespace-nowrap">{{ user.title }}</span>
            <span v-if="user.name === currentUser.name" class="ml-1">
              ({{ $t("custom-approval.issue-review.you") }})
            </span>
          </div>
        </div>
      </div>
    </template>
  </NEllipsis>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { storeToRefs } from "pinia";
import { NEllipsis, NTooltip } from "naive-ui";

import { convertUserToPrincipal, useAuthStore } from "@/store";
import { WrappedReviewStep } from "@/types";
import { approvalNodeText } from "@/utils";
import PrincipalAvatar from "@/components/PrincipalAvatar.vue";

const { currentUser } = storeToRefs(useAuthStore());

const props = defineProps<{
  step: WrappedReviewStep;
}>();

const candidates = computed(() => props.step.candidates);
</script>
