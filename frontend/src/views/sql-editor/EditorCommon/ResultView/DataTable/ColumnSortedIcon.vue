<template>
  <span class="w-4 h-auto opacity-80 ml-1">
    <ArrowUpWideNarrow
      v-if="isSorted === 'desc'"
      class="w-full h-auto text-accent"
    />
    <ArrowDownWideNarrow
      v-else
      :class="[
        'w-full h-auto',
        !isSorted && ' text-gray-300',
        !!isSorted && 'text-accent',
      ]"
    />
  </span>
</template>

<script lang="ts" setup>
import { SortDirection } from "@tanstack/vue-table";
import { ArrowUpWideNarrow, ArrowDownWideNarrow } from "lucide-vue-next";

defineProps<{
  isSorted: false | SortDirection;
}>();
</script>
