<template>
  <div
    class="w-full h-auto flex flex-col justify-start items-center"
    v-bind="$attrs"
  >
    <div class="w-full flex flex-row justify-between items-center gap-x-2 mb-2">
      <div class="textlabel flex-1 truncate">
        {{ resourceName }}
      </div>
      <div class="flex flex-row justify-end items-center">
        <NTooltip>
          <template #trigger>
            <NButton
              quaternary
              size="tiny"
              class="!px-1"
              @click="handleCopySchemaString"
            >
              <ClipboardIcon class="w-4 h-4" />
            </NButton>
          </template>
          {{ $t("common.copy") }}
        </NTooltip>
      </div>
    </div>
    <MonacoEditor
      :content="schemaString || ''"
      :readonly="true"
      class="border w-full h-auto grow"
    />
  </div>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import { cloneDeep, isUndefined } from "lodash-es";
import { ClipboardIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { onMounted, ref, computed } from "vue";
import { nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { MonacoEditor } from "@/components/MonacoEditor";
import { sqlServiceClient } from "@/grpcweb";
import { pushNotification, useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DatabaseMetadata,
  DatabaseMetadataView,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

const props = defineProps<{
  database: ComposedDatabase;
  schema?: string;
  table?: string;
}>();

const { t } = useI18n();
const dbSchemaStore = useDBSchemaV1Store();
const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});
const schemaString = ref<string | null>(null);

const engine = computed(() => {
  return props.database.instanceEntity.engine;
});

const databaseMetadata = computed(() => {
  return dbSchemaStore.getDatabaseMetadata(props.database.name);
});

const schemaMetadata = computed(() => {
  if (!isUndefined(props.schema)) {
    return databaseMetadata.value?.schemas.find(
      (schema) => schema.name === props.schema
    );
  }
  return null;
});

const tableMetadata = computed(() => {
  if (props.table && schemaMetadata.value) {
    return schemaMetadata.value.tables.find(
      (table) => table.name === props.table
    );
  }
  return null;
});

const resourceName = computed(() => {
  if (props.table) {
    if (hasSchemaProperty.value) {
      return `${props.schema}.${props.table}`;
    } else {
      return props.table;
    }
  }
  if (props.schema) {
    return props.schema;
  }
  return props.database.name;
});

const hasSchemaProperty = computed(() => {
  return (
    engine.value === Engine.POSTGRES ||
    engine.value === Engine.SNOWFLAKE ||
    engine.value === Engine.ORACLE ||
    engine.value === Engine.DM ||
    engine.value === Engine.MSSQL ||
    engine.value === Engine.REDSHIFT ||
    engine.value === Engine.RISINGWAVE
  );
});

onMounted(async () => {
  await dbSchemaStore.getOrFetchDatabaseMetadata({
    database: props.database.name,
    view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
    silent: true,
  });

  nextTick(async () => {
    const mockedDatabaseMetadata = DatabaseMetadata.fromPartial({
      name: props.database.name,
    });
    if (!isUndefined(props.schema)) {
      const schemaMetadata = SchemaMetadata.fromPartial({
        name: props.schema,
      });
      if (props.table) {
        schemaMetadata.tables = [
          TableMetadata.fromPartial({
            ...cloneDeep(tableMetadata.value),
            foreignKeys: [],
          }),
        ];
      }
      mockedDatabaseMetadata.schemas = [schemaMetadata];
    }

    const { schema } = await sqlServiceClient.stringifyMetadata({
      metadata: mockedDatabaseMetadata,
      engine: engine.value,
    });
    schemaString.value = schema.trim();
  });
});

const handleCopySchemaString = () => {
  if (!isSupported.value) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Copy to clipboard is not enabled in your browser.",
    });
    return;
  }

  copyTextToClipboard(schemaString.value || "");
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-editor.copy-success"),
  });
};
</script>
