<template>
  <div
    class="flex flex-col lg:flex-row lg:flex-wrap lg:items-center justify-start gap-y-2 gap-x-4 text-sm"
  >
    <StageSelect />
    <DatabaseInfo :link="pageMode === 'BUNDLED'" />
  </div>
</template>

<script setup lang="ts">
import { usePageMode } from "@/store";
import DatabaseInfo from "./DatabaseInfo.vue";
import StageSelect from "./StageSelect.vue";

const pageMode = usePageMode();
</script>
