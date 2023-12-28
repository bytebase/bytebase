<template>
  <div
    v-if="showPreBackupSection"
    class="flex items-center justify-between gap-x-2 min-h-[34px] whitespace-nowrap"
  >
    <div class="textlabel flex items-center">
      {{ $t("task.prior-backup") }}
    </div>
    <div class="flex items-center gap-x-2">
      <PreBackupSwitch />
    </div>
  </div>
</template>

<script lang="ts" setup>
import PreBackupSwitch from "./PreBackupSwitch.vue";
import { usePreBackupContext } from "./common";

const { showPreBackupSection } = usePreBackupContext();
</script>
