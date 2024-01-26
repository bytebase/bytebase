<template>
  <NPopover
    trigger="click"
    placement="bottom-end"
    @update:show="(show: boolean) => show && handleShowSchemaString()"
  >
    <template #trigger>
      <NButton quaternary size="tiny" class="!px-1" v-bind="$attrs">
        <CodeIcon class="w-4 h-4" />
      </NButton>
    </template>
    <div class="w-[32rem] h-[20rem] flex flex-col justify-center items-center">
      <BBSpin v-if="isFetching" />
      <template v-else>
        <div
          class="w-full flex flex-row justify-between items-center gap-x-2 mb-2"
        >
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
          class="border w-full h-full"
        />
      </template>
    </div>
  </NPopover>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { CodeIcon, ClipboardIcon } from "lucide-vue-next";
import { NButton, NPopover, NTooltip } from "naive-ui";
import { onMounted, ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { MonacoEditor } from "@/components/MonacoEditor";
import { sqlServiceClient } from "@/grpcweb";
import { pushNotification, useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

const props = defineProps<{
  database: ComposedDatabase;
  schema?: SchemaMetadata;
  table?: TableMetadata;
}>();

const { t } = useI18n();
const dbSchemaStore = useDBSchemaV1Store();
const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});
const schemaString = ref<string | null>(null);
const isFetching = ref<boolean>(false);

const engine = computed(() => {
  return props.database.instanceEntity.engine;
});

const resourceName = computed(() => {
  if (props.table) {
    if (hasSchemaProperty.value) {
      return `${props.schema?.name}.${props.table.name}`;
    } else {
      return props.table.name;
    }
  }
  if (props.schema) {
    return props.schema.name;
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
    silent: true,
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

const handleShowSchemaString = async () => {
  const databaseMetadata = DatabaseMetadata.fromPartial({
    name: props.database.name,
  });
  if (props.schema) {
    const schemaMetadata = SchemaMetadata.fromPartial({
      name: props.schema.name,
    });
    if (props.table) {
      schemaMetadata.tables = [
        TableMetadata.fromPartial({
          ...cloneDeep(props.table),
          foreignKeys: [],
        }),
      ];
    }
    databaseMetadata.schemas = [schemaMetadata];
  }

  isFetching.value = true;
  const { schema } = await sqlServiceClient.stringifyMetadata({
    metadata: databaseMetadata,
    engine: engine.value,
  });
  schemaString.value = schema.trim();
  isFetching.value = false;
};
</script>
