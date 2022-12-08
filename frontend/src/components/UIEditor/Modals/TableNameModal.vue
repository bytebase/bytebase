<template>
  <BBModal
    :title="
      isCreatingTable
        ? $t('ui-editor.actions.create-table')
        : $t('ui-editor.actions.rename')
    "
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72">
      <p>{{ $t("ui-editor.table.name") }}</p>
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
import { DatabaseId, Table, UNKNOWN_ID, UIEditorTabType } from "@/types";
import {
  useUIEditorStore,
  useNotificationStore,
  generateUniqueTabId,
} from "@/store";
import { cloneDeep } from "lodash-es";
import { useI18n } from "vue-i18n";

const tableNameFieldRegexp = /^\S+$/;

interface LocalState {
  tableName: string;
}

const props = defineProps({
  databaseId: {
    type: Number as PropType<DatabaseId>,
    default: UNKNOWN_ID,
  },
  table: {
    type: Object as PropType<Table | undefined>,
    default: undefined,
  },
});

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const editorStore = useUIEditorStore();
const notificationStore = useNotificationStore();
const state = reactive<LocalState>({
  tableName: props.table?.name || "",
});

const isCreatingTable = computed(() => {
  return props.table === undefined;
});

const handleTableNameChange = (event: Event) => {
  state.tableName = (event.target as HTMLInputElement).value;
};

const handleConfirmButtonClick = async () => {
  if (!tableNameFieldRegexp.test(state.tableName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("ui-editor.message.invalid-table-name"),
    });
    return;
  }

  const databaseId = props.databaseId;
  const tableList = await editorStore.getOrFetchTableListByDatabaseId(
    databaseId
  );
  const tableNameList = tableList.map((table) => table.name);
  if (tableNameList.includes(state.tableName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("ui-editor.message.duplicated-table-name"),
    });
    return;
  }

  if (isCreatingTable.value) {
    const table = editorStore.createNewTable(props.databaseId);
    table.name = state.tableName;
    editorStore.addTab({
      id: generateUniqueTabId(),
      type: UIEditorTabType.TabForTable,
      databaseId: props.databaseId,
      tableId: table.id,
      table: table,
      tableCache: cloneDeep(table),
    });
    dismissModal();
  } else {
    const originTable = editorStore.tableList.find(
      (item) => item === props.table
    );
    if (originTable) {
      originTable.name = state.tableName;
      const tab = editorStore.getTableTabByTable(originTable);
      if (tab) {
        tab.tableCache.name = originTable.name;
      }
    }
    dismissModal();
  }
};

const dismissModal = () => {
  emit("close");
};
</script>
