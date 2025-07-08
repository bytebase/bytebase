<template>
  <div class="text-sm">
    <HumanizeDate
      v-if="logTime"
      :date="logTime"
      format="YYYY-MM-DD HH:mm:ss.SSS UTCZZ"
    />
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import type { FlattenLogEntry } from "./common";

const props = defineProps<{
  entry: FlattenLogEntry;
}>();

const logTime = computed(() => {
  return props.entry.startTime;
});
</script>
