<template>
  <div v-if="available">
    <NButton :loading="exporting" @click="exportSchema">
      {{ $t("database.export-schema") }}
    </NButton>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { databaseServiceClientConnect } from "@/grpcweb";
import { pushNotification } from "@/store";
import { isValidDatabaseName } from "@/types";
import type { ComposedDatabase } from "@/types";
import {
  GetDatabaseSDLSchemaRequestSchema,
  GetDatabaseSDLSchemaRequest_SDLFormat,
} from "@/types/proto-es/v1/database_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const exporting = ref(false);
const { t } = useI18n();

const available = computed(() => {
  if (!isValidDatabaseName(props.database.name)) {
    return false;
  }

  return hasProjectPermissionV2(
    props.database.projectEntity,
    "bb.databases.getSchema"
  );
});

const exportSchema = async () => {
  exporting.value = true;

  try {
    const request = create(GetDatabaseSDLSchemaRequestSchema, {
      name: `${props.database.name}/sdlSchema`,
      format: GetDatabaseSDLSchemaRequest_SDLFormat.SINGLE_FILE,
    });

    const response =
      await databaseServiceClientConnect.getDatabaseSDLSchema(request);

    const filename = `${props.database.databaseName}_schema.sql`;
    const content = new TextDecoder().decode(response.schema);

    const blob = new Blob([content], { type: "text/plain" });
    const downloadLink = document.createElement("a");
    downloadLink.href = URL.createObjectURL(blob);
    downloadLink.download = filename;
    document.body.appendChild(downloadLink);
    downloadLink.click();
    document.body.removeChild(downloadLink);
    URL.revokeObjectURL(downloadLink.href);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("database.successfully-exported-schema"),
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("database.failed-to-export-schema"),
      description: (error as ConnectError).message,
    });
  } finally {
    exporting.value = false;
  }
};
</script>
