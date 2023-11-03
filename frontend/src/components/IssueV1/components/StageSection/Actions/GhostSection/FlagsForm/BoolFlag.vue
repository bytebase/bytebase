<template>
  <NRadioGroup
    :value="value"
    :disabled="disabled"
    @update:value="$emit('update:value', $event)"
  >
    <NRadio :value="''">Unset ({{ defaults ? "True" : "False" }}) </NRadio>
    <NRadio :value="'true'">True</NRadio>
    <NRadio :value="'false'">False</NRadio>
  </NRadioGroup>
</template>
<script setup lang="ts">
import { NRadio, NRadioGroup } from "naive-ui";

defineProps<{
  value: "true" | "false" | "";
  defaults: boolean;
  disabled?: boolean;
}>();

defineEmits<{
  (event: "update:value", value: "true" | "false" | ""): void;
}>();
</script>
