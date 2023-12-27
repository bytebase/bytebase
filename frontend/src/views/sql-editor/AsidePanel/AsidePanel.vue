<template>
  <div class="aside-panel h-full flex flex-col overflow-hidden">
    <GroupingBar />
    <div class="flex-1 overflow-hidden">
      <DatabaseTree />
    </div>
  </div>
</template>

<script lang="ts" setup>
import DatabaseTree from "./DatabaseTree.vue";
import GroupingBar from "./GroupingBar";
</script>
