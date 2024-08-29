<template>
  <div class="flex items-center">
    <NTooltip :disabled="!disallowPreBackupMessage">
      <template #trigger>
        <NSwitch
          :value="preBackupEnabled"
          class="bb-pre-backup-switch"
          :disabled="!allowPreBackup"
          @update:value="togglePreBackup"
        >
          <template #checked>
            <span style="font-size: 10px">{{ $t("common.on") }}</span>
          </template>
          <template #unchecked>
            <span style="font-size: 10px">{{ $t("common.off") }}</span>
          </template>
        </NSwitch>
      </template>
      <template #default>
        {{ disallowPreBackupMessage }}
        <LearnMoreLink
          v-if="disallowPreBackupLink"
          :url="disallowPreBackupLink"
          color="light"
          class="ml-1 text-sm"
        />
      </template>
    </NTooltip>
  </div>
</template>

<script lang="ts" setup>
import { NSwitch, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { databaseForTask, useIssueContext } from "@/components/IssueV1/logic";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { usePreBackupContext } from "./common";

const { t } = useI18n();
const { issue, selectedTask } = useIssueContext();
const { preBackupEnabled, allowPreBackup, togglePreBackup } =
  usePreBackupContext();

const database = computed(() =>
  databaseForTask(issue.value, selectedTask.value)
);

const disallowPreBackupMessage = computed(() => {
  if (allowPreBackup.value) {
    return undefined;
  }
  if (!database.value.backupAvailable) {
    return t("database.create-target-database-or-schema-for-backup");
  }
  return undefined;
});

const disallowPreBackupLink = computed(() => {
  if (allowPreBackup.value) {
    return undefined;
  }
  if (!database.value.backupAvailable) {
    return "https://www.bytebase.com/docs/change-database/rollback-data-changes?source=console";
  }
  return undefined;
});
</script>

<style>
.bb-pre-backup-switch {
  --n-width: max(
    var(--n-rail-width),
    calc(var(--n-rail-width) + var(--n-button-width) - var(--n-rail-height))
  ) !important;
}
.bb-pre-backup-switch .n-switch__checked {
  padding-right: calc(var(--n-rail-height) - var(--n-offset) + 1px) !important;
}
.bb-pre-backup-switch .n-switch__unchecked {
  padding-left: calc(var(--n-rail-height) - var(--n-offset) + 1px) !important;
}
.bb-pre-backup-switch .n-switch__button-placeholder {
  width: calc(1.25 * var(--n-rail-height)) !important;
}
</style>
