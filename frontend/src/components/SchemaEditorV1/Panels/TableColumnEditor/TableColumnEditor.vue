<template>
  <div
    ref="containerElRef"
    class="w-full h-full"
    :data-height="containerHeight"
  >
    <NDataTable
      v-bind="$attrs"
      size="small"
      class="schema-editor-table-column-editor"
      :row-key="(column: Column) => column.id"
      :columns="columns"
      :data="shownColumnList"
      :row-class-name="classesForRow"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
    />
  </div>

  <ColumnDefaultValueExpressionModal
    v-if="editColumnDefaultValueExpressionContext"
    :expression="editColumnDefaultValueExpressionContext.defaultExpression"
    @close="editColumnDefaultValueExpressionContext = undefined"
    @update:expression="handleSelectColumnDefaultValueExpression"
  />

  <SelectClassificationDrawer
    v-if="classificationConfig && state.pendingUpdateColumn"
    :show="state.showClassificationDrawer"
    :classification-config="classificationConfig"
    @dismiss="state.showClassificationDrawer = false"
    @apply="onClassificationSelect"
  />

  <SemanticTypesDrawer
    v-if="state.pendingUpdateColumn"
    :show="state.showSemanticTypesDrawer"
    :semantic-type-list="semanticTypeList"
    @dismiss="state.showSemanticTypesDrawer = false"
    @apply="onSemanticTypeApply($event)"
  />

  <LabelEditorDrawer
    v-if="state.pendingUpdateColumn"
    :show="state.showLabelsDrawer"
    :readonly="false"
    :title="
      $t('db.labels-for-resource', {
        resource: `'${state.pendingUpdateColumn.name}'`,
      })
    "
    :labels="[state.pendingUpdateColumn.config.labels]"
    @dismiss="state.showLabelsDrawer = false"
    @apply="onLabelsApply"
  />
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { flatten } from "lodash-es";
import { DataTableColumn, NCheckbox, NDataTable } from "naive-ui";
import { computed, h, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import LabelEditorDrawer from "@/components/LabelEditorDrawer.vue";
import SelectClassificationDrawer from "@/components/SchemaTemplate/SelectClassificationDrawer.vue";
import SemanticTypesDrawer from "@/components/SensitiveData/components/SemanticTypesDrawer.vue";
import { InlineInput } from "@/components/v2";
import { useSettingV1Store, useSubscriptionV1Store } from "@/store/modules";
import { Engine } from "@/types/proto/v1/common";
import { ColumnConfig } from "@/types/proto/v1/database_service";
import { Table, Column, ForeignKey } from "@/types/v1/schemaEditor";
import ColumnDefaultValueExpressionModal from "../../Modals/ColumnDefaultValueExpressionModal.vue";
import {
  getDefaultValueByKey,
  isTextOfColumnType,
} from "../../utils/columnDefaultValue";
import {
  ClassificationCell,
  DataTypeCell,
  ForeignKeyCell,
  OperationCell,
  SemanticTypeCell,
} from "./components";
import DefaultValueCell from "./components/DefaultValueCell.vue";
import LabelsCell from "./components/LabelsCell.vue";

interface LocalState {
  pendingUpdateColumn?: Column;
  showClassificationDrawer: boolean;
  showSemanticTypesDrawer: boolean;
  showLabelsDrawer: boolean;
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
    getColumnItemComputedClassList?: (column: Column) => string;
  }>(),
  {
    showForeignKey: true,
    disableChangeTable: false,
    foreignKeyList: () => [],
    classificationConfigId: "",
    filterColumn: (_: Column) => true,
    disableAlterColumn: (_: Column) => false,
    getReferencedForeignKeyName: (_: Column) => "",
    getColumnItemComputedClassList: (_: Column) => "",
  }
);

const emit = defineEmits<{
  (event: "drop", column: Column): void;
  (event: "restore", column: Column): void;
  (event: "edit", index: number): void;
  (event: "foreign-key-edit", column: Column): void;
  (event: "foreign-key-click", column: Column): void;
  (event: "primary-key-set", column: Column, isPrimaryKey: boolean): void;
}>();

const state = reactive<LocalState>({
  showClassificationDrawer: false,
  showSemanticTypesDrawer: false,
  showLabelsDrawer: false,
});

const containerElRef = ref<HTMLElement>();
const tableHeaderElRef = computed(
  () =>
    containerElRef.value?.querySelector("thead.n-data-table-thead") as
      | HTMLElement
      | undefined
);
const { height: containerHeight } = useElementSize(containerElRef);
const { height: tableHeaderHeight } = useElementSize(tableHeaderElRef);
const tableBodyHeight = computed(() => {
  return containerHeight.value - tableHeaderHeight.value;
});
const { t } = useI18n();
const subscriptionV1Store = useSubscriptionV1Store();
const settingStore = useSettingV1Store();
const editColumnDefaultValueExpressionContext = ref<Column>();

const classificationConfig = computed(() => {
  if (!props.classificationConfigId) {
    return;
  }
  return settingStore.getProjectClassification(props.classificationConfigId);
});

const semanticTypeList = computed(() => {
  return (
    settingStore.getSettingByName("bb.workspace.semantic-types")?.value
      ?.semanticTypeSettingValue?.types ?? []
  );
});

const showSemanticTypeColumn = computed(() => {
  return subscriptionV1Store.hasFeature("bb.feature.sensitive-data");
});

const columns = computed(() => {
  const columns: (DataTableColumn<Column> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("schema-editor.column.name"),
      width: 120,
      className: "input-cell",
      render: (column) => {
        return h(InlineInput, {
          value: column.name,
          disabled: props.readonly || props.disableAlterColumn(column),
          placeholder: "column name",
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
          },
          "onUpdate:value": (value) => (column.name = value),
        });
      },
    },
    {
      key: "semantic-types",
      title: t("settings.sensitive-data.semantic-types.self"),
      hide: !showSemanticTypeColumn.value,
      render: (column) => {
        return h(SemanticTypeCell, {
          column,
          readonly: props.readonly,
          disabled: props.disableChangeTable,
          semanticTypeList: semanticTypeList.value,
          disableAlterColumn: props.disableAlterColumn,
          onRemove: () => onSemanticTypeRemove(column),
          onEdit: () => openSemanticTypeDrawer(column),
        });
      },
    },
    {
      key: "classification",
      title: t("schema-editor.column.classification"),
      hide: !classificationConfig.value,
      render: (column) => {
        return h(ClassificationCell, {
          column,
          readonly: props.readonly,
          disabled: props.disableChangeTable,
          classificationConfig: classificationConfig.value,
          onEdit: () => openClassificationDrawer(column),
          onRemove: () => (column.classification = ""),
        });
      },
    },
    {
      key: "type",
      title: t("schema-editor.column.type"),
      width: 120,
      className: "input-cell",
      render: (column) => {
        return h(DataTypeCell, {
          column,
          disabled: props.readonly || props.disableAlterColumn(column),
          schemaTemplateColumnTypes: schemaTemplateColumnTypes.value,
          engine: props.engine,
          "onUpdate:value": (value) => (column.type = value),
        });
      },
    },
    {
      key: "default-value",
      title: t("schema-editor.column.default"),
      width: 120,
      className: "input-cell",
      render: (column) => {
        return h(DefaultValueCell, {
          column,
          disabled: props.readonly || props.disableAlterColumn(column),
          schemaTemplateColumnTypes: schemaTemplateColumnTypes.value,
          engine: props.engine,
          onInput: (value) => handleColumnDefaultInput(column, value),
          onSelect: (option) => handleColumnDefaultSelect(column, option.key),
        });
      },
    },
    {
      key: "comment",
      title: t("schema-editor.column.comment"),
      width: 120,
      className: "input-cell",
      render: (column) => {
        return h(InlineInput, {
          value: column.userComment,
          disabled: props.readonly || props.disableAlterColumn(column),
          placeholder: "comment",
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
          },
          "onUpdate:value": (value) => (column.userComment = value),
        });
      },
    },
    {
      key: "not-null",
      title: t("schema-editor.column.not-null"),
      width: 80,
      className: "checkbox-cell",
      render: (column) => {
        return h(NCheckbox, {
          checked: !column.nullable,
          disabled:
            props.readonly ||
            props.disableAlterColumn(column) ||
            isColumnPrimaryKey(column),
          "onUpdate:checked": (checked: boolean) =>
            (column.nullable = !checked),
        });
      },
    },
    {
      key: "primary",
      title: t("schema-editor.column.primary"),
      width: 80,
      className: "checkbox-cell",
      render: (column) => {
        return h(NCheckbox, {
          checked: isColumnPrimaryKey(column),
          disabled: props.readonly || props.disableAlterColumn(column),
          "onUpdate:checked": (checked: boolean) =>
            emit("primary-key-set", column, checked),
        });
      },
    },
    {
      key: "foreign-key",
      title: t("schema-editor.column.foreign-key"),
      hide: !props.showForeignKey,
      width: 120,
      className: "text-cell",
      render: (column) => {
        return h(ForeignKeyCell, {
          column,
          readonly: props.readonly,
          disabled: props.readonly || props.disableAlterColumn(column),
          hasForeignKey: checkColumnHasForeignKey(column),
          referencedForeignKeyName: props.getReferencedForeignKeyName(column),
          onClick: () => emit("foreign-key-click", column),
          onEdit: () => emit("foreign-key-edit", column),
        });
      },
    },
    {
      key: "labels",
      title: t("common.labels"),
      render: (column) => {
        return h(LabelsCell, {
          column,
          readonly: props.readonly,
          disabled: props.disableChangeTable,
          onEdit: () => openLabelsDrawer(column),
        });
      },
    },
    {
      key: "operations",
      title: "",
      width: 30,
      hide: props.readonly,
      className: "!px-0",
      render: (column) => {
        return h(OperationCell, {
          column,
          disabled: props.disableChangeTable,
          onDrop: () => emit("drop", column),
          onRestore: () => emit("restore", column),
        });
      },
    },
  ];
  return columns.filter((header) => !header.hide);
});

const shownColumnList = computed(() => {
  return props.table.columnList.filter(props.filterColumn);
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

const handleColumnDefaultInput = (column: Column, value: string) => {
  column.hasDefault = true;
  column.defaultNull = undefined;
  // If column is text type or has default string, we will treat user's input as string.
  if (
    isTextOfColumnType(props.engine, column.type) ||
    column.defaultString !== undefined
  ) {
    column.defaultString = value;
    column.defaultExpression = undefined;
    return;
  }
  // Otherwise we will treat user's input as expression.
  column.defaultExpression = value;
};

const handleColumnDefaultSelect = (column: Column, key: string) => {
  if (key === "expression") {
    editColumnDefaultValueExpressionContext.value = column;
    return;
  }

  const defaultValue = getDefaultValueByKey(key);
  if (!defaultValue) {
    return;
  }

  column.hasDefault = defaultValue.hasDefault;
  column.defaultNull = defaultValue.defaultNull;
  column.defaultString = defaultValue.defaultString;
  column.defaultExpression = defaultValue.defaultExpression;
  if (column.hasDefault && column.defaultNull) {
    column.nullable = true;
  }
};

const handleSelectColumnDefaultValueExpression = (expression: string) => {
  const column = editColumnDefaultValueExpressionContext.value;
  if (!column) {
    return;
  }
  column.hasDefault = true;
  column.defaultNull = undefined;
  column.defaultString = undefined;
  column.defaultExpression = expression;
};

const openClassificationDrawer = (column: Column) => {
  state.pendingUpdateColumn = column;
  state.showClassificationDrawer = true;
};

const openSemanticTypeDrawer = (column: Column) => {
  state.pendingUpdateColumn = column;
  state.showSemanticTypesDrawer = true;
};

const openLabelsDrawer = (column: Column) => {
  state.pendingUpdateColumn = column;
  state.showLabelsDrawer = true;
};

const onClassificationSelect = (classificationId: string) => {
  if (!state.pendingUpdateColumn) {
    return;
  }
  state.pendingUpdateColumn.classification = classificationId;
};

const onSemanticTypeApply = async (semanticTypeId: string) => {
  if (!state.pendingUpdateColumn) {
    return;
  }

  state.pendingUpdateColumn.config = ColumnConfig.fromPartial({
    ...state.pendingUpdateColumn.config,
    semanticTypeId,
  });
};

const onLabelsApply = (labelsList: { [key: string]: string }[]) => {
  if (!state.pendingUpdateColumn) {
    return;
  }
  state.pendingUpdateColumn.config = ColumnConfig.fromPartial({
    ...state.pendingUpdateColumn.config,
    labels: labelsList[0],
  });
};

const onSemanticTypeRemove = async (column: Column) => {
  column.config = ColumnConfig.fromPartial({
    ...column.config,
    semanticTypeId: "",
  });
};

const classesForRow = (column: Column, index: number) => {
  return props.getColumnItemComputedClassList(column);
};
</script>

<style lang="postcss" scoped>
.schema-editor-table-column-editor :deep(.n-data-table-td.input-cell) {
  @apply pl-0.5 pr-1 py-0;
}
.schema-editor-table-column-editor
  :deep(.n-data-table-td.input-cell .n-input__placeholder),
.schema-editor-table-column-editor
  :deep(.n-data-table-td.input-cell .n-base-selection-placeholder) {
  @apply italic;
}
.schema-editor-table-column-editor :deep(.n-data-table-td.checkbox-cell) {
  @apply pr-1 py-0;
}
.schema-editor-table-column-editor :deep(.n-data-table-td.text-cell) {
  @apply pr-1 py-0;
}
.schema-editor-table-column-editor
  :deep(.n-data-table-tr.created .n-data-table-td) {
  @apply text-green-700 !bg-green-50;
}
.schema-editor-table-column-editor
  :deep(.n-data-table-tr.dropped .n-data-table-td) {
  @apply text-red-700 cursor-not-allowed !bg-red-50 opacity-70;
}
.schema-editor-table-column-editor
  :deep(.n-data-table-tr.updated .n-data-table-td) {
  @apply text-yellow-700 !bg-yellow-50;
}
</style>
