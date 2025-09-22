<template>
  <div class="w-full flex items-center gap-3">
    <div class="flex items-center min-w-24">
      <label class="text-sm text-main">
        {{ $t("task.prior-backup") }}
      </label>
      <NTooltip>
        <template #trigger>
          <heroicons:information-circle
            class="w-4 h-4 text-control-light cursor-help"
          />
        </template>
        <template #default>
          <div class="max-w-xs">
            {{ $t("task.prior-backup-tips") }}
          </div>
        </template>
      </NTooltip>
    </div>
    <PreBackupSwitch />
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import PreBackupSwitch from "./PreBackupSwitch.vue";
</script>
