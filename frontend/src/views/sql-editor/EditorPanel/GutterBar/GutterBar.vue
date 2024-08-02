<template>
  <div
    class="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1 shrink-0"
  >
    <div class="flex flex-col gap-y-1">
      <TabItem view="CODE" />
      <TabItem view="INFO" />
      <TabItem view="TABLES" />
      <TabItem view="VIEWS" />
      <TabItem view="FUNCTIONS" />
      <TabItem view="PROCEDURES" />
    </div>
  </div>
</template>

<script setup lang="ts">
import TabItem from "./TabItem.vue";
</script>
