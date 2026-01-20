<template>
  <PermissionGuardWrapper
    v-if="available"
    v-slot="slotProps"
    :project="getDatabaseProject(database)"
    :permissions="['bb.databases.sync']"
  >
    <NButton
      :text="text"
      :type="type"
      :disabled="slotProps.disabled"
      :loading="syncingSchema"
      @click="syncDatabaseSchema"
    >
      {{ $t("database.sync-database") }}
    </NButton>
  </PermissionGuardWrapper>
</template>

<script setup lang="ts">
import { ConnectError } from "@connectrpc/connect";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import {
  pushNotification,
  useDatabaseV1Store,
  useDBSchemaV1Store,
} from "@/store";
import { isValidDatabaseName } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName, getDatabaseProject } from "@/utils";

const props = defineProps<{
  text: boolean;
  type: "default" | "primary";
  database: Database;
}>();

const emit = defineEmits<{
  (event: "finish"): void;
}>();

const syncingSchema = ref(false);
const { t } = useI18n();
const databaseV1Store = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();

const available = computed(() => {
  return isValidDatabaseName(props.database.name);
});

const syncDatabaseSchema = async () => {
  syncingSchema.value = true;

  try {
    await databaseV1Store.syncDatabase(props.database.name);

    await dbSchemaStore.getOrFetchDatabaseMetadata({
      database: props.database.name,
      skipCache: true,
    });
    const { databaseName } = extractDatabaseResourceName(props.database.name);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t(
        "db.successfully-synced-schema-for-database-database-value-name",
        [databaseName]
      ),
    });
    emit("finish");
  } catch (error) {
    const { databaseName } = extractDatabaseResourceName(props.database.name);
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("db.failed-to-sync-schema-for-database-database-value-name", [
        databaseName,
      ]),
      description: (error as ConnectError).message,
    });
  } finally {
    syncingSchema.value = false;
  }
};
</script>
