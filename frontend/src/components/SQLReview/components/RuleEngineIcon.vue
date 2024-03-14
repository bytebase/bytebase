<template>
  <NTooltip>
    <template #trigger>
      <EngineIcon :engine="engine" />
    </template>
    <RichEngineName :engine="engine" class="text-white text-xs" />
  </NTooltip>
</template>

<script lang="ts" setup>
import RichEngineName from "@/components/v2/Model/Instance/RichEngineName.vue";
import { Engine } from "@/types/proto/v1/common";

defineProps<{
  engine: Engine;
}>();
</script>
