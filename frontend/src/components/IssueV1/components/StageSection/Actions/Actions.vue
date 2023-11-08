<template>
  <div class="flex flex-col items-end gap-y-2">
    <EarliestAllowedTime />

    <RollbackSection />

    <GhostSection />
  </div>
</template>

<script setup lang="ts">
import EarliestAllowedTime from "./EarliestAllowedTime.vue";
import GhostSection from "./GhostSection";
import RollbackSection from "./RollbackSection";
</script>
