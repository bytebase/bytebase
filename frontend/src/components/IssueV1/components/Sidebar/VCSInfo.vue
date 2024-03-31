<template>
  <div
    v-if="false"
    class="text-sm text-control-light flex flex-col items-start gap-y-1"
  ></div>
</template>

<script setup lang="ts">
import { computed } from "vue";

defineExpose({
  shown: computed(() => {
    return false;
  }),
});
</script>
