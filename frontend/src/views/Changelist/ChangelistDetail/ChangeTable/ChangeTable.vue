<template>
  <div class="flex flex-col gap-y-4">
    <h1>ha ha change table</h1>
    <pre v-for="change in changes" :key="change.sheet" class="text-xs">
      {{ change }}
    </pre>
  </div>
</template>

<script setup lang="ts">
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";

defineProps<{
  changes: Change[];
  reorderMode: boolean;
}>();

defineEmits<{
  // TBD
}>();
</script>
