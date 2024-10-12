<template>
  <div
    v-if="showPreBackupSection"
    class="w-full flex items-center justify-between gap-x-2 gap-y-1 flex-wrap min-h-[34px] whitespace-nowrap"
  >
    <NTooltip>
      <template #trigger>
        <div class="textlabel flex items-center">
          <span>{{ $t("task.prior-backup") }}</span>
          <span
            v-if="showRollbackSection"
            class="ml-1 px-1 py-0.5 rounded-lg text-xs bg-green-100 text-green-800"
          >
            {{ $t("common.on") }}
          </span>
        </div>
      </template>
      <template #default>
        <div class="max-w-[20rem]">
          {{ $t("task.prior-backup-tips") }}
        </div>
      </template>
    </NTooltip>
    <div class="flex items-center">
      <TaskRollbackButton v-if="showRollbackSection" />
      <PreBackupSwitch v-else />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import PreBackupSwitch from "./PreBackupSwitch.vue";
import TaskRollbackButton from "./TaskRollbackButton.vue";
import { usePreBackupContext } from "./common";

const { showPreBackupSection, showRollbackSection } = usePreBackupContext();
</script>
