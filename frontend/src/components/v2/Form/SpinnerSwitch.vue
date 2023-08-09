<template>
  <NSwitch :loading="loading" @update:value="handleToggle" />
</template>

<script lang="ts" setup>
import { NSwitch } from "naive-ui";
import { ref } from "vue";

const props = defineProps<{
  onToggle: (on: boolean) => Promise<any>;
}>();

const loading = ref(false);

const handleToggle = async (on: boolean) => {
  if (loading.value) return;

  loading.value = true;
  try {
    await props.onToggle(on);
  } finally {
    loading.value = false;
  }
};
</script>
