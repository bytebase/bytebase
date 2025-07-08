<template>
  <div
    class="flex flex-col pt-2 gap-y-2 w-full h-full overflow-y-hidden"
    v-bind="$attrs"
  >
    <div class="w-full flex flex-row justify-between items-center px-2">
      <div class="w-full flex justify-start items-center gap-x-2">
        <slot v-if="state.mode === 'COLUMNS'" name="toolbar-prefix" />

        <template
          v-if="state.mode === 'INDEXES' || state.mode === 'PARTITIONS'"
        >
          <NButton size="small" @click="state.mode = 'COLUMNS'">
            <ArrowLeftIcon class="w-4 h-4" />
          </NButton>
          <template v-if="!readonly">
            <NButton
              v-if="state.mode === 'INDEXES'"
              size="small"
              @click="handleAddIndex"
            >
              <PlusIcon class="w-4 h-4 mr-1" />
              {{ $t("common.add") }}
            </NButton>
            <NButton
              v-if="state.mode === 'PARTITIONS'"
              size="small"
              @click="handleAddPartition"
            >
              <PlusIcon class="w-4 h-4 mr-1" />
              {{ $t("common.add") }}
            </NButton>
          </template>
        </template>
        <template v-if="state.mode === 'COLUMNS'">
          <template v-if="!readonly">
            <NButton
              size="small"
              :disabled="disableChangeTable"
              @click="handleAddColumn"
            >
              <PlusIcon class="w-4 h-auto mr-1 text-gray-400" />
              {{ $t("schema-editor.actions.add-column") }}
            </NButton>
            <NButton
              size="small"
              :disabled="disableChangeTable"
              @click="state.showSchemaTemplateDrawer = true"
            >
              <template #icon>
                <PlusIcon class="w-4 h-auto mr-1 text-gray-400" />
              </template>
              {{ $t("schema-editor.actions.add-from-template") }}
            </NButton>
          </template>
          <NButton
            v-if="showIndexes"
            size="small"
            :disabled="disableChangeTable"
            @click="state.mode = 'INDEXES'"
          >
            <IndexIcon class="mr-1" />
            {{
              readonly
                ? $t("schema-editor.index.indexes")
                : $t("schema-editor.index.edit-indexes")
            }}
          </NButton>
          <NButton
            v-if="showPartitions"
            size="small"
            :disabled="disableChangeTable"
            @click="state.mode = 'PARTITIONS'"
          >
            <TablePartitionIcon class="w-3 h-3 mr-1" />
            {{
              readonly
                ? $t("schema-editor.table-partition.partitions")
                : $t("schema-editor.table-partition.edit-partitions")
            }}
          </NButton>
        </template>
      </div>
      <div class="text-sm flex flex-row items-center justify-end gap-x-2">
        <div
          v-if="selectionEnabled"
          class="text-sm flex flex-row items-center gap-x-2 h-[28px] whitespace-nowrap"
        >
          <span class="text-main">
            {{ $t("branch.select-tables-to-rollout") }}
          </span>
          <ColumnSelectionSummary
            :db="db"
            :metadata="{
              database,
              schema,
              table,
            }"
          />
        </div>
      </div>
    </div>

    <div class="flex-1 overflow-y-hidden px-2">
      <TableColumnEditor
        :show="state.mode === 'COLUMNS'"
        :readonly="readonly"
        :show-foreign-key="true"
        :db="db"
        :database="database"
        :schema="schema"
        :table="table"
        :engine="engine"
        :disable-change-table="disableChangeTable"
        :allow-change-primary-keys="allowChangePrimaryKeys"
        :allow-reorder-columns="allowReorderColumns"
        :filter-column="
          (column: ColumnMetadata) =>
            column.name.includes(props.searchPattern.trim())
        "
        :disable-alter-column="disableAlterColumn"
        :get-column-item-computed-class-list="getColumnItemComputedClassList"
        :show-database-catalog-column="false"
        @drop="handleDropColumn"
        @restore="handleRestoreColumn"
        @reorder="handleReorderColumn"
        @primary-key-set="setColumnPrimaryKey"
        @foreign-key-edit="handleEditColumnForeignKey"
        @foreign-key-click="gotoForeignKeyReferencedTable"
      />
      <IndexesEditor
        :show="state.mode === 'INDEXES'"
        :readonly="readonly"
        :db="db"
        :database="database"
        :schema="schema"
        :table="table"
        @update="markTableStatus('updated')"
      />
      <PartitionsEditor
        :show="state.mode === 'PARTITIONS'"
        :readonly="readonly"
        :db="db"
        :database="database"
        :schema="schema"
        :table="table"
        @update="markTableStatus('updated')"
      />
    </div>

    <PreviewPane
      :db="db"
      :database="database"
      :schema="schema"
      :title="$t('schema-editor.preview-schema-text')"
      :mocked="mocked"
    />
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
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { cloneDeep, head, pull } from "lodash-es";
import { ArrowLeftIcon, PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { IndexIcon, TablePartitionIcon } from "@/components/Icon";
import { Drawer, DrawerContent } from "@/components/v2";
import { pushNotification } from "@/store/modules";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DatabaseCatalogSchema,
  SchemaCatalogSchema,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  TablePartitionMetadata_Type,
  ColumnMetadataSchema,
  IndexMetadataSchema,
  TablePartitionMetadataSchema,
  DatabaseMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { ColumnMetadata as NewColumnMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { SchemaTemplateSetting_FieldTemplate } from "@/types/proto-es/v1/setting_service_pb";
import {
  arraySwap,
  instanceV1AllowsReorderColumns,
  randomString,
} from "@/utils";
import FieldTemplates from "@/views/SchemaTemplate/FieldTemplates.vue";
import EditColumnForeignKeyModal from "../Modals/EditColumnForeignKeyModal.vue";
import { useSchemaEditorContext } from "../context";
import {
  removeColumnFromAllForeignKeys,
  removeColumnPrimaryKey,
  upsertColumnPrimaryKey,
} from "../edit";
import {
  engineSupportsEditIndexes,
  engineSupportsEditTablePartitions,
} from "../spec";
import type { EditStatus } from "../types";
import ColumnSelectionSummary from "./ColumnSelectionSummary.vue";
import IndexesEditor from "./IndexesEditor";
import PartitionsEditor from "./PartitionsEditor";
import PreviewPane from "./PreviewPane.vue";
import TableColumnEditor from "./TableColumnEditor";

type EditMode = "COLUMNS" | "INDEXES" | "PARTITIONS";

const props = withDefaults(
  defineProps<{
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
    searchPattern?: string;
  }>(),
  {
    searchPattern: "",
  }
);

interface LocalState {
  mode: EditMode;
  showEditColumnForeignKeyModal: boolean;
  showSchemaTemplateDrawer: boolean;
  showFeatureModal: boolean;
}

const { t } = useI18n();
const {
  readonly,
  events,
  options,
  addTab,
  markEditStatus,
  removeEditStatus,
  getSchemaStatus,
  getTableStatus,
  getColumnStatus,
  getDatabaseCatalog,
  removeColumnCatalog,
  upsertColumnCatalog,
  queuePendingScrollToColumn,
  selectionEnabled,
} = useSchemaEditorContext();

// Conversion function for ColumnMetadata at service boundaries
const convertNewColumnToOld = (
  newColumn: NewColumnMetadata
): ColumnMetadata => {
  return create(ColumnMetadataSchema, {
    name: newColumn.name,
    position: newColumn.position,
    hasDefault: newColumn.hasDefault,
    defaultNull: newColumn.defaultNull,
    defaultString: newColumn.defaultString,
    defaultExpression: newColumn.defaultExpression,
    onUpdate: newColumn.onUpdate,
    nullable: newColumn.nullable,
    type: newColumn.type,
    characterSet: newColumn.characterSet,
    collation: newColumn.collation,
    userComment: newColumn.userComment,
    comment: newColumn.comment,
    // classification, labels, effectiveMaskingLevel are not available in old proto types
  });
};
const engine = computed((): Engine => {
  return props.db.instanceResource.engine;
});
const state = reactive<LocalState>({
  mode: "COLUMNS",
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
    schema: props.schema,
  });
};
const statusForTable = () => {
  return getTableStatus(props.db, {
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

const allowChangePrimaryKeys = computed(() => {
  return statusForTable() === "created";
});

const allowReorderColumns = computed(() => {
  if (props.searchPattern.trim().length !== 0) {
    // The column keyword filter will break the original indexes of columns
    return false;
  }

  const status = statusForTable();
  return instanceV1AllowsReorderColumns(engine.value) && status === "created";
});

const showIndexes = computed(() => {
  if (options?.value.forceShowIndexes) {
    return props.table.indexes.length > 0;
  }
  return engineSupportsEditIndexes(engine.value);
});

const showPartitions = computed(() => {
  if (options?.value.forceShowPartitions) {
    return props.table.partitions.length > 0;
  }
  return engineSupportsEditTablePartitions(engine.value);
});

const disableAlterColumn = (column: ColumnMetadata): boolean => {
  return (
    isDroppedSchema.value || isDroppedTable.value || isDroppedColumn(column)
  );
};

const setColumnPrimaryKey = (column: ColumnMetadata, isPrimaryKey: boolean) => {
  if (isPrimaryKey) {
    column.nullable = false;
    upsertColumnPrimaryKey(engine.value, props.table, column.name);
  } else {
    removeColumnPrimaryKey(props.table, column.name);
  }
  markColumnStatus(column, "updated");
};

const handleAddColumn = () => {
  const column = create(ColumnMetadataSchema, {});
  /* eslint-disable-next-line vue/no-mutating-props */
  props.table.columns.push(column);
  markColumnStatus(column, "created");

  queuePendingScrollToColumn({
    db: props.db,
    metadata: {
      database: props.database,
      schema: props.schema,
      table: props.table,
      column,
    },
  });

  events.emit("rebuild-tree", {
    openFirstChild: false,
  });
};

const handleApplyColumnTemplate = (
  template: SchemaTemplateSetting_FieldTemplate
) => {
  state.showSchemaTemplateDrawer = false;
  if (!template.column) {
    return;
  }
  if (template.engine !== engine.value) {
    return;
  }
  const column = convertNewColumnToOld(template.column);
  /* eslint-disable-next-line vue/no-mutating-props */
  props.table.columns.push(column);
  if (template.catalog) {
    upsertColumnCatalog(
      {
        database: props.db.name,
        schema: props.schema.name,
        table: props.table.name,
        column: template.column.name,
      },
      (catalog) => {
        Object.assign(catalog, template.catalog);
      }
    );
  }
  markColumnStatus(column, "created");
  queuePendingScrollToColumn({
    db: props.db,
    metadata: {
      database: props.database,
      schema: props.schema,
      table: props.table,
      column,
    },
  });
  events.emit("rebuild-tree", {
    openFirstChild: false,
  });
};
const handleAddIndex = () => {
  // eslint-disable-next-line vue/no-mutating-props
  props.table.indexes.push(
    create(IndexMetadataSchema, {
      name: `${props.table.name}_index_${randomString(8).toLowerCase()}`,
    })
  );
  markTableStatus("updated");
};
const handleAddPartition = () => {
  const first = head(props.table.partitions);
  const partition = create(TablePartitionMetadataSchema, {
    type: first?.type ?? TablePartitionMetadata_Type.HASH,
    expression: first?.expression ?? "",
  });
  // eslint-disable-next-line vue/no-mutating-props
  props.table.partitions.push(partition);
  markEditStatus(
    props.db,
    {
      schema: props.schema,
      table: props.table,
      partition,
    },
    "created"
  );
  markTableStatus("updated");
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

  queuePendingScrollToColumn({
    db: props.db,
    metadata: {
      database: props.database,
      schema: referencedSchema,
      table: referencedTable,
      column: referencedColumn,
    },
  });
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
    removeColumnCatalog({
      database: props.db.name,
      schema: props.schema.name,
      table: props.table.name,
      column: column.name,
    });
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

const mocked = computed(() => {
  const { db, database, schema, table } = props;
  const databaseCatalog = getDatabaseCatalog(db.name);

  const mockedTable = cloneDeep(table);
  mockedTable.columns = mockedTable.columns.filter((column) => {
    const status = getColumnStatus(db, { schema, table, column });
    return status !== "dropped";
  });
  const mockedDatabase = create(DatabaseMetadataSchema, {
    name: database.name,
    characterSet: database.characterSet,
    collation: database.collation,
    schemas: [
      {
        name: schema.name,
        tables: [mockedTable],
      },
    ],
  });
  const mockedCatalog = create(DatabaseCatalogSchema, {
    name: database.name,
  });
  const schemaCatalog = databaseCatalog.schemas.find(
    (sc) => sc.name === schema.name
  );
  const tableCatalog = schemaCatalog?.tables.find(
    (tc) => tc.name === table.name
  );
  if (schemaCatalog && tableCatalog) {
    mockedCatalog.schemas = [
      create(SchemaCatalogSchema, {
        ...schemaCatalog,
        tables: [cloneDeep(tableCatalog)],
      }),
    ];
  }
  return { metadata: mockedDatabase, catalog: mockedCatalog };
});

const markTableStatus = (status: EditStatus) => {
  const oldStatus = statusForTable();
  if (
    (oldStatus === "created" || oldStatus === "dropped") &&
    status === "updated"
  ) {
    return;
  }
  markEditStatus(
    props.db,
    {
      schema: props.schema,
      table: props.table,
    },
    status
  );
};
</script>
