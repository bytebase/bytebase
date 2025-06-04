<template>
  <NSwitch :loading="loading" @update:value="handleToggle" />
</template>

<script lang="ts" setup>
import { type SwitchProps, NSwitch } from "naive-ui";
import { ref } from "vue";

export interface SpinnerSwitchProps extends /* @vue-ignore */ SwitchProps {
  onToggle: (on: boolean) => Promise<any>;
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
