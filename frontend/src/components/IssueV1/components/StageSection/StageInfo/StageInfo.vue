<template>
  <div
    class="flex flex-col lg:flex-row lg:flex-wrap lg:items-center justify-start gap-y-2 gap-x-4 text-sm"
  >
    <StageSelect />
    <DatabaseInfo :link="!disallowNavigateToConsole" />
  </div>
</template>

<script setup lang="ts">
import { useAppFeature } from "@/store";
import DatabaseInfo from "./DatabaseInfo.vue";
import StageSelect from "./StageSelect.vue";

const disallowNavigateToConsole = useAppFeature(
  "bb.feature.disallow-navigate-to-console"
);
</script>
