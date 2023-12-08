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
import { InputInst, NButton, NInput } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useNotificationStore } from "@/store";
import { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { useSchemaEditorContext } from "../context";
import { upsertColumnPrimaryKey } from "../edit";

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
const context = useSchemaEditorContext();
const { addTab, markEditStatus } = context;
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
    const table = TableMetadata.fromPartial({
      name: state.tableName,
      columns: [],
    });
    schema.tables.push(table);
    markEditStatus(
      props.database,
      {
        database: props.metadata,
        schema,
        table,
      },
      "created"
    );

    const column = ColumnMetadata.fromPartial({});
    column.name = "id";
    const engine = props.database.instanceEntity.engine;
    column.type = engine === Engine.POSTGRES ? "INTEGER" : "INT";
    column.comment = "ID";
    table.columns.push(column);
    upsertColumnPrimaryKey(table, column.name);
    markEditStatus(
      props.database,
      {
        database: props.metadata,
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
  } else {
    const { table } = props;
    table.name = state.tableName;
    markEditStatus(
      props.database,
      {
        database: props.metadata,
        schema,
        table,
      },
      "updated"
    );
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
