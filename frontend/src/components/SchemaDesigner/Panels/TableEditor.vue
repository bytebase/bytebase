<template>
  <div class="flex flex-col w-full h-full overflow-y-auto">
    <div class="py-2 w-full flex flex-row justify-between items-center">
      <div>
        <div
          v-if="state.selectedSubtab === 'column-list'"
          class="w-full flex justify-between items-center space-x-2"
        >
          <button
            class="flex flex-row justify-center items-center border px-3 py-1 leading-6 text-sm text-gray-700 rounded cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
            @click="handleAddColumn"
          >
            <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
            {{ $t("schema-editor.actions.add-column") }}
          </button>
        </div>
      </div>
      <div class="flex justify-end items-center">
        <NInput
          v-if="state.selectedSubtab === 'column-list'"
          v-model:value="searchPattern"
          class="!w-48 mr-3"
          :placeholder="$t('schema-editor.search-column')"
        >
          <template #prefix>
            <heroicons-outline:search class="w-4 h-auto text-gray-300" />
          </template>
        </NInput>
        <div
          class="flex flex-row justify-end items-center bg-gray-100 p-1 rounded"
        >
          <button
            class="px-2 leading-7 text-sm text-gray-500 cursor-pointer select-none rounded flex justify-center items-center"
            :class="
              state.selectedSubtab === 'column-list' &&
              'bg-gray-200 text-gray-800'
            "
            @click="handleChangeTab('column-list')"
          >
            <heroicons-outline:queue-list class="inline w-4 h-auto mr-1" />
            {{ $t("schema-editor.columns") }}
          </button>
          <button
            class="px-2 leading-7 text-sm text-gray-500 cursor-pointer select-none rounded flex justify-center items-center"
            :class="
              state.selectedSubtab === 'raw-sql' && 'bg-gray-200 text-gray-800'
            "
            @click="handleChangeTab('raw-sql')"
          >
            <heroicons-outline:clipboard class="inline w-4 h-auto mr-1" />
            {{ $t("schema-editor.raw-sql") }}
          </button>
        </div>
      </div>
    </div>

    <template v-if="state.selectedSubtab === 'column-list'">
      <!-- column table -->
      <div
        id="table-editor-container"
        ref="tableEditorContainerRef"
        class="w-full h-auto grid auto-rows-auto border-y relative overflow-y-auto"
      >
        <!-- column table header -->
        <div
          class="sticky top-0 z-10 grid grid-cols-[repeat(4,_minmax(0,_1fr))_repeat(2,_96px)_minmax(0,_1fr)_32px] w-full text-sm leading-6 select-none bg-gray-50 text-gray-400"
          :class="shownColumnList.length > 0 && 'border-b'"
        >
          <span
            v-for="header in columnHeaderList"
            :key="header.key"
            class="table-header-item-container"
            >{{ header.label }}</span
          >
          <span></span>
        </div>
        <!-- column table body -->
        <div class="w-full">
          <div
            v-for="(column, index) in shownColumnList"
            :key="`${column.name}-${index}`"
            class="grid grid-cols-[repeat(4,_minmax(0,_1fr))_repeat(2,_96px)_minmax(0,_1fr)_32px] gr text-sm even:bg-gray-50"
            :class="[getColumnComputedClassList(column)]"
          >
            <div class="table-body-item-container">
              <input
                v-model="column.name"
                placeholder="column name"
                class="column-field-input column-name-input"
                type="text"
              />
            </div>
            <div
              class="table-body-item-container flex flex-row justify-between items-center"
            >
              <input
                v-model="column.type"
                placeholder="column type"
                class="column-field-input column-type-input !pr-8"
                type="text"
              />
              <NDropdown
                trigger="click"
                :options="dataTypeOptions"
                @select="(dataType: string) => (column.type = dataType)"
              >
                <button class="absolute right-5">
                  <heroicons-solid:chevron-up-down
                    class="w-4 h-auto text-gray-400"
                  />
                </button>
              </NDropdown>
            </div>
            <div
              class="table-body-item-container flex flex-row justify-between items-center"
            >
              <input
                v-model="column.default"
                :placeholder="column.default === undefined ? 'NULL' : 'EMPTY'"
                class="column-field-input !pr-8"
                type="text"
              />
              <NDropdown
                trigger="click"
                :options="dataDefaultOptions"
                @select="(defaultString:string)=>handleColumnDefaultFieldChange(column, defaultString)"
              >
                <button class="absolute right-5">
                  <heroicons-solid:chevron-up-down
                    class="w-4 h-auto text-gray-400"
                  />
                </button>
              </NDropdown>
            </div>
            <div class="table-body-item-container">
              <input
                v-model="column.comment"
                placeholder="comment"
                class="column-field-input"
                type="text"
              />
            </div>
            <div
              class="table-body-item-container flex justify-start items-center"
            >
              <BBCheckbox
                class="ml-3"
                :value="!column.nullable"
                :disabled="isColumnPrimaryKey(column)"
                @toggle="(value) => (column.nullable = !value)"
              />
            </div>
            <div
              class="table-body-item-container flex justify-start items-center"
            >
              <BBCheckbox
                class="ml-3"
                :value="isColumnPrimaryKey(column)"
                @toggle="(value) => setColumnPrimaryKey(column, value)"
              />
            </div>
            <div
              class="table-body-item-container foreign-key-field flex justify-start items-center"
            >
              <span
                v-if="checkColumnHasForeignKey(column)"
                class="column-field-text cursor-pointer !w-auto hover:opacity-80"
              >
                {{ getReferencedForeignKeyName(column) }}
              </span>
              <span
                v-else
                class="column-field-text italic text-gray-400 !w-auto"
                >EMPTY</span
              >
              <button
                class="foreign-key-edit-button hidden cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
                @click="handleEditColumnForeignKey(column)"
              >
                <heroicons:pencil-square class="w-4 h-auto text-gray-400" />
              </button>
            </div>
            <div class="w-full flex justify-start items-center">
              <n-tooltip trigger="hover">
                <template #trigger>
                  <button
                    class="text-gray-500 cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
                    @click="handleDropColumn(column)"
                  >
                    <heroicons:trash class="w-4 h-auto" />
                  </button>
                </template>
                <span>{{ $t("schema-editor.actions.drop-column") }}</span>
              </n-tooltip>
            </div>
          </div>
        </div>
      </div>
    </template>
    <div
      v-else-if="state.selectedSubtab === 'raw-sql'"
      class="w-full h-full overflow-y-auto"
    >
      <div
        v-if="state.isFetchingDDL"
        class="w-full h-full min-h-[64px] flex justify-center items-center"
      >
        <BBSpin />
      </div>
      <template v-else>
        <HighlightCodeBlock
          v-if="state.statement !== ''"
          class="text-sm px-3 py-2 whitespace-pre-wrap break-all"
          language="sql"
          :code="state.statement"
        ></HighlightCodeBlock>
        <div v-else class="flex px-3 py-2 italic text-sm text-gray-600">
          {{ $t("schema-editor.nothing-changed") }}
        </div>
      </template>
    </div>
  </div>

  <EditColumnForeignKeyModal
    v-if="state.showEditColumnForeignKeyModal && editForeignKeyColumn"
    :schema="schema.name"
    :table="table.name"
    :column="editForeignKeyColumn.name"
    @close="state.showEditColumnForeignKeyModal = false"
  />
</template>

<script lang="ts" setup>
import { isUndefined } from "lodash-es";
import { computed, nextTick, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { NDropdown } from "naive-ui";
import { ColumnMetadata } from "@/types/proto/store/database";
import { getDataTypeSuggestionList } from "@/utils";
import { BBCheckbox, BBSpin } from "@/bbkit";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";
import EditColumnForeignKeyModal from "../Modals/EditColumnForeignKeyModal.vue";
import { Engine } from "@/types/proto/v1/common";
import { TableTabContext, useSchemaDesignerContext } from "../common";
import {
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

type SubtabType = "column-list" | "raw-sql";

interface LocalState {
  selectedSubtab: SubtabType;
  isFetchingDDL: boolean;
  statement: string;
  showEditColumnForeignKeyModal: boolean;
}

const { t } = useI18n();
const { engine, metadata, getCurrentTab } = useSchemaDesignerContext();
const currentTab = computed(() => getCurrentTab() as TableTabContext);
const state = reactive<LocalState>({
  selectedSubtab:
    (currentTab.value.selectedSubtab as SubtabType) || "column-list",
  isFetchingDDL: false,
  statement: "",
  showEditColumnForeignKeyModal: false,
});

const schema = computed(() => {
  return metadata.value.schemas.find(
    (schema) => schema.name === currentTab.value.schema
  ) as SchemaMetadata;
});
const table = computed(() => {
  return schema.value.tables.find(
    (table) => table.name === currentTab.value.table
  ) as TableMetadata;
});
const foreignKeyList = computed(() => {
  return table.value.foreignKeys;
});

const searchPattern = ref("");
const tableEditorContainerRef = ref<HTMLDivElement>();
const editForeignKeyColumn = ref<ColumnMetadata>();

const shownColumnList = computed(() => {
  return table.value.columns.filter((column) =>
    column.name.includes(searchPattern.value.trim())
  );
});

const columnHeaderList = computed(() => {
  return [
    {
      key: "name",
      label: t("schema-editor.column.name"),
    },
    {
      key: "type",
      label: t("schema-editor.column.type"),
    },
    {
      key: "default",
      label: t("schema-editor.column.default"),
    },
    {
      key: "comment",
      label: t("schema-editor.column.comment"),
    },
    {
      key: "nullable",
      label: t("schema-editor.column.not-null"),
    },
    {
      key: "primary",
      label: t("schema-editor.column.primary"),
    },
    {
      key: "foreign_key",
      label: "Foreign Key",
    },
  ];
});

const dataDefaultOptions = [
  {
    label: "NULL",
    key: "NULL",
  },
  {
    label: "EMPTY",
    key: "EMPTY",
  },
];

const dataTypeOptions = computed(() => {
  return getDataTypeSuggestionList(engine).map((dataType) => {
    return {
      label: dataType,
      key: dataType,
    };
  });
});

const getColumnComputedClassList = (column: ColumnMetadata) => {
  return [`column-${column.name}`];
};

const isColumnPrimaryKey = (column: ColumnMetadata): boolean => {
  return (
    table.value.indexes.find(
      (index) => index.primary && index.expressions.includes(column.name)
    ) !== undefined
  );
};

const checkColumnHasForeignKey = (column: ColumnMetadata): boolean => {
  return foreignKeyList.value
    .map((fk) => fk.columns)
    .flat()
    .flat()
    .includes(column.name);
};

const getReferencedForeignKeyName = (column: ColumnMetadata) => {
  if (!checkColumnHasForeignKey(column)) {
    return;
  }
  const fk = foreignKeyList.value.find(
    (fk) =>
      fk.columns.find((columnName) => columnName === column.name) !== undefined
  );
  const index = fk?.columns.findIndex(
    (columnName) => columnName === column.name
  );
  if (isUndefined(fk) || isUndefined(index) || index < 0) {
    return;
  }

  const referencedSchema = fk.referencedSchema[index];
  const referencedTable = fk.referencedTable[index];
  const referencedColumn = fk.referencedSchema[index];
  if (!referencedTable || !referencedColumn) {
    return;
  }

  if (engine === Engine.MYSQL) {
    return `${referencedTable}(${referencedColumn})`;
  } else {
    return `${referencedSchema}.${referencedTable}(${referencedColumn})`;
  }
};

const setColumnPrimaryKey = (column: ColumnMetadata, isPrimaryKey: boolean) => {
  // TODO
};

const handleChangeTab = (tab: SubtabType) => {
  state.selectedSubtab = tab;
};

const handleAddColumn = () => {
  const column = ColumnMetadata.fromPartial({});
  table.value.columns.push(column);
  nextTick(() => {
    (
      tableEditorContainerRef.value?.querySelector(
        `.column-${column.name} .column-name-input`
      ) as HTMLInputElement
    )?.focus();
  });
};

const handleColumnDefaultFieldChange = (
  column: ColumnMetadata,
  defaultString: string
) => {
  if (defaultString === "NULL") {
    column.default = undefined;
  } else if (defaultString === "EMPTY") {
    column.default = "";
  }
};

const handleEditColumnForeignKey = (column: ColumnMetadata) => {
  editForeignKeyColumn.value = column;
  state.showEditColumnForeignKeyModal = true;
};

const handleDropColumn = (column: ColumnMetadata) => {
  table.value.columns = table.value.columns.filter((col) => col !== column);
};
</script>

<style scoped>
.table-header-item-container {
  @apply py-2 px-3;
}
.table-body-item-container {
  @apply w-full h-10 box-border p-px pr-2 relative;
}
.column-field-input {
  @apply w-full pr-1 box-border border-transparent truncate select-none rounded bg-transparent text-sm placeholder:italic placeholder:text-gray-400 focus:bg-white focus:text-black;
}
.column-field-text {
  @apply w-full pl-3 pr-1 box-border border-transparent truncate select-none rounded bg-transparent text-sm placeholder:italic placeholder:text-gray-400 focus:bg-white focus:text-black;
}
.foreign-key-field:hover .foreign-key-edit-button {
  @apply block;
}
</style>
