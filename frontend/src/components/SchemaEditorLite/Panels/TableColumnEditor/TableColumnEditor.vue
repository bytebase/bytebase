<template>
  <div
    v-show="show"
    ref="containerElRef"
    class="w-full h-full overflow-x-auto"
    :data-height="containerHeight"
    :data-table-header-height="tableHeaderHeight"
    :data-table-body-height="tableBodyHeight"
  >
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="getColumnKey"
      :columns="columns"
      :data="layoutReady ? shownColumnList : []"
      :row-class-name="classesForRow"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      class="schema-editor-table-column-editor"
      :class="[disableDiffColoring && 'disable-diff-coloring']"
    />
  </div>

  <ColumnDefaultValueExpressionModal
    v-if="editColumnDefaultValueExpressionContext"
    :expression="editColumnDefaultValueExpressionContext.default"
    @close="editColumnDefaultValueExpressionContext = undefined"
    @update:expression="handleSelectColumnDefaultValueExpression"
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
    :labels="[catalogForColumn(state.pendingUpdateColumn.name).labels]"
    @dismiss="state.showLabelsDrawer = false"
    @apply="onLabelsApply"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { useElementSize } from "@vueuse/core";
import { pick } from "lodash-es";
import type { DataTableColumn, DataTableInst } from "naive-ui";
import { NCheckbox, NDataTable } from "naive-ui";
import { computed, h, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import ClassificationCell from "@/components/ColumnDataTable/ClassificationCell.vue";
import LabelEditorDrawer from "@/components/LabelEditorDrawer.vue";
import SemanticTypesDrawer from "@/components/SensitiveData/components/SemanticTypesDrawer.vue";
import { InlineInput } from "@/components/v2";
import { useSettingV1Store, hasFeature } from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { ColumnCatalog } from "@/types/proto-es/v1/database_catalog_service_pb";
import { ColumnCatalogSchema } from "@/types/proto-es/v1/database_catalog_service_pb";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import ColumnDefaultValueExpressionModal from "../../Modals/ColumnDefaultValueExpressionModal.vue";
import { useSchemaEditorContext } from "../../context";
import type { EditStatus } from "../../types";
import type { DefaultValueOption } from "../../utils";
import { markUUID } from "../common";
import {
  DataTypeCell,
  ForeignKeyCell,
  OperationCell,
  ReorderCell,
  SelectionCell,
  DefaultValueCell,
  SemanticTypeCell,
  LabelsCell,
} from "./components";

interface LocalState {
  pendingUpdateColumn?: ColumnMetadata;
  showSemanticTypesDrawer: boolean;
  showLabelsDrawer: boolean;
}

const props = withDefaults(
  defineProps<{
    show?: boolean;
    readonly: boolean;
    showForeignKey?: boolean;
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
    engine: Engine;
    disableChangeTable?: boolean;
    allowChangePrimaryKeys?: boolean;
    allowReorderColumns?: boolean;
    maxBodyHeight?: number;
    showDatabaseCatalogColumn?: boolean;
    showClassificationColumn?: "ALWAYS" | "AUTO";
    filterColumn?: (column: ColumnMetadata) => boolean;
    disableAlterColumn?: (column: ColumnMetadata) => boolean;
    getColumnItemComputedClassList?: (column: ColumnMetadata) => string;
  }>(),
  {
    show: true,
    showForeignKey: true,
    disableChangeTable: false,
    allowChangePrimaryKeys: false,
    allowReorderColumns: false,
    maxBodyHeight: undefined,
    showDatabaseCatalogColumn: false,
    showClassificationColumn: "AUTO",
    filterColumn: (_: ColumnMetadata) => true,
    disableAlterColumn: (_: ColumnMetadata) => false,
    getColumnItemComputedClassList: (_: ColumnMetadata) => "",
  }
);

const emit = defineEmits<{
  (event: "drop", column: ColumnMetadata): void;
  (event: "restore", column: ColumnMetadata): void;
  (
    event: "reorder",
    column: ColumnMetadata,
    index: number,
    delta: -1 | 1
  ): void;
  (
    event: "foreign-key-edit",
    column: ColumnMetadata,
    fk: ForeignKeyMetadata | undefined
  ): void;
  (
    event: "foreign-key-click",
    column: ColumnMetadata,
    fk: ForeignKeyMetadata
  ): void;
  (
    event: "primary-key-set",
    column: ColumnMetadata,
    isPrimaryKey: boolean
  ): void;
}>();

const state = reactive<LocalState>({
  showSemanticTypesDrawer: false,
  showLabelsDrawer: false,
});

const {
  classificationConfig,
  showClassificationColumn: canShowClassificationColumn,
  disableDiffColoring,
  selectionEnabled,
  markEditStatus,
  getColumnStatus,
  getColumnCatalog,
  removeColumnCatalog,
  upsertColumnCatalog,
  useConsumePendingScrollToColumn,
  getAllColumnsSelectionState,
  updateAllColumnsSelection,
} = useSchemaEditorContext();
const dataTableRef = ref<DataTableInst>();
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
  const bodyHeight = containerHeight.value - tableHeaderHeight.value - 2;
  const { maxBodyHeight = 0 } = props;
  if (maxBodyHeight > 0) {
    return Math.min(maxBodyHeight, bodyHeight);
  }
  return bodyHeight;
});
// Use this to avoid unnecessary initial rendering
const layoutReady = computed(() => tableHeaderHeight.value > 0);
const { t } = useI18n();
const settingStore = useSettingV1Store();
const editColumnDefaultValueExpressionContext = ref<ColumnMetadata>();

const metadataForColumn = (column: ColumnMetadata) => {
  return {
    database: props.database,
    schema: props.schema,
    table: props.table,
    column,
  };
};

const statusForColumn = (column: ColumnMetadata) => {
  return getColumnStatus(props.db, metadataForColumn(column));
};

const markColumnStatus = (
  column: ColumnMetadata,
  status: EditStatus,
  oldStatus: EditStatus | undefined = undefined
) => {
  if (!oldStatus) {
    oldStatus = statusForColumn(column);
  }
  if (
    (oldStatus === "created" || oldStatus === "dropped") &&
    status === "updated"
  ) {
    markEditStatus(props.db, metadataForColumn(column), oldStatus);
    return;
  }
  markEditStatus(props.db, metadataForColumn(column), status);
};

const semanticTypeList = computed(() => {
  const setting = settingStore.getSettingByName(
    Setting_SettingName.SEMANTIC_TYPES
  );
  if (setting?.value?.value?.case === "semanticTypeSettingValue") {
    return setting.value.value.value.types ?? [];
  }
  return [];
});

const catalogForColumn = (column: string) => {
  return (
    getColumnCatalog({
      database: props.db.name,
      schema: props.schema.name,
      table: props.table.name,
      column,
    }) ?? create(ColumnCatalogSchema, { name: column })
  );
};

const primaryKey = computed(() => {
  return props.table.indexes.find((idx) => idx.primary);
});

const showClassification = computed(() => {
  return (
    props.showClassificationColumn === "ALWAYS" ||
    canShowClassificationColumn(
      props.engine,
      classificationConfig.value?.classificationFromConfig ?? false
    )
  );
});

const openSemanticTypeDrawer = (column: ColumnMetadata) => {
  state.pendingUpdateColumn = column;
  state.showSemanticTypesDrawer = true;
};

const openLabelsDrawer = (column: ColumnMetadata) => {
  state.pendingUpdateColumn = column;
  state.showLabelsDrawer = true;
};

const columns = computed(() => {
  const columns: (DataTableColumn<ColumnMetadata> & { hide?: boolean })[] = [
    {
      key: "__selected__",
      width: 32,
      hide: !selectionEnabled.value,
      title: () => {
        const state = getAllColumnsSelectionState(
          props.db,
          pick(props, "database", "schema", "table"),
          shownColumnList.value
        );
        return h(NCheckbox, {
          checked: state.checked,
          indeterminate: state.indeterminate,
          onUpdateChecked: (on: boolean) => {
            updateAllColumnsSelection(
              props.db,
              pick(props, "database", "schema", "table"),
              shownColumnList.value,
              on
            );
          },
        });
      },
      render: (column) => {
        return h(SelectionCell, {
          db: props.db,
          metadata: {
            ...pick(props, "database", "schema", "table"),
            column,
          },
        });
      },
    },
    {
      key: "reorder",
      title: "",
      resizable: false,
      width: 44,
      hide: props.readonly || !props.allowReorderColumns,
      className: "!px-0",
      render: (column, index) => {
        return h(ReorderCell, {
          allowMoveUp: index > 0,
          allowMoveDown: index < shownColumnList.value.length - 1,
          disabled: props.disableChangeTable,
          onReorder: (delta: -1 | 1) => emit("reorder", column, index, delta),
        });
      },
    },
    {
      key: "name",
      title: t("schema-editor.column.name"),
      resizable: true,
      minWidth: 140,
      className: "input-cell",
      render: (column) => {
        return h(InlineInput, {
          value: column.name,
          disabled: props.readonly || props.disableAlterColumn(column),
          placeholder: "column name",
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
            "--n-text-color-disabled": "rgb(var(--color-main))",
          },
          "onUpdate:value": (value: string) => {
            upsertColumnCatalog(
              {
                database: props.db.name,
                schema: props.schema.name,
                table: props.table.name,
                column: column.name,
              },
              (catalog) => {
                catalog.name = value;
              }
            );
            removeColumnCatalog({
              database: props.db.name,
              schema: props.schema.name,
              table: props.table.name,
              column: column.name,
            });
            const oldStatus = statusForColumn(column);

            column.name = value;
            markColumnStatus(column, "updated", oldStatus);
          },
        });
      },
    },
    {
      key: "semantic-types",
      title: t("settings.sensitive-data.semantic-types.table.semantic-type"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      hide:
        !props.showDatabaseCatalogColumn ||
        !hasFeature(PlanFeature.FEATURE_DATA_MASKING),
      render: (column) => {
        return h(SemanticTypeCell, {
          database: props.database.name,
          schema: props.schema.name,
          table: props.table.name,
          column: column.name,
          readonly: props.readonly,
          disabled:
            props.disableChangeTable || props.disableAlterColumn(column),
          semanticTypeList: semanticTypeList.value,
          onRemove: () => onSemanticTypeRemove(column),
          onEdit: () => openSemanticTypeDrawer(column),
        });
      },
    },
    {
      key: "classification",
      title: t("schema-editor.column.classification"),
      hide: !showClassification.value,
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      render: (column) => {
        const config = catalogForColumn(column.name);
        return h(ClassificationCell, {
          classification: config.classification,
          readonly: props.readonly,
          disabled: props.disableChangeTable,
          engine: props.engine,
          classificationConfig: classificationConfig.value,
          onApply: (id: string) => {
            state.pendingUpdateColumn = column;
            onClassificationSelect(id);
          },
        });
      },
    },
    {
      key: "type",
      title: t("schema-editor.column.type"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "input-cell",
      render: (column) => {
        return h(DataTypeCell, {
          column,
          disabled: props.readonly || props.disableAlterColumn(column),
          schemaTemplateColumnTypes: schemaTemplateColumnTypes.value,
          engine: props.engine,
          "onUpdate:value": (value: string) => {
            column.type = value;
            markColumnStatus(column, "updated");
          },
        });
      },
    },
    {
      key: "default-value",
      title: t("schema-editor.column.default"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "input-cell",
      render: (column) => {
        return h(DefaultValueCell, {
          column,
          disabled: props.readonly || props.disableAlterColumn(column),
          engine: props.engine,
          onInput: (value) => handleColumnDefaultInput(column, value),
          onSelect: (option) => handleColumnDefaultSelect(column, option),
        });
      },
    },
    {
      key: "on-update",
      title: t("schema-editor.column.on-update"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      hide: props.engine !== Engine.MYSQL && props.engine !== Engine.TIDB,
      className: "input-cell",
      render: (column) => {
        return h(InlineInput, {
          value: column.onUpdate,
          disabled: props.readonly || props.disableAlterColumn(column),
          placeholder: "",
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
            "--n-text-color-disabled": "rgb(var(--color-main))",
          },
          "onUpdate:value": (value: string) => {
            column.onUpdate = value;
            markColumnStatus(column, "updated");
          },
        });
      },
    },
    {
      key: "comment",
      title: t("schema-editor.column.comment"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "input-cell",
      render: (column) => {
        return h(InlineInput, {
          value: column.userComment,
          disabled: props.readonly || props.disableAlterColumn(column),
          placeholder: "comment",
          style: {
            "--n-padding-left": "6px",
            "--n-padding-right": "4px",
            "--n-text-color-disabled": "rgb(var(--color-main))",
          },
          "onUpdate:value": (value: string) => {
            column.userComment = value;
            markColumnStatus(column, "updated");
          },
        });
      },
    },
    {
      key: "not-null",
      title: t("schema-editor.column.not-null"),
      resizable: true,
      minWidth: 80,
      maxWidth: 160,
      className: "checkbox-cell",
      render: (column) => {
        return h(NCheckbox, {
          checked: !column.nullable,
          disabled:
            props.readonly ||
            props.disableAlterColumn(column) ||
            isColumnPrimaryKey(column),
          "onUpdate:checked": (checked: boolean) => {
            column.nullable = !checked;
            markColumnStatus(column, "updated");
          },
        });
      },
    },
    {
      key: "primary",
      title: t("schema-editor.column.primary"),
      resizable: true,
      minWidth: 80,
      maxWidth: 160,
      className: "checkbox-cell",
      render: (column) => {
        return h(NCheckbox, {
          checked: isColumnPrimaryKey(column),
          disabled:
            props.readonly ||
            !props.allowChangePrimaryKeys ||
            props.disableAlterColumn(column),
          "onUpdate:checked": (checked: boolean) =>
            emit("primary-key-set", column, checked),
        });
      },
    },
    {
      key: "foreign-key",
      title: t("schema-editor.column.foreign-key"),
      hide: !props.showForeignKey,
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      className: "text-cell",
      render: (column) => {
        return h(ForeignKeyCell, {
          db: props.db,
          database: props.database,
          schema: props.schema,
          table: props.table,
          column: column,
          readonly: props.readonly,
          disabled: props.readonly || props.disableAlterColumn(column),
          onClick: (fk: ForeignKeyMetadata) =>
            emit("foreign-key-click", column, fk),
          onEdit: (fk: ForeignKeyMetadata | undefined) =>
            emit("foreign-key-edit", column, fk),
        });
      },
    },
    {
      key: "labels",
      title: t("common.labels"),
      resizable: true,
      minWidth: 140,
      maxWidth: 320,
      hide: !props.showDatabaseCatalogColumn,
      render: (column) => {
        return h(LabelsCell, {
          database: props.database.name,
          schema: props.schema.name,
          table: props.table.name,
          column: column.name,
          readonly: props.readonly,
          disabled: props.disableChangeTable,
          onEdit: () => openLabelsDrawer(column),
        });
      },
    },
    {
      key: "operations",
      title: "",
      resizable: false,
      width: 30,
      hide: props.readonly,
      className: "!px-0",
      render: (column) => {
        return h(OperationCell, {
          dropped: isDroppedColumn(column),
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
  const filtered = props.table.columns.filter(props.filterColumn);
  if (disableDiffColoring.value) {
    return filtered.filter((column) => {
      const status = statusForColumn(column);
      return status !== "dropped";
    });
  }
  return filtered;
});

const isColumnPrimaryKey = (column: ColumnMetadata): boolean => {
  const pk = primaryKey.value;
  if (!pk) return false;
  return pk.expressions.includes(column.name);
};

const schemaTemplateColumnTypes = computed(() => {
  const setting = settingStore.getSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );
  if (setting?.value?.value?.case === "schemaTemplateSettingValue") {
    const columnTypes = setting.value.value.value.columnTypes;
    if (columnTypes && columnTypes.length > 0) {
      const columnType = columnTypes.find(
        (columnType) => columnType.engine === props.engine
      );
      if (columnType && columnType.enabled) {
        return columnType.types;
      }
    }
  }
  return [];
});

const handleColumnDefaultInput = (column: ColumnMetadata, value: string) => {
  if (!column.hasDefault) return;
  if (column.default === "NULL") return;

  column.default = value;
  markColumnStatus(column, "updated");
};

const handleColumnDefaultSelect = (
  column: ColumnMetadata,
  option: DefaultValueOption
) => {
  const defaults = option.value;
  if (!defaults.hasDefault) {
    Object.assign(column, defaults);
    markColumnStatus(column, "updated");
    return;
  }
  if (defaults.default === "NULL") {
    Object.assign(column, defaults);
    markColumnStatus(column, "updated");
    return;
  }
  if (typeof defaults.default === "string") {
    Object.assign(column, {
      ...defaults,
      // copy current editing string to column.default
      default: column.default ?? defaults.default ?? "",
    });
    markColumnStatus(column, "updated");
    return;
  }
};

const handleSelectColumnDefaultValueExpression = (expression: string) => {
  const column = editColumnDefaultValueExpressionContext.value;
  if (!column) {
    return;
  }
  column.hasDefault = true;
  column.default = expression;

  markColumnStatus(column, "updated");
};

const onSemanticTypeApply = async (semanticTypeId: string) => {
  if (!state.pendingUpdateColumn) {
    return;
  }

  updateColumnConfig(state.pendingUpdateColumn, (catalog) => {
    catalog.semanticType = semanticTypeId;
  });
};

const onSemanticTypeRemove = async (column: ColumnMetadata) => {
  markColumnStatus(column, "updated");
  updateColumnConfig(column, (catalog) => {
    catalog.semanticType = "";
  });
};

const onClassificationSelect = (classificationId: string) => {
  if (!state.pendingUpdateColumn) {
    return;
  }

  markColumnStatus(state.pendingUpdateColumn, "updated");
  updateColumnConfig(state.pendingUpdateColumn, (catalog) => {
    catalog.classification = classificationId;
  });
};

const updateColumnConfig = (
  column: ColumnMetadata,
  update: (config: ColumnCatalog) => void
) => {
  upsertColumnCatalog(
    {
      database: props.db.name,
      schema: props.schema.name,
      table: props.table.name,
      column: column.name,
    },
    update
  );
  markColumnStatus(column, "updated");
};

const onLabelsApply = (labelsList: { [key: string]: string }[]) => {
  if (!state.pendingUpdateColumn) {
    return;
  }
  updateColumnConfig(state.pendingUpdateColumn, (catalog) => {
    catalog.labels = labelsList[0];
  });
  markColumnStatus(state.pendingUpdateColumn, "updated");
};

const classesForRow = (column: ColumnMetadata) => {
  return props.getColumnItemComputedClassList(column);
};

const isDroppedColumn = (column: ColumnMetadata): boolean => {
  return statusForColumn(column) === "dropped";
};

const getColumnKey = (column: ColumnMetadata) => {
  return markUUID(column);
};

const vlRef = computed(() => {
  return (dataTableRef.value as any)?.$refs?.mainTableInstRef?.bodyInstRef
    ?.virtualListRef;
});
useConsumePendingScrollToColumn(
  computed(() => ({
    db: props.db,
    metadata: {
      database: props.database,
      schema: props.schema,
      table: props.table,
    },
  })),
  vlRef,
  (params, vl) => {
    const key = getColumnKey(params.metadata.column);
    if (!key) return;
    requestAnimationFrame(() => {
      try {
        console.debug("scroll-to-column", vl, params, key);
        vl.scrollTo({ key });
        // TODO: focus name or type input element
      } catch {
        // Do nothing
      }
    });
  }
);
</script>

<style lang="postcss" scoped>
.schema-editor-table-column-editor
  :deep(.n-data-table-th .n-data-table-resize-button::after) {
  @apply bg-control-bg h-2/3;
}
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
.schema-editor-table-column-editor:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.created .n-data-table-td) {
  @apply text-green-700 !bg-green-50;
}
.schema-editor-table-column-editor:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.dropped .n-data-table-td) {
  @apply text-red-700 cursor-not-allowed !bg-red-50 opacity-70;
}
.schema-editor-table-column-editor:not(.disable-diff-coloring)
  :deep(.n-data-table-tr.updated .n-data-table-td) {
  @apply text-yellow-700 !bg-yellow-50;
}
</style>
