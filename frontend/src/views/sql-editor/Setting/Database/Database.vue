<template>
  <div class="w-full h-full flex flex-col gap-4 py-4 px-2 overflow-y-auto">
    <DatabaseDashboard :open-in-new-window="true" />
  </div>
</template>

<script setup lang="ts">
import DatabaseDashboard from "@/views/DatabaseDashboard.vue";
</script>
