<template>
  <router-link
    v-if="userEmail !== SYSTEM_BOT_EMAIL"
    :to="`/u/${userId}`"
    class="font-medium text-main whitespace-nowrap hover:underline"
    exact-active-class=""
    >{{ user?.title }}</router-link
  >
  <div v-else class="inline-flex items-center">
    <span class="font-medium text-main whitespace-nowrap">
      {{ user?.title }}
    </span>
    <span
      class="ml-0.5 inline-flex items-center px-1 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
    >
      {{ $t("settings.members.system-bot") }}
    </span>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { SYSTEM_BOT_EMAIL } from "@/types";
import { LogEntity } from "@/types/proto/v1/logging_service";
import { extractUserResourceName, extractUserUID } from "@/utils";
import { useUserStore } from "@/store";

const props = defineProps<{
  activity: LogEntity;
}>();

const userEmail = computed(() => {
  return extractUserResourceName(props.activity.creator);
});

const user = computed(() => {
  return useUserStore().getUserByEmail(userEmail.value);
});

const userId = computed(() => {
  const username = user.value?.name ?? "";
  return extractUserUID(username);
});
</script>
