<template>
  <BBModal
    :title="$t('schema-editor.actions.create-table')"
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
        {{ $t("common.create") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useNotificationStore } from "@/store";
import { ColumnMetadata, TableMetadata } from "@/types/proto/store/database";
import { Engine } from "@/types/proto/v1/common";
import { useSchemaDesignerContext } from "../common";
import { convertTableMetadataToTable } from "@/types";

const tableNameFieldRegexp = /^\S+$/;

interface LocalState {
  tableName: string;
}

const props = defineProps({
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
const { engine, editableSchemas } = useSchemaDesignerContext();
const notificationStore = useNotificationStore();
const state = reactive<LocalState>({
  tableName: "",
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

  const schema = editableSchemas.value.find(
    (schema) => schema.id === props.schemaId
  );
  if (!schema) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.schema-not-found"),
    });
    return;
  }

  const table = TableMetadata.fromPartial({});
  table.name = state.tableName;
  const column = ColumnMetadata.fromPartial({});
  column.name = "id";
  if (engine === Engine.POSTGRES) {
    column.type = "INTEGER";
  } else {
    column.type = "INT";
  }
  column.comment = "ID";
  table.columns.push(column);
  schema.tableList.push(convertTableMetadataToTable(table));

  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
