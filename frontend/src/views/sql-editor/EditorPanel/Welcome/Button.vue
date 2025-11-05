<template>
  <NButton
    type="primary"
    style="--n-height: 7rem; --n-padding: 0 6px; min-width: 7rem"
  >
    <div>
      <slot name="icon" />
    </div>
    <div>
      <slot name="default" />
    </div>
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
</script>

<style scoped lang="postcss">
:deep(.n-button__content) {
  display: flex;
  flex-direction: column;
  height: 100%;
  justify-content: center;
  align-items: center;
  gap: 0.5rem;
}
</style>
