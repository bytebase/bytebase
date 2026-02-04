<template>
  <NTooltip :disabled="!tooltipMessage" :showArrow="false">
    <template #trigger>
      <NSwitch
        size="small"
        :value="enabled"
        :disabled="
          !allowChange ||
          isSheetOversize ||
          (!enabled &&
            (databasesNotMeetingRequirements.length > 0 || errors.length > 0))
        "
        @update:value="toggleChecked"
      />
    </template>
    <div class="max-w-sm">
      <p class="opacity-80">{{ tooltipMessage }}</p>
      <ErrorList v-if="errors.length > 0" :errors="errors" class="mt-2" />
      <LearnMoreLink
        v-if="databasesNotMeetingRequirements.length > 0"
        :url="'https://docs.bytebase.com/change-database/online-schema-migration-for-mysql?source=console'"
        color="light"
        class="mt-1 text-sm"
      />
    </div>
  </NTooltip>
</template>

<script setup lang="tsx">
import { NSwitch, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import type { ErrorItem } from "@/components/misc/ErrorList.vue";
import { default as ErrorList } from "@/components/misc/ErrorList.vue";
import {
  targetsForSpec,
  updateSpecSheetWithStatement,
} from "@/components/Plan/logic";
import { pushNotification } from "@/store";
import {
  extractDatabaseResourceName,
  getInstanceResource,
  setSheetStatement,
} from "@/utils";
import { useSelectedSpec } from "../../SpecDetailView/context";
import {
  getDefaultGhostConfig,
  getGhostConfig,
  isGhostEnabled,
  updateGhostConfig,
} from "../../StatementSection/directiveUtils";
import { useSpecSheet } from "../../StatementSection/useSpecSheet";
import { allowGhostForDatabase } from "./common";
import { useGhostSettingContext } from "./context";

const { t } = useI18n();
const { isCreating, plan, allowChange, events, databases } =
  useGhostSettingContext();
const { selectedSpec } = useSelectedSpec();
const { sheet, sheetStatement, sheetReady, isSheetOversize } =
  useSpecSheet(selectedSpec);

const enabled = computed(() => {
  if (!sheetReady.value) return false;
  return isGhostEnabled(sheetStatement.value);
});

const errors = computed(() => {
  const errors: ErrorItem[] = [];
  const unsupportedDatabases = databases.value.filter(
    (db) => !allowGhostForDatabase(db)
  );
  if (unsupportedDatabases.length > 0) {
    errors.push(
      t(
        "task.online-migration.error.not-applicable.database-doesnt-meet-ghost-requirement",
        {
          database: unsupportedDatabases
            .map((db) => extractDatabaseResourceName(db.name).databaseName)
            .join(", "),
        }
      )
    );
  }

  // Check for databases with supported engine/version but missing backup location (bbdataarchive)
  const missingBackupDatabases = databases.value.filter(
    (db) => allowGhostForDatabase(db) && !db.backupAvailable
  );
  if (missingBackupDatabases.length > 0) {
    // Ghost only supports MySQL/MariaDB, group by instance to avoid duplicates
    const instanceMap = new Map<string, string>();
    for (const db of missingBackupDatabases) {
      const instanceResource = getInstanceResource(db);
      if (!instanceMap.has(instanceResource.name)) {
        instanceMap.set(instanceResource.name, instanceResource.title);
      }
    }
    for (const [, instanceTitle] of instanceMap) {
      errors.push(
        t("plan.ghost.missing-backup-database", {
          instance: instanceTitle,
        })
      );
    }
  }

  return errors;
});

// Computed properties for tooltip functionality
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
    // Check if database doesn't have backup available or doesn't allow ghost
    if (!db.backupAvailable || !allowGhostForDatabase(db)) return true;
    return false;
  });
});

const tooltipMessage = computed(() => {
  // Check if sheet is oversized
  if (isSheetOversize.value) {
    return t("issue.options-disabled-due-to-oversize");
  }

  // Check if only some databases are checked
  if (
    checkedDatabasesCount.value < totalDatabasesCount.value &&
    checkedDatabasesCount.value > 0
  ) {
    return t("plan.ghost.only-some-databases-checked");
  }

  // Check if some databases don't meet requirements (show summary)
  if (databasesNotMeetingRequirements.value.length > 0) {
    // If there are specific errors, show a general summary
    if (errors.value.length > 0) {
      return t("plan.ghost.some-databases-have-issues", {
        count: databasesNotMeetingRequirements.value.length,
      });
    }
    // Otherwise show the databases that don't meet requirements
    const dbNames = databasesNotMeetingRequirements.value
      .map((db) => extractDatabaseResourceName(db.name).databaseName)
      .join(", ");
    return t("plan.ghost.some-databases-not-meeting-requirements", {
      databases: dbNames,
    });
  }

  // If there are errors but all databases are selected, show a general error summary
  if (errors.value.length > 0) {
    return t("plan.ghost.configuration-has-issues", {
      count: errors.value.length,
    });
  }

  return undefined;
});

const toggleChecked = async (on: boolean) => {
  if (errors.value.length > 0) {
    return;
  }

  // Get current ghost config from sheet (to preserve flags when enabling)
  const currentConfig = on
    ? (getGhostConfig(sheetStatement.value) ?? getDefaultGhostConfig())
    : undefined;
  const updatedStatement = updateGhostConfig(
    sheetStatement.value,
    currentConfig
  );

  if (isCreating.value) {
    // When creating a plan, update the local sheet directly.
    if (!sheet.value) return;
    setSheetStatement(sheet.value, updatedStatement);
  } else {
    // For created plans, create new sheet and update plan/spec
    await updateSpecSheetWithStatement(
      plan.value,
      selectedSpec.value,
      updatedStatement
    );
    events.emit("update");
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
};
</script>
