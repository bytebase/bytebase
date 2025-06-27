<template>
  <NTooltip :disabled="!tooltipMessage" :showArrow="false">
    <template #trigger>
      <NSwitch
        size="small"
        :value="enabled"
        :disabled="!allowChange || errors.length > 0"
        @update:value="(on) => handleToggle(on)"
      />
    </template>
    <div class="max-w-sm">
      <p class="opacity-80">{{ tooltipMessage }}</p>
      <ErrorList v-if="errors.length > 0" :errors="errors" class="mt-2" />
      <LearnMoreLink
        v-if="disallowPreBackupLink && errors.length === 0"
        :url="disallowPreBackupLink"
        color="light"
        class="mt-1 text-sm"
      />
    </div>
  </NTooltip>
</template>

<script lang="ts" setup>
import { NSwitch, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { targetsForSpec } from "@/components/Plan/logic";
import type { ErrorItem } from "@/components/misc/ErrorList.vue";
import ErrorList from "@/components/misc/ErrorList.vue";
import { pushNotification } from "@/store";
import { convertEngineToNew } from "@/utils/v1/common-conversions";
import {
  PRE_BACKUP_AVAILABLE_ENGINES,
  usePreBackupSettingContext,
} from "./context";

const { t } = useI18n();
const { enabled, allowChange, databases, toggle, selectedSpec } =
  usePreBackupSettingContext();

const hasBackupUnavailable = computed(() => {
  return !databases.value.some((db) => !db.backupAvailable);
});

const disallowPreBackupLink = computed(() => {
  if (allowChange.value || !hasBackupUnavailable.value) {
    return undefined;
  }
  return "https://docs.bytebase.com/change-database/rollback-data-changes?source=console";
});

// Compute detailed errors for ErrorList
const errors = computed(() => {
  const errors: ErrorItem[] = [];

  // Check for databases without backup available
  const backupUnavailableDatabases = databases.value.filter(
    (db) => !db.backupAvailable
  );
  if (backupUnavailableDatabases.length > 0) {
    errors.push(
      t("database.backup-unavailable-for-databases", {
        databases: backupUnavailableDatabases
          .map((db) => db.databaseName)
          .join(", "),
      })
    );
  }

  // Check for unsupported database engines
  const unsupportedEngineDatabases = databases.value.filter(
    (db) => !PRE_BACKUP_AVAILABLE_ENGINES.includes(convertEngineToNew(db.instanceResource.engine))
  );
  if (unsupportedEngineDatabases.length > 0) {
    errors.push(
      t("database.engine-not-supported-for-backup", {
        databases: unsupportedEngineDatabases
          .map((db) => `${db.databaseName} (${db.instanceResource.engine})`)
          .join(", "),
      })
    );
  }

  return errors;
});

// Computed properties for new tooltip functionality
const specTargets = computed(() => {
  if (!selectedSpec.value) return [];
  return targetsForSpec(selectedSpec.value);
});

const checkedDatabasesCount = computed(() => {
  return databases.value.length;
});

const totalDatabasesCount = computed(() => {
  return specTargets.value.length;
});

const databasesNotMeetingRequirements = computed(() => {
  return databases.value.filter((db) => {
    // Check if database doesn't have backup available
    if (!db.backupAvailable) return true;
    // Check if database engine is not supported
    if (!PRE_BACKUP_AVAILABLE_ENGINES.includes(convertEngineToNew(db.instanceResource.engine))) {
      return true;
    }
    return false;
  });
});

const tooltipMessage = computed(() => {
  // Check if only some databases are checked
  if (
    checkedDatabasesCount.value < totalDatabasesCount.value &&
    checkedDatabasesCount.value > 0
  ) {
    return t("plan.pre-backup.only-some-databases-checked");
  }

  // Check if some databases don't meet requirements (show summary)
  if (databasesNotMeetingRequirements.value.length > 0) {
    // If there are specific errors, show a general summary
    if (errors.value.length > 0) {
      return t("plan.pre-backup.some-databases-have-issues", {
        count: databasesNotMeetingRequirements.value.length,
      });
    }
    // Otherwise show the databases that don't meet requirements
    const dbNames = databasesNotMeetingRequirements.value
      .map((db) => db.databaseName)
      .join(", ");
    return t("plan.pre-backup.some-databases-not-meeting-requirements", {
      databases: dbNames,
    });
  }

  // If there are errors but all databases are selected, show a general error summary
  if (errors.value.length > 0) {
    return t("plan.pre-backup.configuration-has-issues", {
      count: errors.value.length,
    });
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
