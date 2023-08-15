<template>
  <EngineIcon :engine="engineFromJSON(engine)" />
</template>

<script lang="ts" setup>
import { Engine, engineFromJSON } from "@/types/proto/v1/common";

defineProps<{
  engine: Engine;
}>();
</script>
