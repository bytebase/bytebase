<template>
  <SchemaPanel @alter-schema="$emit('alter-schema', $event)" />
</template>

<script setup lang="ts">
import SchemaPanel from "./SchemaPanel";

defineEmits<{
  (
    event: "alter-schema",
    params: { databaseId: string; schema: string; table: string }
  ): void;
}>();
</script>
