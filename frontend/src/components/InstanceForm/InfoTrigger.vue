<template>
  <button
    class="inline-flex items-center gap-x-0.5 text-control-light hover:text-accent cursor-pointer ml-1"
    @click.prevent.stop="$emit('click')"
  >
    <InfoIcon class="w-3.5 h-3.5" />
    <span class="text-xs">{{ $t("common.info") }}</span>
  </button>
</template>

<script lang="ts" setup>
import { InfoIcon } from "lucide-vue-next";

defineEmits<{
  click: [];
}>();
</script>
