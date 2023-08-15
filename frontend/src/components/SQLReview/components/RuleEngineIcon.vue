<template>
  <EngineIcon :engine="engine" />
</template>

<script lang="ts" setup>
import { Engine } from "@/types/proto/v1/common";

defineProps<{
  engine: Engine;
}>();
</script>
