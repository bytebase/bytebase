<template>
  <NTooltip :disabled="errors.length === 0">
    <template #trigger>
      <NSwitch
        :value="enabled"
        :disabled="!allowChange"
        @update:value="toggleChecked"
      />
    </template>
    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script setup lang="tsx">
import { cloneDeep } from "lodash-es";
import { NSwitch, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { ErrorItem } from "@/components/misc/ErrorList.vue";
import { default as ErrorList } from "@/components/misc/ErrorList.vue";
import { planServiceClient } from "@/grpcweb";
import { pushNotification } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { Plan_ChangeDatabaseConfig_Type } from "@/types/proto/v1/plan_service";
import { engineNameV1 } from "@/utils";
import {
  allowGhostForDatabase,
  MIN_GHOST_SUPPORT_MARIADB_VERSION,
  MIN_GHOST_SUPPORT_MYSQL_VERSION,
} from "./common";
import { useGhostSettingContext } from "./context";

const { t } = useI18n();
const {
  isCreating,
  plan,
  selectedSpec,
  allowChange,
  enabled,
  database,
  events,
} = useGhostSettingContext();

const errors = computed(() => {
  const errors: ErrorItem[] = [];
  if (!database.value.instanceResource.activation) {
    errors.push(
      t("subscription.instance-assignment.missing-license-attention")
    );
  }
  // As we use the same database from backup to save temp tables in gh-ost, check if backup is available.
  if (!database.value.backupAvailable) {
    errors.push(
      t(
        "task.online-migration.error.not-applicable.needs-database-for-saving-temp-data",
        {
          // The same database name as backup.
          database: "bbdataarchive",
        }
      )
    );
  }
  if (!allowGhostForDatabase(database.value)) {
    errors.push(
      t(
        "task.online-migration.error.not-applicable.task-doesnt-meet-ghost-requirement"
      )
    );
    errors.push({
      error: `${engineNameV1(Engine.MYSQL)} >= ${MIN_GHOST_SUPPORT_MYSQL_VERSION}, ${engineNameV1(Engine.MARIADB)} >= ${MIN_GHOST_SUPPORT_MARIADB_VERSION}`,
      indent: 1,
    });
  }
  return errors;
});

const toggleChecked = async (on: boolean) => {
  if (errors.value.length > 0) {
    return;
  }

  if (isCreating.value) {
    if (!selectedSpec.value || !selectedSpec.value.changeDatabaseConfig) return;
    selectedSpec.value.changeDatabaseConfig.type = on
      ? Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST
      : Plan_ChangeDatabaseConfig_Type.MIGRATE;
  } else {
    const planPatch = cloneDeep(plan.value);
    const spec = (planPatch?.specs || []).find((spec) => {
      return spec.id === selectedSpec.value?.id;
    });
    if (!planPatch || !spec || !spec.changeDatabaseConfig) {
      // Should not reach here.
      throw new Error(
        "Plan or spec is not defined. Cannot update gh-ost setting."
      );
    }

    spec.changeDatabaseConfig.type = on
      ? Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST
      : Plan_ChangeDatabaseConfig_Type.MIGRATE;
    await planServiceClient.updatePlan({
      plan: planPatch,
      updateMask: ["steps"],
    });

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    events.emit("update");
  }
};
</script>
