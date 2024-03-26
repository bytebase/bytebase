<template>
  <router-link
    v-if="userEmail !== SYSTEM_BOT_EMAIL"
    :to="`/users/${userEmail}`"
    class="font-medium text-main whitespace-nowrap hover:underline"
    exact-active-class=""
    >{{ user?.title }}</router-link
  >
  <div v-else class="inline-flex items-center">
    <span class="font-medium text-main whitespace-nowrap">
      {{ user?.title }}
    </span>
    <SystemBotTag />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useUserStore } from "@/store";
import { SYSTEM_BOT_EMAIL } from "@/types";
import type { LogEntity } from "@/types/proto/v1/logging_service";
import { extractUserResourceName } from "@/utils";

const props = defineProps<{
  activity: LogEntity;
}>();

const userEmail = computed(() => {
  return extractUserResourceName(props.activity.creator);
});

const user = computed(() => {
  return useUserStore().getUserByEmail(userEmail.value);
});
</script>
