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
        :foreign-key-list="foreignKeyList"
        :classification-config-id="project.dataClassificationConfigId"
        :disable-change-table="disableChangeTable"
        :allow-reorder-columns="allowReorderColumns"
        :filter-column="(column: Column) => column.name.includes(props.searchPattern.trim())"
        :disable-alter-column="disableAlterColumn"
        :get-referenced-foreign-key-name="getReferencedForeignKeyName"
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
    v-if="state.showEditColumnForeignKeyModal && editForeignKeyColumn"
    :parent-name="currentTab.parentName"
    :schema-id="schema.id"
    :table-id="table.id"
    :column-id="editForeignKeyColumn.id"
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
import { isUndefined, flatten } from "lodash-es";
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, nextTick, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  hasFeature,
  generateUniqueTabId,
  useSchemaEditorV1Store,
  pushNotification,
} from "@/store/modules";
import { Engine } from "@/types/proto/v1/common";
import { ColumnMetadata } from "@/types/proto/v1/database_service";
import { SchemaTemplateSetting_FieldTemplate } from "@/types/proto/v1/setting_service";
import {
  Column,
  Table,
  Schema,
  convertColumnMetadataToColumn,
  ForeignKey,
  SchemaEditorTabType,
} from "@/types/v1/schemaEditor";
import { TableTabContext } from "@/types/v1/schemaEditor";
import { arraySwap } from "@/utils";
import FieldTemplates from "@/views/SchemaTemplate/FieldTemplates.vue";
import EditColumnForeignKeyModal from "../Modals/EditColumnForeignKeyModal.vue";
import { isColumnChanged } from "../utils";
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
const schemaEditorV1Store = useSchemaEditorV1Store();
const currentTab = computed(
  () => schemaEditorV1Store.currentTab as TableTabContext
);

const parentResouce = computed(() => {
  return schemaEditorV1Store.resourceMap[schemaEditorV1Store.resourceType].get(
    currentTab.value.parentName
  )!;
});
const engine = computed(() => {
  return schemaEditorV1Store.getCurrentEngine(currentTab.value.parentName);
});
const readonly = computed(() => schemaEditorV1Store.readonly);
const project = computed(() => schemaEditorV1Store.project);
const state = reactive<LocalState>({
  showEditColumnForeignKeyModal: false,
  showSchemaTemplateDrawer: false,
  showFeatureModal: false,
});

const schema = computed(() => {
  return schemaEditorV1Store.getSchema(
    currentTab.value.parentName,
    currentTab.value.schemaId
  ) as Schema;
});
const table = computed(
  () =>
    schema.value.tableList.find(
      (table) => table.id === currentTab.value.tableId
    ) as Table
);
const foreignKeyList = computed(() => {
  return table.value.foreignKeyList.filter(
    (pk) => pk.tableId === currentTab.value.tableId
  ) as ForeignKey[];
});

const editForeignKeyColumn = ref<Column>();

const isDroppedSchema = computed(() => {
  return schema.value.status === "dropped";
});

const isDroppedTable = computed(() => {
  return table.value.status === "dropped";
});

const getColumnItemComputedClassList = (column: Column): string => {
  if (column.status === "dropped") {
    return "dropped";
  } else if (column.status === "created") {
    return "created";
  } else if (
    isColumnChanged(
      currentTab.value.parentName,
      currentTab.value.schemaId,
      currentTab.value.tableId,
      column.id
    )
  ) {
    return "updated";
  }
  return "";
};

const checkColumnHasForeignKey = (column: Column): boolean => {
  const columnIdList = flatten(
    foreignKeyList.value.map((fk) => fk.columnIdList)
  );
  return columnIdList.includes(column.id);
};

const getReferencedForeignKeyName = (column: Column) => {
  if (!checkColumnHasForeignKey(column)) {
    return "";
  }
  const fk = foreignKeyList.value.find(
    (fk) =>
      fk.columnIdList.find((columnId) => columnId === column.id) !== undefined
  );
  const index = fk?.columnIdList.findIndex(
    (columnId) => columnId === column.id
  );

  if (isUndefined(fk) || isUndefined(index) || index < 0) {
    return "";
  }
  const referencedSchema = parentResouce.value.schemaList.find(
    (schema) => schema.id === fk.referencedSchemaId
  );
  const referencedTable = referencedSchema?.tableList.find(
    (table) => table.id === fk.referencedTableId
  );
  if (!referencedTable) {
    return "";
  }
  const referColumn = referencedTable.columnList.find(
    (column) => column.id === fk.referencedColumnIdList[index]
  );
  if (engine.value === Engine.MYSQL) {
    return `${referencedTable.name}(${referColumn?.name})`;
  } else {
    return `${referencedSchema?.name}.${referencedTable.name}(${referColumn?.name})`;
  }
};

const isDroppedColumn = (column: Column): boolean => {
  return column.status === "dropped";
};

const disableChangeTable = computed(() => {
  return isDroppedSchema.value || isDroppedTable.value;
});

const allowReorderColumns = computed(() => {
  return (
    table.value.status === "created" && props.searchPattern.trim().length === 0
  );
});

const disableAlterColumn = (column: Column): boolean => {
  return (
    isDroppedSchema.value || isDroppedTable.value || isDroppedColumn(column)
  );
};

const setColumnPrimaryKey = (column: Column, isPrimaryKey: boolean) => {
  if (isPrimaryKey) {
    column.nullable = false;
    table.value.primaryKey.columnIdList.push(column.id);
  } else {
    table.value.primaryKey.columnIdList =
      table.value.primaryKey.columnIdList.filter(
        (columnId) => columnId !== column.id
      );
  }
};

const handleAddColumn = () => {
  const column = convertColumnMetadataToColumn(
    ColumnMetadata.fromPartial({}),
    "created"
  );
  table.value.columnList.push(column);
  nextTick(() => {
    const container = document.querySelector("#table-editor-container");
    (
      container?.querySelector(
        `.column-${column.id} .column-name-input`
      ) as HTMLInputElement
    )?.focus();
  });
};

const handleApplyColumnTemplate = (
  template: SchemaTemplateSetting_FieldTemplate
) => {
  state.showSchemaTemplateDrawer = false;
  if (!hasFeature("bb.feature.schema-template")) {
    state.showFeatureModal = true;
    return;
  }
  if (template.engine !== engine.value || !template.column) {
    return;
  }
  const column = convertColumnMetadataToColumn(
    template.column,
    "created",
    template.config
  );
  table.value.columnList.push(column);
};

const gotoForeignKeyReferencedTable = (column: Column) => {
  if (!checkColumnHasForeignKey(column)) {
    return;
  }
  const fk = foreignKeyList.value.find(
    (fk) =>
      fk.columnIdList.find((columnId) => columnId === column.id) !== undefined
  );
  const index = fk?.columnIdList.findIndex(
    (columnId) => columnId === column.id
  );
  if (isUndefined(fk) || isUndefined(index) || index < 0) {
    return;
  }

  const referencedSchema = parentResouce.value.schemaList.find(
    (schema) => schema.id === fk.referencedSchemaId
  );
  const referencedTable = referencedSchema?.tableList.find(
    (table) => table.id === fk.referencedTableId
  );
  if (!referencedTable) {
    return;
  }
  const referColumn = referencedTable?.columnList.find(
    (column) => column.id === fk.referencedColumnIdList[index]
  );
  if (!referencedSchema || !referencedTable || !referColumn) {
    return;
  }

  schemaEditorV1Store.addTab({
    id: generateUniqueTabId(),
    type: SchemaEditorTabType.TabForTable,
    parentName: currentTab.value.parentName,
    schemaId: referencedSchema.id,
    tableId: referencedTable.id,
    name: referencedTable.name,
  });

  nextTick(() => {
    const container = document.querySelector("#table-editor-container");
    const input = container?.querySelector(
      `.column-${referColumn.id} .column-name-input`
    ) as HTMLInputElement | undefined;
    if (input) {
      input.focus();
      scrollIntoView(input);
    }
  });
};

const handleEditColumnForeignKey = (column: Column) => {
  editForeignKeyColumn.value = column;
  state.showEditColumnForeignKeyModal = true;
};

const handleDropColumn = (column: Column) => {
  // Disallow to drop the last column.
  if (
    table.value.columnList.filter((column) => column.status !== "dropped")
      .length === 1
  ) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.cannot-drop-the-last-column"),
    });
    return;
  }

  if (column.status === "created") {
    table.value.columnList = table.value.columnList.filter(
      (item) => item !== column
    );
    table.value.primaryKey.columnIdList =
      table.value.primaryKey.columnIdList.filter(
        (columnId) => columnId !== column.id
      );

    const foreignKeyList = table.value.foreignKeyList.filter(
      (fk) => fk.tableId === currentTab.value.tableId
    );
    for (const foreignKey of foreignKeyList) {
      const columnRefIndex = foreignKey.columnIdList.findIndex(
        (columnId) => columnId === column.id
      );
      if (columnRefIndex > -1) {
        foreignKey.columnIdList.splice(columnRefIndex, 1);
        foreignKey.referencedColumnIdList.splice(columnRefIndex, 1);
      }
    }
  } else {
    column.status = "dropped";
  }
};

const handleRestoreColumn = (column: Column) => {
  if (column.status === "created") {
    return;
  }
  column.status = "normal";
};

const handleReorderColumn = (column: Column, index: number, delta: -1 | 1) => {
  const target = index + delta;
  const { columnList } = table.value;
  if (target < 0) return;
  if (target >= columnList.length) return;

  arraySwap(columnList, index, target);
};
</script>
