<template>
  <div class="flex flex-col pt-2 gap-y-2 w-full h-full overflow-y-hidden">
    <div
      v-if="!readonly"
      class="w-full flex flex-row justify-between items-center"
    >
      <div>
        <div class="w-full flex justify-between items-center space-x-2">
          <NButton
            size="small"
            :disabled="disableChangeTable"
            @click="handleAddColumn"
          >
            <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
            {{ $t("schema-editor.actions.add-column") }}
          </NButton>
          <NButton
            size="small"
            :disabled="disableChangeTable"
            @click="state.showSchemaTemplateDrawer = true"
          >
            <FeatureBadge feature="bb.feature.schema-template" />
            <heroicons-outline:plus class="w-4 h-auto mr-1 text-gray-400" />
            {{ $t("schema-editor.actions.add-from-template") }}
          </NButton>
        </div>
      </div>
    </div>

    <div class="flex-1 overflow-y-hidden">
      <TableColumnEditor
        :readonly="readonly"
        :show-foreign-key="true"
        :table="table"
        :engine="engine"
        :classification-config-id="project.dataClassificationConfigId"
        :disable-change-table="disableChangeTable"
        :allow-reorder-columns="allowReorderColumns"
        :filter-column="(column: ColumnMetadata) => column.name.includes(props.searchPattern.trim())"
        :disable-alter-column="disableAlterColumn"
        :get-column-item-computed-class-list="getColumnItemComputedClassList"
        @drop="handleDropColumn"
        @restore="handleRestoreColumn"
        @reorder="handleReorderColumn"
        @primary-key-set="setColumnPrimaryKey"
        @foreign-key-edit="handleEditColumnForeignKey"
        @foreign-key-click="gotoForeignKeyReferencedTable"
      />
    </div>
  </div>

  <EditColumnForeignKeyModal
    v-if="state.showEditColumnForeignKeyModal && editForeignKeyContext"
    :database="database"
    :metadata="metadata.database"
    :schema="metadata.schema"
    :table="metadata.table"
    :column="editForeignKeyContext.column"
    :foreign-key="editForeignKeyContext.foreignKey"
    @close="state.showEditColumnForeignKeyModal = false"
  />

  <Drawer
    :show="state.showSchemaTemplateDrawer"
    @close="state.showSchemaTemplateDrawer = false"
  >
    <DrawerContent :title="$t('schema-template.field-template.self')">
      <div class="w-[calc(100vw-36rem)] min-w-[64rem] max-w-[calc(100vw-8rem)]">
        <FieldTemplates
          :engine="engine"
          :readonly="true"
          @apply="handleApplyColumnTemplate"
        />
      </div>
    </DrawerContent>
  </Drawer>
  <FeatureModal
    feature="bb.feature.schema-template"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, pull } from "lodash-es";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent } from "@/components/v2";
import { hasFeature, pushNotification } from "@/store/modules";
import {
  ColumnMetadata,
  ForeignKeyMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaTemplateSetting_FieldTemplate } from "@/types/proto/v1/setting_service";
import { arraySwap, instanceV1AllowsReorderColumns } from "@/utils";
import FieldTemplates from "@/views/SchemaTemplate/FieldTemplates.vue";
import EditColumnForeignKeyModal from "../Modals/EditColumnForeignKeyModal.vue";
import { useSchemaEditorContext } from "../context";
import {
  removeColumnFromAllForeignKeys,
  removeColumnPrimaryKey,
  upsertColumnPrimaryKey,
} from "../edit";
import { EditStatus, TableTabContext } from "../types";
import TableColumnEditor from "./TableColumnEditor";

const props = withDefaults(
  defineProps<{
    searchPattern: string;
  }>(),
  {
    searchPattern: "",
  }
);

interface LocalState {
  showEditColumnForeignKeyModal: boolean;
  showSchemaTemplateDrawer: boolean;
  showFeatureModal: boolean;
}

const { t } = useI18n();
const context = useSchemaEditorContext();
const {
  project,
  readonly,
  markEditStatus,
  removeEditStatus,
  getSchemaStatus,
  getTableStatus,
  getColumnStatus,
  upsertColumnConfig,
} = context;
const currentTab = computed(() => {
  return context.currentTab.value as TableTabContext;
});
const database = computed(() => currentTab.value.database);
const engine = computed(() => {
  return database.value.instanceEntity.engine;
});
const metadata = computed(() => {
  return currentTab.value.metadata;
});
const state = reactive<LocalState>({
  showEditColumnForeignKeyModal: false,
  showSchemaTemplateDrawer: false,
  showFeatureModal: false,
});
const table = computed(() => {
  return currentTab.value.metadata.table;
});

const editForeignKeyContext = ref<{
  column: ColumnMetadata;
  foreignKey: ForeignKeyMetadata | undefined;
}>();

const metadataForColumn = (column: ColumnMetadata) => {
  return {
    ...currentTab.value.metadata,
    column,
  };
};
const statusForSchema = () => {
  const { metadata } = currentTab.value;
  return getSchemaStatus(database.value, {
    database: metadata.database,
    schema: metadata.schema,
  });
};
const statusForTable = () => {
  const { metadata } = currentTab.value;
  return getTableStatus(database.value, {
    database: metadata.database,
    schema: metadata.schema,
    table: metadata.table,
  });
};
const statusForColumn = (column: ColumnMetadata) => {
  return getColumnStatus(database.value, metadataForColumn(column));
};
const markColumnStatus = (column: ColumnMetadata, status: EditStatus) => {
  const { metadata } = currentTab.value;
  markEditStatus(database.value, { ...metadata, column }, status);
};

const isDroppedSchema = computed(() => {
  return statusForSchema() === "dropped";
});

const isDroppedTable = computed(() => {
  return statusForTable() === "dropped";
});

const getColumnItemComputedClassList = (column: ColumnMetadata): string => {
  return statusForColumn(column);
};

const isDroppedColumn = (column: ColumnMetadata): boolean => {
  return statusForColumn(column) === "dropped";
};

const disableChangeTable = computed(() => {
  return isDroppedSchema.value || isDroppedTable.value;
});

const allowReorderColumns = computed(() => {
  if (props.searchPattern.trim().length !== 0) {
    // The column keyword filter will break the original indexes of columns
    return false;
  }

  const status = statusForTable();
  return instanceV1AllowsReorderColumns(engine.value) && status === "created";
});

const disableAlterColumn = (column: ColumnMetadata): boolean => {
  return (
    isDroppedSchema.value || isDroppedTable.value || isDroppedColumn(column)
  );
};

const setColumnPrimaryKey = (column: ColumnMetadata, isPrimaryKey: boolean) => {
  if (isPrimaryKey) {
    column.nullable = false;
    upsertColumnPrimaryKey(table.value, column.name);
  } else {
    removeColumnPrimaryKey(table.value, column.name);
  }
  markColumnStatus(column, "updated");
};

const handleAddColumn = () => {
  const column = ColumnMetadata.fromPartial({});
  table.value.columns.push(column);
  markColumnStatus(column, "created");
  // TODO: scroll to the new column and focus its name textbox
  // table.value.columnList.push(column);
  // nextTick(() => {
  //   const container = document.querySelector("#table-editor-container");
  //   (
  //     container?.querySelector(
  //       `.column-${column.id} .column-name-input`
  //     ) as HTMLInputElement
  //   )?.focus();
  // });
};

const handleApplyColumnTemplate = (
  template: SchemaTemplateSetting_FieldTemplate
) => {
  state.showSchemaTemplateDrawer = false;
  if (!hasFeature("bb.feature.schema-template")) {
    state.showFeatureModal = true;
    return;
  }
  if (!template.column) {
    return;
  }
  if (template.engine !== engine.value) {
    return;
  }
  const column = cloneDeep(template.column);
  table.value.columns.push(column);
  const { metadata } = currentTab.value;
  if (template.config) {
    upsertColumnConfig(
      database.value,
      {
        ...metadata,
        column,
      },
      (config) => {
        Object.assign(config, template.config);
      }
    );
  }
  markColumnStatus(column, "created");
};

const gotoForeignKeyReferencedTable = (
  column: ColumnMetadata,
  fk: ForeignKeyMetadata
) => {
  console.log("TODO: gotoForeignKeyReferencedTable", column, fk);
  // const fk = foreignKeyList.value.find(
  //   (fk) =>
  //     fk.columnIdList.find((columnId) => columnId === column.id) !== undefined
  // );
  // const index = fk?.columnIdList.findIndex(
  //   (columnId) => columnId === column.id
  // );
  // if (isUndefined(fk) || isUndefined(index) || index < 0) {
  //   return;
  // }

  // const referencedSchema = parentResource.value.schemaList.find(
  //   (schema) => schema.id === fk.referencedSchemaId
  // );
  // const referencedTable = referencedSchema?.tableList.find(
  //   (table) => table.id === fk.referencedTableId
  // );
  // if (!referencedTable) {
  //   return;
  // }
  // const referColumn = referencedTable?.columnList.find(
  //   (column) => column.id === fk.referencedColumnIdList[index]
  // );
  // if (!referencedSchema || !referencedTable || !referColumn) {
  //   return;
  // }

  // schemaEditorV1Store.addTab({
  //   id: generateUniqueTabId(),
  //   type: SchemaEditorTabType.TabForTable,
  //   parentName: currentTab.value.parentName,
  //   schemaId: referencedSchema.id,
  //   tableId: referencedTable.id,
  //   name: referencedTable.name,
  // });

  // nextTick(() => {
  //   const container = document.querySelector("#table-editor-container");
  //   const input = container?.querySelector(
  //     `.column-${referColumn.id} .column-name-input`
  //   ) as HTMLInputElement | undefined;
  //   if (input) {
  //     input.focus();
  //     scrollIntoView(input);
  //   }
  // });
};

const handleEditColumnForeignKey = (
  column: ColumnMetadata,
  foreignKey: ForeignKeyMetadata | undefined
) => {
  editForeignKeyContext.value = {
    column,
    foreignKey,
  };
  state.showEditColumnForeignKeyModal = true;
};

const handleDropColumn = (column: ColumnMetadata) => {
  // Disallow to drop the last column.
  const nonDroppedColumns = table.value.columns.filter((column) => {
    return statusForColumn(column) !== "dropped";
  });
  if (nonDroppedColumns.length === 1) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.cannot-drop-the-last-column"),
    });
    return;
  }
  const status = statusForColumn(column);
  if (status === "created") {
    pull(table.value.columns, column);
    table.value.columns = table.value.columns.filter((col) => col !== column);

    removeColumnPrimaryKey(table.value, column.name);
    removeColumnFromAllForeignKeys(table.value, column.name);
  } else {
    markColumnStatus(column, "dropped");
  }
};

const handleRestoreColumn = (column: ColumnMetadata) => {
  if (statusForColumn(column) === "created") {
    return;
  }
  removeEditStatus(
    database.value,
    metadataForColumn(column),
    /* recursive */ false
  );
};

const handleReorderColumn = (
  column: ColumnMetadata,
  index: number,
  delta: -1 | 1
) => {
  const target = index + delta;
  const { columns } = table.value;
  if (target < 0) return;
  if (target >= columns.length) return;
  arraySwap(columns, index, target);
};
</script>
