<template>
  <PermissionGuardWrapper
    v-if="available"
    v-slot="slotProps"
    :project="database.projectEntity"
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
import type { ComposedDatabase } from "@/types";
import { isValidDatabaseName } from "@/types";

const props = defineProps<{
  text: boolean;
  type: "default" | "primary";
  database: ComposedDatabase;
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
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t(
        "db.successfully-synced-schema-for-database-database-value-name",
        [props.database.databaseName]
      ),
    });
    emit("finish");
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("db.failed-to-sync-schema-for-database-database-value-name", [
        props.database.databaseName,
      ]),
      description: (error as ConnectError).message,
    });
  } finally {
    syncingSchema.value = false;
  }
};
</script>
