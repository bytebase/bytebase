<template>
  <NSwitch
    class="bb-switch"
    style="
      --n-width: max(
        var(--n-rail-width),
        calc(var(--n-rail-width) + var(--n-button-width) - var(--n-rail-height))
      );
    "
  >
    <template v-if="text" #checked>
      <span style="font-size: 10px">{{ $t("common.on") }}</span>
    </template>
    <template v-if="text" #unchecked>
      <span style="font-size: 10px">{{ $t("common.off") }}</span>
    </template>
  </NSwitch>
</template>

<script lang="ts" setup>
import { NSwitch } from "naive-ui";

defineProps<{
  text?: boolean;
}>();
</script>

<style lang="postcss" scoped>
.bb-switch :deep(.n-switch__checked) {
  padding-right: calc(var(--n-rail-height) - var(--n-offset) + 1px) !important;
}
.bb-switch :deep(.n-switch__unchecked) {
  padding-left: calc(var(--n-rail-height) - var(--n-offset) + 1px) !important;
}
.bb-switch :deep(.n-switch__button-placeholder) {
  width: calc(1.25 * var(--n-rail-height)) !important;
}
</style>
