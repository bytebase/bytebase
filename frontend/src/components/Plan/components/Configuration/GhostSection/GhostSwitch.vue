<template>
  <NTooltip :disabled="!tooltipMessage" :showArrow="false">
    <template #trigger>
      <NSwitch
        size="small"
        :value="enabled"
        :disabled="
          !allowChange ||
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
        :url="'https://docs.bytebase.com/change-database/rollback-data-changes?source=console'"
        color="light"
        class="mt-1 text-sm"
      />
    </div>
  </NTooltip>
</template>

<script setup lang="tsx">
import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { NSwitch, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import type { ErrorItem } from "@/components/misc/ErrorList.vue";
import { default as ErrorList } from "@/components/misc/ErrorList.vue";
import { targetsForSpec } from "@/components/Plan/logic";
import { planServiceClientConnect } from "@/connect";
import { pushNotification } from "@/store";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { allowGhostForDatabase } from "./common";
import { useGhostSettingContext } from "./context";

const { t } = useI18n();
const {
  isCreating,
  plan,
  selectedSpec,
  allowChange,
  enabled,
  events,
  databases,
} = useGhostSettingContext();

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
            .map((db) => db.databaseName)
            .join(", "),
        }
      )
    );
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
      .map((db) => db.databaseName)
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

  if (isCreating.value) {
    if (
      !selectedSpec.value ||
      selectedSpec.value.config?.case !== "changeDatabaseConfig"
    )
      return;
    selectedSpec.value.config.value.enableGhost = on;
  } else {
    const planPatch = cloneDeep(plan.value);
    const spec = (planPatch?.specs || []).find((spec) => {
      return spec.id === selectedSpec.value?.id;
    });
    if (!planPatch || !spec || spec.config?.case !== "changeDatabaseConfig") {
      // Should not reach here.
      throw new Error(
        "Plan or spec is not defined. Cannot update gh-ost setting."
      );
    }

    spec.config.value.enableGhost = on;
    const request = create(UpdatePlanRequestSchema, {
      plan: planPatch,
      updateMask: { paths: ["specs"] },
    });
    await planServiceClientConnect.updatePlan(request);
    events.emit("update");
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
};
</script>
