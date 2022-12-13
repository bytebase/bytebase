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
import { computed, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { DatabaseId, UNKNOWN_ID, SchemaEditorTabType, unknown } from "@/types";
import { TableTabContext } from "@/types/schemaEditor";
import {
  useSchemaEditorStore,
  useNotificationStore,
  generateUniqueTabId,
} from "@/store";
import {
  transformColumnDataToColumn,
  transformTableDataToTable,
} from "@/utils/schemaEditor/transform";

const tableNameFieldRegexp = /^\S+$/;

interface LocalState {
  tableName: string;
}

const props = defineProps({
  databaseId: {
    type: Number as PropType<DatabaseId>,
    default: UNKNOWN_ID,
  },
  tableName: {
    type: String as PropType<string | undefined>,
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
  tableName: props.tableName || "",
});

const isCreatingTable = computed(() => {
  return props.tableName === undefined;
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
  const tableList = await editorStore.getOrFetchTableListByDatabaseId(
    databaseId
  );
  const tableNameList = tableList.map((table) => table.newName);
  if (tableNameList.includes(state.tableName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.duplicated-table-name"),
    });
    return;
  }

  if (isCreatingTable.value) {
    const unknownTable = unknown("TABLE");
    const unknownColumn = unknown("COLUMN");
    unknownColumn.name = "id";
    unknownColumn.type = "int";
    unknownColumn.comment = "ID";

    const table = transformTableDataToTable(unknownTable);
    table.databaseId = databaseId;
    table.oldName = state.tableName;
    table.newName = state.tableName;
    table.status = "created";
    const column = transformColumnDataToColumn(unknownColumn);
    column.status = "created";
    table.columnList.push(column);
    editorStore.tableList.push(table);
    editorStore.addTab({
      id: generateUniqueTabId(),
      type: SchemaEditorTabType.TabForTable,
      databaseId: props.databaseId,
      tableName: table.newName,
    });
    dismissModal();
  } else {
    const table = editorStore.tableList.find(
      (table) =>
        table.databaseId === databaseId && table.newName === props.tableName
    );
    if (table) {
      const tab = editorStore.findTab(
        table.databaseId,
        table.newName
      ) as TableTabContext;
      table.newName = state.tableName;
      if (tab) {
        tab.tableName = table.newName;
      }
    }
    dismissModal();
  }
};

const dismissModal = () => {
  emit("close");
};
</script>
