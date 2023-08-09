<template>
  <NSwitch :loading="loading" @update:value="handleToggle" />
</template>

<script lang="ts">
import { defineComponent } from "vue";

defineComponent({
  inheritAttrs: false,
});
</script>

<script lang="ts" setup>
import { ref } from "vue";
import { type SwitchProps, NSwitch } from "naive-ui";

export interface SpinnerSwitchProps extends SwitchProps {
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
