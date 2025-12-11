<template>
  <div v-bind="$attrs">
    <!-- This awkward code is author couldn't figure out proper way to use dynamic src under vite
                   https://github.com/vitejs/vite/issues/1265 -->
    <template v-if="type === WebhookType.SLACK">
      <img src="../../assets/im/slack.png" alt="" />
    </template>
    <template v-else-if="type === WebhookType.DISCORD">
      <img src="../../assets/im/discord.svg" />
    </template>
    <template v-else-if="type === WebhookType.TEAMS">
      <img src="../../assets/im/teams.svg" />
    </template>
    <template v-else-if="type === WebhookType.DINGTALK">
      <img src="../../assets/im/dingtalk.png" />
    </template>
    <template v-else-if="type === WebhookType.FEISHU">
      <img src="../../assets/im/feishu.webp" />
    </template>
    <template v-else-if="type === WebhookType.LARK">
      <img src="../../assets/im/feishu.webp" />
    </template>
    <template v-else-if="type === WebhookType.WECOM">
      <img src="../../assets/im/wecom.png" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { WebhookType } from "@/types/proto-es/v1/common_pb";

defineProps<{
  type: WebhookType;
}>();
</script>
