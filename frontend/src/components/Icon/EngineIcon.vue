<template>
  <img class="h-4 w-auto mx-auto" :class="customClass" :src="icon" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import { Engine } from "@/types/proto/v1/common";

const props = defineProps<{
  engine: Engine;
  customClass?: string;
}>();

const icon = computed(() => {
  return EngineIconPath[props.engine];
});
</script>
