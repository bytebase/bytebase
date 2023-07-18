<template>
  <div
    class="flex flex-col lg:flex-row lg:flex-wrap lg:items-center justify-start gap-y-2 gap-x-4 text-sm"
  >
    <StageSelect />
    <DatabaseInfo />
  </div>
</template>

<script setup lang="ts">
import StageSelect from "./StageSelect.vue";
import DatabaseInfo from "./DatabaseInfo.vue";
</script>
