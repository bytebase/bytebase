<template>
  <ChatPanel v-if="!!provider" />
</template>

<script lang="ts" setup>
import { useAIContext } from "../logic";

const [{ default: ChatPanel }] = await Promise.all([import("./ChatPanel.vue")]);
const { provider } = useAIContext();
</script>
