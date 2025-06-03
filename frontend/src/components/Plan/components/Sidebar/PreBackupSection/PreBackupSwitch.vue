<template>
  <div class="flex items-center">
    <NTooltip :disabled="!disallowPreBackupMessage">
      <template #trigger>
        <NSwitch
          :value="enabled"
          :disabled="!allowChange"
          @update:value="(on) => handleToggle(on)"
        />
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
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { pushNotification } from "@/store";
import {
  PRE_BACKUP_AVAILABLE_ENGINES,
  usePreBackupSettingContext,
} from "./context";

const { t } = useI18n();
const { enabled, allowChange, databases, toggle } =
  usePreBackupSettingContext();

const disallowPreBackupMessage = computed(() => {
  if (allowChange.value) {
    return undefined;
  }
  if (!databases.value.some((db) => !db.backupAvailable)) {
    return t("database.create-target-database-or-schema-for-backup");
  }
  if (
    databases.value.some(
      (db) => !PRE_BACKUP_AVAILABLE_ENGINES.includes(db.instanceResource.engine)
    )
  ) {
    return false;
  }
  return undefined;
});

const disallowPreBackupLink = computed(() => {
  if (allowChange.value) {
    return undefined;
  }
  if (!databases.value.some((db) => !db.backupAvailable)) {
    return "https://www.bytebase.com/docs/change-database/rollback-data-changes?source=console";
  }
  return undefined;
});

const handleToggle = async (on: boolean) => {
  await toggle(on);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
