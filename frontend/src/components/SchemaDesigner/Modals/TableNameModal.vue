<template>
  <BBModal
    :title="
      isCreatingTable
        ? $t('schema-editor.actions.create-table')
        : $t('schema-editor.actions.rename')
    "
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72">
      <p>{{ $t("schema-editor.table.name") }}</p>
      <BBTextField
        class="my-2 w-full"
        :required="true"
        :focus-on-mount="true"
        :value="state.tableName"
        @input="handleTableNameChange"
      />
    </div>
    <div class="w-full flex items-center justify-end mt-2 space-x-3 pr-1 pb-1">
      <button type="button" class="btn-normal" @click="dismissModal">
        {{ $t("common.cancel") }}
      </button>
      <button class="btn-primary" @click="handleConfirmButtonClick">
        {{ isCreatingTable ? $t("common.create") : $t("common.save") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { UNKNOWN_ID, SchemaEditorTabType } from "@/types";
import {
  useSchemaEditorStore,
  useNotificationStore,
  generateUniqueTabId,
  useDatabaseV1Store,
} from "@/store";
import { ColumnMetadata, TableMetadata } from "@/types/proto/store/database";
import {
  convertColumnMetadataToColumn,
  convertTableMetadataToTable,
  Schema,
} from "@/types/schemaEditor/atomType";
import { Engine } from "@/types/proto/v1/common";

const tableNameFieldRegexp = /^\S+$/;

interface LocalState {
  tableName: string;
}

const props = defineProps({
  databaseId: {
    type: String,
    default: String(UNKNOWN_ID),
  },
  schemaId: {
    type: String,
    default: "",
  },
  tableId: {
    type: String,
    default: undefined,
  },
});

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const editorStore = useSchemaEditorStore();
const notificationStore = useNotificationStore();
const state = reactive<LocalState>({
  tableName: "",
});

const isCreatingTable = computed(() => {
  return props.tableId === undefined;
});

onMounted(() => {
  if (props.tableId === undefined) {
    return;
  }

  const table = editorStore.getTable(
    props.databaseId,
    props.schemaId,
    props.tableId
  );
  if (table) {
    state.tableName = table.name;
  }
});

const handleTableNameChange = (event: Event) => {
  state.tableName = (event.target as HTMLInputElement).value;
};

const handleConfirmButtonClick = async () => {
  if (!tableNameFieldRegexp.test(state.tableName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.invalid-table-name"),
    });
    return;
  }

  const databaseId = props.databaseId;
  const database = useDatabaseV1Store().getDatabaseByUID(databaseId);
  const instanceEngine = database.instanceEntity.engine;
  const schema = editorStore.getSchema(databaseId, props.schemaId) as Schema;
  const tableNameList = schema.tableList.map((table) => table.name);
  if (tableNameList.includes(state.tableName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.duplicated-table-name"),
    });
    return;
  }

  if (isCreatingTable.value) {
    const table = convertTableMetadataToTable(TableMetadata.fromPartial({}));
    table.name = state.tableName;
    table.status = "created";

    const column = convertColumnMetadataToColumn(
      ColumnMetadata.fromPartial({})
    );
    column.name = "id";
    if (instanceEngine === Engine.POSTGRES) {
      column.type = "INTEGER";
    } else {
      column.type = "INT";
    }
    column.comment = "ID";
    column.status = "created";
    table.columnList.push(column);
    table.primaryKey.columnIdList.push(column.id);

    schema.tableList.push(table);
    editorStore.addTab({
      id: generateUniqueTabId(),
      type: SchemaEditorTabType.TabForTable,
      databaseId: props.databaseId,
      schemaId: props.schemaId,
      tableId: table.id,
    });
  } else {
    const table = editorStore.getTable(
      props.databaseId,
      props.schemaId,
      props.tableId ?? ""
    );
    if (table) {
      table.name = state.tableName;
    }
  }
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
