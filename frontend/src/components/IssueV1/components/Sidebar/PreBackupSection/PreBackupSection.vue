<template>
  <div
    v-if="showPreBackupSection"
    class="flex items-center justify-between gap-x-2 min-h-[34px] whitespace-nowrap"
  >
    <NTooltip>
      <template #trigger>
        <div class="textlabel flex items-center">
          {{ $t("task.prior-backup") }}
          <BBBetaBadge class="scale-75" />
        </div>
      </template>
      <template #default>
        <div class="max-w-[20rem]">
          {{ $t("task.prior-backup-tips") }}
        </div>
      </template>
    </NTooltip>
    <div class="flex items-center gap-x-2">
      <PreBackupSwitch />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { BBBetaBadge } from "@/bbkit";
import PreBackupSwitch from "./PreBackupSwitch.vue";
import { usePreBackupContext } from "./common";

const { showPreBackupSection } = usePreBackupContext();
</script>
