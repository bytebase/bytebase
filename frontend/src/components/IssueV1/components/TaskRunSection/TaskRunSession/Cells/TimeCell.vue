<template>
  <div class="text-sm">
    <HumanizeDate
      v-if="date"
      :date="date"
      format="YYYY-MM-DD HH:mm:ss.SSS UTCZZ"
    />
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import HumanizeDate from "@/components/misc/HumanizeDate.vue";

defineProps<{
  date: Date | undefined;
}>();
</script>
