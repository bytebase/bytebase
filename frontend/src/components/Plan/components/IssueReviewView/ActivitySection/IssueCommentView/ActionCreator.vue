<template>
  <div v-if="userEmail === userStore.systemBotUser?.email" class="inline-flex items-center gap-1">
    <span class="font-medium text-main whitespace-nowrap">
      {{ user?.title }}
    </span>
    <SystemBotTag />
  </div>
  <UserLink v-else-if="user" :title="user.title" :email="user.email" />
  <span v-else class="font-medium text-main whitespace-nowrap">-</span>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import { UserLink } from "@/components/v2/Model/cells";
import { extractUserId, useUserStore } from "@/store";

const props = defineProps<{
  // Format: users/{email}
  creator: string;
}>();

const userStore = useUserStore();

const userEmail = computed(() => {
  return extractUserId(props.creator);
});

const user = computedAsync(() => {
  return userStore.getOrFetchUserByIdentifier(props.creator);
});
</script>
