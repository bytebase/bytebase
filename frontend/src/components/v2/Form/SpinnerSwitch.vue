<template>
  <NSwitch :loading="loading" @update:value="handleToggle" />
</template>

<script lang="ts" setup>
import { NSwitch, type SwitchProps } from "naive-ui";
import { ref } from "vue";

export interface SpinnerSwitchProps extends /* @vue-ignore */ SwitchProps {
  onToggle: (on: boolean) => Promise<void>;
}
const props = defineProps<SpinnerSwitchProps>();

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
