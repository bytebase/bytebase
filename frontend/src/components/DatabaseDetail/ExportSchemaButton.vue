<template>
  <PermissionGuardWrapper
    v-if="available"
    v-slot="slotProps"
    :project="getDatabaseProject(database)"
    :permissions="['bb.databases.getSchema']"
  >
    <NDropdown
      :options="exportOptions"
      :disabled="slotProps.disabled"
      @select="handleExportFormat"
      trigger="click"
    >
      <NButton :loading="exporting">
        {{ $t("database.export-schema") }}
      </NButton>
    </NDropdown>
  </PermissionGuardWrapper>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";
import { NButton, NDropdown } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { databaseServiceClientConnect } from "@/connect";
import { pushNotification } from "@/store";
import { isValidDatabaseName } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  GetDatabaseSDLSchemaRequest_SDLFormat,
  GetDatabaseSDLSchemaRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName, getDatabaseProject } from "@/utils";

const props = defineProps<{
  database: Database;
}>();

const exporting = ref(false);
const { t } = useI18n();

const available = computed(() => {
  return isValidDatabaseName(props.database.name);
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

    const { databaseName } = extractDatabaseResourceName(props.database.name);
    let filename: string;
    let blob: Blob;

    // Check the content type to determine how to handle the response
    if (response.contentType.includes("application/zip")) {
      // Multi-file format - ZIP archive
      filename = `${databaseName}_schema.zip`;
      // Create a proper ArrayBuffer by slicing
      const arrayBuffer = response.schema.slice().buffer;
      blob = new Blob([arrayBuffer], { type: "application/zip" });
    } else {
      // Single file format - text file
      filename = `${databaseName}_schema.sql`;
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
