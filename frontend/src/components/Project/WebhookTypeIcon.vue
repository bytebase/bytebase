<template>
  <div v-bind="$attrs">
    <!-- This awkward code is author couldn't figure out proper way to use dynamic src under vite
                   https://github.com/vitejs/vite/issues/1265 -->
    <template v-if="type === Webhook_Type.TYPE_SLACK">
      <img src="../../assets/slack-logo.png" alt="" />
    </template>
    <template v-else-if="type === Webhook_Type.TYPE_DISCORD">
      <img src="../../assets/discord-logo.svg" />
    </template>
    <template v-else-if="type === Webhook_Type.TYPE_TEAMS">
      <img src="../../assets/teams-logo.svg" />
    </template>
    <template v-else-if="type === Webhook_Type.TYPE_DINGTALK">
      <img src="../../assets/dingtalk-logo.png" />
    </template>
    <template v-else-if="type === Webhook_Type.TYPE_FEISHU">
      <img src="../../assets/feishu-logo.webp" />
    </template>
    <template v-else-if="type === Webhook_Type.TYPE_WECOM">
      <img src="../../assets/wecom-logo.png" />
    </template>
    <template v-else-if="type === Webhook_Type.TYPE_CUSTOM">
      <heroicons-outline:puzzle v-bind="$attrs" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { Webhook_Type } from "@/types/proto/v1/project_service";

defineProps<{
  type: Webhook_Type;
}>();
</script>
