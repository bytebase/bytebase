<template>
  <ChatPanel v-if="aiSetting.enabled" />
</template>

<script lang="ts" setup>
import { useAIContext } from "../logic";

const [{ default: ChatPanel }] = await Promise.all([import("./ChatPanel.vue")]);
const { aiSetting } = useAIContext();
</script>
