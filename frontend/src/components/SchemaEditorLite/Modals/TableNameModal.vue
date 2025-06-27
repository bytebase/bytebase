<template>
  <BBModal
    :title="
      mode === 'create'
        ? $t('schema-editor.actions.create-table')
        : $t('schema-editor.actions.rename')
    "
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72">
      <p>{{ $t("schema-editor.table.name") }}</p>
      <NInput
        ref="inputRef"
        v-model:value="state.tableName"
        class="my-2"
        :autofocus="true"
      />
    </div>
    <div class="w-full flex items-center justify-end mt-2 space-x-3">
      <NButton @click="dismissModal">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton type="primary" @click="handleConfirmButtonClick">
        {{ mode === "create" ? $t("common.create") : $t("common.save") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import type { InputInst } from "naive-ui";
import { NButton, NInput } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import { useNotificationStore } from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  DatabaseMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type {
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  ColumnMetadataSchema,
  TableMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";
import { upsertColumnPrimaryKey } from "../edit";
import { create } from "@bufbuild/protobuf";

// Table name must start with a non-space character, end with a non-space character, and can contain space in between.
const tableNameFieldRegexp = /^\S[\S ]*\S?$/;

interface LocalState {
  tableName: string;
}

const props = defineProps<{
  database: ComposedDatabase;
  metadata: DatabaseMetadata;
  schema: SchemaMetadata;
  table?: TableMetadata;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const {
  events,
  addTab,
  markEditStatus,
  queuePendingScrollToTable,
  upsertTableCatalog,
  removeTableCatalog,
} = useSchemaEditorContext();
const inputRef = ref<InputInst>();
const notificationStore = useNotificationStore();
const mode = computed(() => {
  return props.table ? "edit" : "create";
});
const state = reactive<LocalState>({
  tableName: props.table?.name ?? "",
});

const handleConfirmButtonClick = async () => {
  if (!tableNameFieldRegexp.test(state.tableName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.invalid-table-name"),
    });
    return;
  }
  const { schema } = props;
  const existedTable = schema.tables.find(
    (table) => table.name === state.tableName
  );
  if (existedTable) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.duplicated-table-name"),
    });
    return;
  }

  if (!props.table) {
    const table = create(TableMetadataSchema, {
      name: state.tableName,
      columns: [],
    });
    schema.tables.push(table);
    markEditStatus(
      props.database,
      {
        schema,
        table,
      },
      "created"
    );

    const column = create(ColumnMetadataSchema, {});
    column.name = "id";
    const engine = props.database.instanceResource.engine;
    column.type = engine === Engine.POSTGRES ? "integer" : "int";
    column.comment = "";
    table.columns.push(column);
    upsertColumnPrimaryKey(engine, table, column.name);
    markEditStatus(
      props.database,
      {
        schema,
        table,
        column,
      },
      "created"
    );

    addTab({
      type: "table",
      database: props.database,
      metadata: {
        database: props.metadata,
        schema: props.schema,
        table,
      },
    });

    queuePendingScrollToTable({
      db: props.database,
      metadata: {
        database: props.metadata,
        schema: props.schema,
        table,
      },
    });
    events.emit("rebuild-tree", {
      openFirstChild: false,
    });
  } else {
    const { table } = props;
    upsertTableCatalog(
      {
        database: props.database.name,
        schema: props.schema.name,
        table: table.name,
      },
      (catalog) => {
        catalog.name = state.tableName;
      }
    );
    removeTableCatalog({
      database: props.database.name,
      schema: props.schema.name,
      table: table.name,
    });

    table.name = state.tableName;
    events.emit("rebuild-edit-status", {
      resets: ["tree"],
    });
  }
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};

onMounted(() => {
  inputRef.value?.focus();
});
</script>
