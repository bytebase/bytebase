<template>
  <Suspense>
    <MonacoEditor v-bind="($attrs as any)" />
  </Suspense>
</template>

<script setup lang="ts">
import MonacoEditor from "./MonacoEditor.vue";
</script>
