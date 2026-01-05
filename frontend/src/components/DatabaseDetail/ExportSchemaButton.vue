<template>
  <div v-if="available">
    <NDropdown
      :options="exportOptions"
      @select="handleExportFormat"
      trigger="click"
    >
      <NButton :loading="exporting">
        {{ $t("database.export-schema") }}
      </NButton>
    </NDropdown>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";
import { NButton, NDropdown } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { databaseServiceClientConnect } from "@/connect";
import { pushNotification } from "@/store";
import type { ComposedDatabase } from "@/types";
import { isValidDatabaseName } from "@/types";
import {
  GetDatabaseSDLSchemaRequest_SDLFormat,
  GetDatabaseSDLSchemaRequestSchema,
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

const exportOptions = computed(() => [
  {
    label: t("database.export-schema-single-file"),
    key: "single",
  },
  {
    label: t("database.export-schema-multi-file"),
    key: "multi",
  },
]);

const handleExportFormat = (key: string) => {
  if (key === "single") {
    exportSchema(GetDatabaseSDLSchemaRequest_SDLFormat.SINGLE_FILE);
  } else if (key === "multi") {
    exportSchema(GetDatabaseSDLSchemaRequest_SDLFormat.MULTI_FILE);
  }
};

const exportSchema = async (format: GetDatabaseSDLSchemaRequest_SDLFormat) => {
  exporting.value = true;

  try {
    const request = create(GetDatabaseSDLSchemaRequestSchema, {
      name: `${props.database.name}/sdlSchema`,
      format,
    });

    const response =
      await databaseServiceClientConnect.getDatabaseSDLSchema(request);

    let filename: string;
    let blob: Blob;

    // Check the content type to determine how to handle the response
    if (response.contentType.includes("application/zip")) {
      // Multi-file format - ZIP archive
      filename = `${props.database.databaseName}_schema.zip`;
      // Create a proper ArrayBuffer by slicing
      const arrayBuffer = response.schema.slice().buffer;
      blob = new Blob([arrayBuffer], { type: "application/zip" });
    } else {
      // Single file format - text file
      filename = `${props.database.databaseName}_schema.sql`;
      const content = new TextDecoder().decode(response.schema);
      blob = new Blob([content], { type: "text/plain" });
    }

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
