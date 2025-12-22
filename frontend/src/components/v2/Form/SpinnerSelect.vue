<template>
  <NSelect v-bind="$attrs" :loading="loading" @update:value="handleUpdate" />
</template>

<script lang="ts">
import { defineComponent } from "vue";

defineComponent({
  inheritAttrs: false,
});
</script>

<script lang="ts" setup>
import { NSelect, type SelectProps } from "naive-ui";
import { ref } from "vue";

export interface SpinnerSelectProps extends /* @vue-ignore */ SelectProps {
  onUpdate: (value: string | undefined) => Promise<void>;
}
const props = defineProps<SpinnerSelectProps>();

const loading = ref(false);

const handleUpdate = async (value: string | undefined) => {
  if (loading.value) return;

  loading.value = true;
  try {
    await props.onUpdate(value);
  } finally {
    loading.value = false;
  }
};
</script>
