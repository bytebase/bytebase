<template>
  <ChatPanel v-if="openAIKey" />
</template>

<script lang="ts" setup>
import { useAIContext } from "../logic";

const [{ default: ChatPanel }] = await Promise.all([import("./ChatPanel.vue")]);
const { openAIKey } = useAIContext();
</script>
