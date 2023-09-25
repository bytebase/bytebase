<template>
  <!-- column table -->
  <div
    id="table-editor-container"
    class="w-full h-auto grid auto-rows-auto border-y relative overflow-y-auto border"
  >
    <!-- column table header -->
    <div
      class="sticky top-0 z-10 border-b grid w-full text-sm leading-6 select-none bg-gray-50 text-gray-400"
      :class="gridColumnClass"
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
        :key="`${index}-${column.id}`"
        class="grid gr text-sm even:bg-gray-50"
        :class="[
          gridColumnClass,
          `column-${column.id}`,
          getColumnItemComputedClassList(column),
        ]"
      >
        <div class="table-body-item-container">
          <input
            v-model="column.name"
            :disabled="readonly || disableAlterColumn(column)"
            placeholder="column name"
            class="column-field-input column-name-input"
            type="text"
          />
        </div>
        <div
          class="table-body-item-container flex items-center gap-x-2 ml-3 text-sm"
        >
          <ClassificationLevelBadge
            v-if="column.classification && classificationConfig"
            :classification="column.classification"
            :classification-config="classificationConfig"
          />
          <div
            v-if="classificationConfig && !readonly && !disableChangeTable"
            class="flex"
          >
            <button
              v-if="column.classification"
              class="w-4 h-4 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer"
              @click.prevent="column.classification = ''"
            >
              <heroicons-outline:x class="w-3 h-3" />
            </button>
            <button
              class="w-4 h-4 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer"
              @click.prevent="state.pendingUpdateColumn = column"
            >
              <heroicons-outline:pencil class="w-3 h-3" />
            </button>
          </div>
        </div>
        <div
          class="table-body-item-container flex flex-row justify-between items-center"
        >
          <input
            v-model="column.type"
            :disabled="
              readonly ||
              disableAlterColumn(column) ||
              schemaTemplateColumnTypes.length > 0
            "
            placeholder="column type"
            class="column-field-input column-type-input !pr-8"
            type="text"
          />
          <NDropdown
            trigger="click"
            :disabled="readonly || disableAlterColumn(column)"
            :options="columnTypeOptions"
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
            :disabled="readonly || disableAlterColumn(column)"
            :placeholder="column.default === undefined ? 'EMPTY' : 'NULL'"
            class="column-field-input !pr-8"
            type="text"
          />
          <NDropdown
            trigger="click"
            :disabled="readonly || disableAlterColumn(column)"
            :options="dataDefaultOptions"
            @select="(defaultString:string) => handleColumnDefaultFieldChange(column, defaultString)"
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
            v-model="column.userComment"
            :disabled="readonly || disableAlterColumn(column)"
            placeholder="comment"
            class="column-field-input"
            type="text"
          />
        </div>
        <div class="table-body-item-container flex justify-start items-center">
          <BBCheckbox
            class="ml-3"
            :value="!column.nullable"
            :disabled="
              readonly ||
              disableAlterColumn(column) ||
              isColumnPrimaryKey(column)
            "
            @toggle="(value) => (column.nullable = !value)"
          />
        </div>
        <div class="table-body-item-container flex justify-start items-center">
          <BBCheckbox
            class="ml-3"
            :value="isColumnPrimaryKey(column)"
            :disabled="readonly || disableAlterColumn(column)"
            @toggle="(value) => $emit('onPrimaryKeySet', column, value)"
          />
        </div>
        <div
          v-if="showForeignKey"
          class="table-body-item-container foreign-key-field flex justify-start items-center"
        >
          <span
            v-if="checkColumnHasForeignKey(column)"
            class="column-field-text cursor-pointer !w-auto hover:opacity-80"
            @click="$emit('onForeignKeyClick', column)"
          >
            {{ getReferencedForeignKeyName(column) }}
          </span>
          <span v-else class="column-field-text italic text-gray-400 !w-auto"
            >EMPTY</span
          >
          <button
            v-if="!readonly"
            class="foreign-key-edit-button hidden cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
            :disabled="disableAlterColumn(column)"
            @click="$emit('onForeignKeyEdit', column)"
          >
            <heroicons:pencil-square class="w-4 h-auto text-gray-400" />
          </button>
        </div>
        <div class="w-full flex justify-start items-center">
          <template v-if="!readonly">
            <n-tooltip v-if="!isDroppedColumn(column)" trigger="hover">
              <template #trigger>
                <button
                  :disabled="disableChangeTable"
                  class="text-gray-500 cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
                  @click="$emit('onDrop', column)"
                >
                  <heroicons:trash class="w-4 h-auto" />
                </button>
              </template>
              <span>{{ $t("schema-editor.actions.drop-column") }}</span>
            </n-tooltip>
            <n-tooltip v-else trigger="hover">
              <template #trigger>
                <button
                  class="text-gray-500 cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
                  :disabled="disableChangeTable"
                  @click="$emit('onRestore', column)"
                >
                  <heroicons:arrow-uturn-left class="w-4 h-auto" />
                </button>
              </template>
              <span>{{ $t("schema-editor.actions.restore") }}</span>
            </n-tooltip>
          </template>
        </div>
      </div>
    </div>
  </div>

  <SelectClassificationDrawer
    v-if="classificationConfig"
    :show="state.pendingUpdateColumn !== undefined"
    :classification-config="classificationConfig"
    @dismiss="state.pendingUpdateColumn = undefined"
    @select="onClassificationSelect"
  />
</template>

<script lang="ts" setup>
import { flatten } from "lodash-es";
import { NDropdown } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBCheckbox } from "@/bbkit";
import { useSettingV1Store } from "@/store/modules";
import { Engine } from "@/types/proto/v1/common";
import { Table, Column, ForeignKey } from "@/types/v1/schemaEditor";
import { getDataTypeSuggestionList } from "@/utils";

interface LocalState {
  pendingUpdateColumn?: Column;
}

const props = withDefaults(
  defineProps<{
    readonly: boolean;
    showForeignKey?: boolean;
    table: Table;
    engine: Engine;
    foreignKeyList?: ForeignKey[];
    classificationConfigId?: string;
    disableChangeTable?: boolean;
    filterColumn?: (column: Column) => boolean;
    disableAlterColumn?: (column: Column) => boolean;
    getReferencedForeignKeyName?: (column: Column) => string;
    getColumnItemComputedClassList?: (column: Column) => string[];
  }>(),
  {
    showForeignKey: true,
    disableChangeTable: false,
    foreignKeyList: () => [],
    classificationConfigId: "",
    filterColumn: (_: Column) => true,
    disableAlterColumn: (_: Column) => false,
    getReferencedForeignKeyName: (_: Column) => "",
    getColumnItemComputedClassList: (_: Column) => [],
  }
);

defineEmits<{
  (event: "onDrop", column: Column): void;
  (event: "onRestore", column: Column): void;
  (event: "onEdit", index: number): void;
  (event: "onForeignKeyEdit", column: Column): void;
  (event: "onForeignKeyClick", column: Column): void;
  (event: "onPrimaryKeySet", column: Column, isPrimaryKey: boolean): void;
}>();

const state = reactive<LocalState>({});

const { t } = useI18n();
const settingStore = useSettingV1Store();

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

const columnHeaderList = computed(() => {
  const list = [
    {
      key: "name",
      label: t("schema-editor.column.name"),
    },
    {
      key: "classification",
      label: t("schema-editor.column.classification"),
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
  ];
  if (props.showForeignKey) {
    list.push({
      key: "foreign_key",
      label: t("schema-editor.column.foreign-key"),
    });
  }

  return list;
});

const gridColumnClass = computed(() => {
  if (props.showForeignKey) {
    return "grid-cols-[6rem_minmax(0,_1.5fr)_repeat(3,_minmax(0,_0.8fr))_repeat(2,_80px)_minmax(0,_7rem)_20px]";
  }
  return "grid-cols-[6rem_minmax(0,_1.5fr)_repeat(3,_minmax(0,_0.8fr))_repeat(2,_80px)_20px]";
});

const shownColumnList = computed(() => {
  return props.table.columnList.filter(props.filterColumn);
});

const classificationConfig = computed(() => {
  if (!props.classificationConfigId) {
    return;
  }
  return settingStore.getProjectClassification(props.classificationConfigId);
});

const isColumnPrimaryKey = (column: Column): boolean => {
  return props.table.primaryKey.columnIdList.includes(column.id);
};

const checkColumnHasForeignKey = (column: Column): boolean => {
  const columnIdList = flatten(
    props.foreignKeyList.map((fk) => fk.columnIdList)
  );
  return columnIdList.includes(column.id);
};

const schemaTemplateColumnTypes = computed(() => {
  const setting = settingStore.getSettingByName("bb.workspace.schema-template");
  const columnTypes = setting?.value?.schemaTemplateSettingValue?.columnTypes;
  if (columnTypes && columnTypes.length > 0) {
    const columnType = columnTypes.find(
      (columnType) => columnType.engine === props.engine
    );
    if (columnType && columnType.enabled) {
      return columnType.types;
    }
  }
  return [];
});

const columnTypeOptions = computed(() => {
  if (schemaTemplateColumnTypes.value.length > 0) {
    return schemaTemplateColumnTypes.value.map((columnType) => {
      return {
        label: columnType,
        key: columnType,
      };
    });
  }

  return getDataTypeSuggestionList(props.engine).map((dataType) => {
    return {
      label: dataType,
      key: dataType,
    };
  });
});

const isDroppedColumn = (column: Column): boolean => {
  return column.status === "dropped";
};

const handleColumnDefaultFieldChange = (
  column: Column,
  defaultString: string
) => {
  if (defaultString === "NULL") {
    column.default = "NULL";
  } else if (defaultString === "EMPTY") {
    column.default = undefined;
  }
};

const onClassificationSelect = (classificationId: string) => {
  if (!state.pendingUpdateColumn) {
    return;
  }
  state.pendingUpdateColumn.classification = classificationId;
  state.pendingUpdateColumn = undefined;
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
