<template>
  <div class="relative">
    <TableIcon v-bind="$attrs" class="opacity-80" />
    <GlassesIcon
      class="absolute bottom-0 right-0 w-3/4 h-3/4"
      stroke="rgb(var(--color-accent))"
      fill="white"
      :stroke-width="3.5"
    />
  </div>
</template>
<script setup lang="ts">
import { GlassesIcon } from "lucide-vue-next";
import TableIcon from "~icons/heroicons-outline/table-cells";
</script>
