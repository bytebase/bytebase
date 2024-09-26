<template>
  <span class="w-4 h-auto opacity-80 ml-1 shrink-0">
    <ArrowDownWideNarrow
      v-if="!isSorted || isSorted === 'desc'"
      class="w-full h-auto"
      :class="[isSorted ? 'text-accent' : 'text-gray-300']"
    />
    <ArrowUpWideNarrow v-else class="w-full h-auto text-accent" />
  </span>
</template>

<script lang="ts" setup>
import type { SortDirection } from "@tanstack/vue-table";
import { ArrowUpWideNarrow, ArrowDownWideNarrow } from "lucide-vue-next";

defineProps<{
  isSorted: false | SortDirection;
}>();
</script>
