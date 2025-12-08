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
import { ArrowDownWideNarrow, ArrowUpWideNarrow } from "lucide-vue-next";
import type { SortDirection } from "../common/types";

defineProps<{
  isSorted: SortDirection;
}>();
</script>
