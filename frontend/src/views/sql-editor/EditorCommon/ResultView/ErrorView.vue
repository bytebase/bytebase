<template>
  <div
    class="text-md font-normal flex items-center gap-x-3"
    :class="[dark ? 'text-matrix-green-hover' : 'text-control-light']"
  >
    <span>{{ error }}</span>
    <slot name="suffix" />
  </div>
</template>

<script lang="ts" setup>
import { useSQLResultViewContext } from "./context";

defineProps<{
  error: string | undefined;
}>();

const { dark } = useSQLResultViewContext();
</script>
