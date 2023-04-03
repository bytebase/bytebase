<template>
  <router-link
    v-if="activity.creator.id !== SYSTEM_BOT_ID"
    :to="`/u/${activity.creator.id}`"
    class="font-medium text-main whitespace-nowrap hover:underline"
    exact-active-class=""
    >{{ activity.creator.name }}</router-link
  >
</template>

<script lang="ts" setup>
import { Activity, SYSTEM_BOT_ID } from "@/types";

defineProps<{
  activity: Activity;
}>();
</script>
