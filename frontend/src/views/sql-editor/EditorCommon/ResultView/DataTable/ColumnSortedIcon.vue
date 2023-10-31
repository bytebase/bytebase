<template>
  <template v-if="isSorted">
    <span class="w-4 h-auto opacity-80 ml-0.5">
      <ArrowDownWideNarrow v-if="isSorted === 'asc'" class="w-full h-auto" />
      <ArrowUpWideNarrow v-else class="w-full h-auto" />
    </span>
  </template>
</template>

<script lang="ts" setup>
import { SortDirection } from "@tanstack/table-core";
import { ArrowUpWideNarrow, ArrowDownWideNarrow } from "lucide-vue-next";

defineProps<{
  isSorted: false | SortDirection;
}>();
</script>
