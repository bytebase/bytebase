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
        {{ isCreatingTable ? $t("common.create") : $t("common.save") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { InputInst, NButton, NInput } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  generateUniqueTabId,
  useNotificationStore,
  useSchemaEditorV1Store,
} from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  ColumnMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import {
  SchemaEditorTabType,
  convertTableMetadataToTable,
} from "@/types/v1/schemaEditor";

// Table name must start with a non-space character, end with a non-space character, and can contain space in between.
const tableNameFieldRegexp = /^\S[\S ]*\S?$/;

interface LocalState {
  tableName: string;
}

const props = defineProps<{
  parentName: string;
  schemaId: string;
  tableId?: string;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const inputRef = ref<InputInst>();
const schemaEditorV1Store = useSchemaEditorV1Store();
const notificationStore = useNotificationStore();
const state = reactive<LocalState>({
  tableName: "",
});

const engine = computed(() => {
  return schemaEditorV1Store.getCurrentEngine(props.parentName);
});

const isCreatingTable = computed(() => {
  return props.tableId === undefined;
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

  const schema = schemaEditorV1Store.getSchema(
    props.parentName,
    props.schemaId
  );
  if (!schema) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.schema-not-found"),
    });
    return;
  }
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
    const column = ColumnMetadata.fromPartial({});
    column.name = "id";
    if (engine.value === Engine.POSTGRES) {
      column.type = "INTEGER";
    } else {
      column.type = "INT";
    }
    column.comment = "ID";
    const table = TableMetadata.fromPartial({
      name: state.tableName,
      columns: [column],
    });
    const tableEdit = convertTableMetadataToTable(table, "created");
    tableEdit.primaryKey.columnIdList.push(
      ...tableEdit.columnList.map((col) => col.id)
    );
    schema.tableList.push(tableEdit);
    schemaEditorV1Store.addTab({
      id: generateUniqueTabId(),
      type: SchemaEditorTabType.TabForTable,
      parentName: props.parentName,
      schemaId: props.schemaId,
      tableId: tableEdit.id,
      name: state.tableName,
    });
  } else {
    const table = schema.tableList.find((table) => table.id === props.tableId);
    if (table) {
      table.name = state.tableName;
    }
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
