<template>
  <div v-bind="$attrs">
    <!-- This awkward code is author couldn't figure out proper way to use dynamic src under vite
                   https://github.com/vitejs/vite/issues/1265 -->
    <template v-if="type === Webhook_Type.SLACK">
      <img src="../../assets/im/slack.png" alt="" />
    </template>
    <template v-else-if="type === Webhook_Type.DISCORD">
      <img src="../../assets/im/discord.svg" />
    </template>
    <template v-else-if="type === Webhook_Type.TEAMS">
      <img src="../../assets/im/teams.svg" />
    </template>
    <template v-else-if="type === Webhook_Type.DINGTALK">
      <img src="../../assets/im/dingtalk.png" />
    </template>
    <template v-else-if="type === Webhook_Type.FEISHU">
      <img src="../../assets/im/feishu.webp" />
    </template>
    <template v-else-if="type === Webhook_Type.LARK">
      <img src="../../assets/im/feishu.webp" />
    </template>
    <template v-else-if="type === Webhook_Type.WECOM">
      <img src="../../assets/im/wecom.png" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { Webhook_Type } from "@/types/proto-es/v1/project_service_pb";

defineProps<{
  type: Webhook_Type;
}>();
</script>
