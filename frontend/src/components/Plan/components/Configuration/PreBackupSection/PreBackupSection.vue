<template>
  <div
    class="w-full flex items-center justify-start gap-x-2 flex-wrap min-h-[34px] whitespace-nowrap"
  >
    <PreBackupSwitch />
    <NTooltip>
      <template #trigger>
        <div class="textlabel flex items-center">
          <span>{{ $t("task.prior-backup") }}</span>
        </div>
      </template>
      <template #default>
        <div class="max-w-[20rem]">
          {{ $t("task.prior-backup-tips") }}
        </div>
      </template>
    </NTooltip>
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import PreBackupSwitch from "./PreBackupSwitch.vue";
</script>
