<template>
  <div class="p-2">
    <DatabaseOverviewInfo :database="database" />
  </div>
</template>

<script setup lang="ts">
import DatabaseOverviewInfo from "@/components/Database/DatabaseOverviewInfo.vue";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";

const { database } = useConnectionOfCurrentSQLEditorTab();
</script>
