<template>
  <div
    class="text-md font-normal"
    :class="[
      dark ? 'text-[var(--color-matrix-green-hover)]' : 'text-control-light',
    ]"
  >
    {{ error }}
  </div>
</template>

<script lang="ts" setup>
import { useSQLResultViewContext } from "./context";

defineProps<{
  error: string | undefined;
}>();

const { dark } = useSQLResultViewContext();
</script>
