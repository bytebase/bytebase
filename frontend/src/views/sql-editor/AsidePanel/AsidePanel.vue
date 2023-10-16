<template>
  <div class="aside-panel h-full flex flex-col overflow-hidden">
    <GroupingBar />
    <div class="flex-1 overflow-hidden">
      <DatabaseTree v-model:search-pattern="searchPattern" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import DatabaseTree from "./DatabaseTree.vue";
import GroupingBar from "./GroupingBar";

const searchPattern = ref("");
</script>
