<template>
  <div class="menu-item">
    <div class="flex flex-row items-center gap-x-4 justify-between">
      <span>{{ label }}</span>
      <NSwitch size="small" :value="value" @update:value="handleUpdate" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { NSwitch } from "naive-ui";

defineProps<{
  label: string;
  value: boolean;
}>();

const emit = defineEmits<{
  "update:value": [value: boolean];
}>();

const handleUpdate = (val: boolean) => {
  emit("update:value", val);
};
</script>
