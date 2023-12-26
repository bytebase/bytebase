<template>
  <div
    v-if="showPreBackupSection"
    class="flex items-center gap-x-3 min-h-[34px]"
  >
    <NTooltip>
      <template #trigger>
        <div class="textlabel flex items-center">
          {{ $t("task.prior-backup") }}
        </div>
      </template>
    </NTooltip>
    <div class="w-[12rem] flex items-center gap-x-2">
      <PreBackupSwitch />
    </div>
  </div>
</template>

<script lang="ts" setup>
import PreBackupSwitch from "./PreBackupSwitch.vue";
import { usePreBackupContext } from "./common";

const { showPreBackupSection } = usePreBackupContext();
</script>
