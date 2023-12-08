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
        :db="db"
        :database="database"
        :schema="schema"
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
    :database="db"
    :metadata="database"
    :schema="schema"
    :table="table"
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
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
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
import { EditStatus } from "../types";
import TableColumnEditor from "./TableColumnEditor";

const props = withDefaults(
  defineProps<{
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
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
const {
  project,
  readonly,
  addTab,
  markEditStatus,
  removeEditStatus,
  getSchemaStatus,
  getTableStatus,
  getColumnStatus,
  upsertColumnConfig,
} = useSchemaEditorContext();
const engine = computed(() => {
  return props.db.instanceEntity.engine;
});
const state = reactive<LocalState>({
  showEditColumnForeignKeyModal: false,
  showSchemaTemplateDrawer: false,
  showFeatureModal: false,
});

const editForeignKeyContext = ref<{
  column: ColumnMetadata;
  foreignKey: ForeignKeyMetadata | undefined;
}>();

const metadataForColumn = (column: ColumnMetadata) => {
  return {
    database: props.database,
    schema: props.schema,
    table: props.table,
    column,
  };
};
const statusForSchema = () => {
  return getSchemaStatus(props.db, {
    database: props.database,
    schema: props.schema,
  });
};
const statusForTable = () => {
  return getTableStatus(props.db, {
    database: props.database,
    schema: props.schema,
    table: props.table,
  });
};
const statusForColumn = (column: ColumnMetadata) => {
  return getColumnStatus(props.db, metadataForColumn(column));
};
const markColumnStatus = (column: ColumnMetadata, status: EditStatus) => {
  markEditStatus(props.db, metadataForColumn(column), status);
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
    upsertColumnPrimaryKey(props.table, column.name);
  } else {
    removeColumnPrimaryKey(props.table, column.name);
  }
  markColumnStatus(column, "updated");
};

const handleAddColumn = () => {
  const column = ColumnMetadata.fromPartial({});
  /* eslint-disable-next-line vue/no-mutating-props */
  props.table.columns.push(column);
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
  /* eslint-disable-next-line vue/no-mutating-props */
  props.table.columns.push(column);
  if (template.config) {
    upsertColumnConfig(props.db, metadataForColumn(column), (config) => {
      Object.assign(config, template.config);
    });
  }
  markColumnStatus(column, "created");
};

const gotoForeignKeyReferencedTable = (
  column: ColumnMetadata,
  fk: ForeignKeyMetadata
) => {
  const position = fk.columns.indexOf(column.name);
  if (position < 0) {
    return;
  }
  const referencedColumnName = fk.referencedColumns[position];
  if (!referencedColumnName) {
    return;
  }
  const referencedSchema = props.database.schemas.find(
    (schema) => schema.name === fk.referencedSchema
  );
  if (!referencedSchema) {
    return;
  }
  const referencedTable = referencedSchema.tables.find(
    (table) => table.name === fk.referencedTable
  );
  if (!referencedTable) {
    return;
  }
  const referencedColumn = referencedTable.columns.find(
    (column) => column.name === referencedColumnName
  );
  if (!referencedColumn) {
    return;
  }

  addTab({
    type: "table",
    database: props.db,
    metadata: {
      database: props.database,
      schema: referencedSchema,
      table: referencedTable,
    },
  });

  // TODO: scroll column into view
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
  const { table } = props;
  // Disallow to drop the last column.
  const nonDroppedColumns = table.columns.filter((column) => {
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
    pull(table.columns, column);
    table.columns = table.columns.filter((col) => col !== column);

    removeColumnPrimaryKey(table, column.name);
    removeColumnFromAllForeignKeys(table, column.name);
  } else {
    markColumnStatus(column, "dropped");
  }
};

const handleRestoreColumn = (column: ColumnMetadata) => {
  if (statusForColumn(column) === "created") {
    return;
  }
  removeEditStatus(props.db, metadataForColumn(column), /* recursive */ false);
};

const handleReorderColumn = (
  column: ColumnMetadata,
  index: number,
  delta: -1 | 1
) => {
  const target = index + delta;
  const { columns } = props.table;
  if (target < 0) return;
  if (target >= columns.length) return;
  arraySwap(columns, index, target);
};
</script>
