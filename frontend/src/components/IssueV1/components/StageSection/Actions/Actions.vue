<template>
  <div class="flex flex-col items-end gap-y-2">
    <EarliestAllowedTime />

    <RollbackSection />
  </div>
</template>

<script setup lang="ts">
import EarliestAllowedTime from "./EarliestAllowedTime.vue";
import RollbackSection from "./RollbackSection";
</script>
