<template>
  <Suspense>
    <DiffEditor v-bind="($attrs as any)" />
  </Suspense>
</template>

<script setup lang="ts">
import DiffEditor from "./DiffEditor.vue";
</script>
