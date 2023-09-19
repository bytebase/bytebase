<template>
  <div class="flex items-center">
    <NSwitch
      :disabled="!allowRollback"
      :value="rollbackEnabled"
      class="bb-rollback-switch"
      @update:value="toggleRollback"
    >
      <template #checked>
        <span style="font-size: 10px">{{ $t("common.on") }}</span>
      </template>
      <template #unchecked>
        <span style="font-size: 10px">{{ $t("common.off") }}</span>
      </template>
    </NSwitch>
  </div>
</template>

<script lang="ts" setup>
import { NSwitch } from "naive-ui";
import { useRollbackContext } from "./common";

const { allowRollback, rollbackEnabled, toggleRollback } = useRollbackContext();
</script>

<style>
.bb-rollback-switch {
  --n-width: max(
    var(--n-rail-width),
    calc(var(--n-rail-width) + var(--n-button-width) - var(--n-rail-height))
  ) !important;
}
.bb-rollback-switch .n-switch__checked {
  padding-right: calc(var(--n-rail-height) - var(--n-offset) + 1px) !important;
}
.bb-rollback-switch .n-switch__unchecked {
  padding-left: calc(var(--n-rail-height) - var(--n-offset) + 1px) !important;
}
.bb-rollback-switch .n-switch__button-placeholder {
  width: calc(1.25 * var(--n-rail-height)) !important;
}
</style>
