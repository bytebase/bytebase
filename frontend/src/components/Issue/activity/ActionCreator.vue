<template>
  <router-link
    v-if="activity.creator.id !== SYSTEM_BOT_ID"
    :to="`/u/${activity.creator.id}`"
    class="font-medium text-main whitespace-nowrap hover:underline"
    exact-active-class=""
    >{{ activity.creator.name }}</router-link
  >
  <div v-else class="inline-flex items-center">
    <span class="font-medium text-main whitespace-nowrap">
      {{ activity.creator.name }}
    </span>
    <span
      class="ml-0.5 inline-flex items-center px-1 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
    >
      {{ $t("settings.members.system-bot") }}
    </span>
  </div>
</template>

<script lang="ts" setup>
import { Activity, SYSTEM_BOT_ID } from "@/types";

defineProps<{
  activity: Activity;
}>();
</script>
