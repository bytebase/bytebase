<template>
  <div class="relative">
    <TableIcon v-bind="$attrs" class="text-gray-400" />
    <GlassesIcon
      class="absolute bottom-0 right-0 w-3/4 h-3/4"
      stroke="rgb(var(--color-accent))"
      fill="white"
      :stroke-width="3.5"
    />
  </div>
</template>
<script setup lang="ts">
import { GlassesIcon, TableIcon } from "lucide-vue-next";
</script>
