<template>
  <NButtonGroup size="tiny" class="bg-white rounded">
    <NButton :disabled="zoom <= min" @click="$emit('zoom-out')">
      <template #icon>
        <heroicons-outline:minus />
      </template>
    </NButton>
    <NButton ghost class="pointer-events-none">
      <span class="w-8 text-xs"> {{ displayZoom }} </span>
    </NButton>
    <NButton :disabled="zoom >= max" @click="$emit('zoom-in')">
      <template #icon>
        <heroicons-outline:plus />
      </template>
    </NButton>
  </NButtonGroup>
</template>

<script lang="ts" setup>
import { NButtonGroup, NButton } from "naive-ui";
import { computed } from "vue";
import { useSchemaDiagramContext } from "../common";

defineProps<{
  min: number;
  max: number;
}>();

defineEmits<{
  (e: "zoom-in"): void;
  (e: "zoom-out"): void;
}>();

const { zoom } = useSchemaDiagramContext();

const displayZoom = computed(() => `${Math.round(zoom.value * 100)}%`);
</script>
