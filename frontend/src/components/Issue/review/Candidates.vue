<template>
  <NEllipsis
    class="flex-1 truncate"
    :tooltip="{
      raw: true,
      showArrow: false,
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
        class="w-[12rem] max-h-[18rem] bg-white text-control-light py-1 px-2 overflow-auto divide-y"
      >
        <div
          v-for="user in candidates"
          :key="user.name"
          class="py-1"
          :class="[user.name === currentUser.name && 'font-bold']"
        >
          <span class="whitespace-nowrap">{{ user.title }}</span>
          <span v-if="user.name === currentUser.name" class="ml-1">
            ({{ $t("custom-approval.issue-review.you") }})
          </span>
        </div>
      </div>
    </template>
  </NEllipsis>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { NEllipsis } from "naive-ui";

import { useAuthStore } from "@/store";
import { User } from "@/types/proto/v1/auth_service";

const { currentUser } = storeToRefs(useAuthStore());

defineProps<{
  candidates: User[];
}>();
</script>
