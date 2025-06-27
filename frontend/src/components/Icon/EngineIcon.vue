<template>
  <img
    :class="twMerge('h-4 w-auto mx-auto object-contain', customClass)"
    :src="icon"
  />
</template>

<script lang="ts" setup>
import { twMerge } from "tailwind-merge";
import { computed } from "vue";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import type { Engine } from "@/types/proto-es/v1/common_pb";

const props = defineProps<{
  engine: Engine;
  customClass?: string;
}>();

const icon = computed(() => {
  return EngineIconPath[props.engine];
});
</script>
