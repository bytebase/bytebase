<template>
  <div v-if="available">
    <NButton
      :text="text"
      :type="type"
      :loading="syncingSchema"
      @click="syncDatabaseSchema"
    >
      {{ $t("database.sync-database") }}
    </NButton>
  </div>
</template>

<script setup lang="ts">
import { ConnectError } from "@connectrpc/connect";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useDatabaseV1Store,
  useDBSchemaV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import { isValidDatabaseName } from "@/types";
import { hasProjectPermissionV2 } from "@/utils";

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
  if (!isValidDatabaseName(props.database.name)) {
    return false;
  }

  return hasProjectPermissionV2(
    props.database.projectEntity,
    "bb.databases.sync"
  );
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
