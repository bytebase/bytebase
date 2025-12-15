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
        :show-database-catalog-column="false"
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
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { cloneDeep, head } from "lodash-es";
import { ArrowLeftIcon, PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { IndexIcon, TablePartitionIcon } from "@/components/Icon";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  ColumnMetadataSchema,
  DatabaseMetadataSchema,
  IndexMetadataSchema,
  TablePartitionMetadata_Type,
  TablePartitionMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { instanceV1AllowsReorderColumns, randomString } from "@/utils";
import { useSchemaEditorContext } from "../context";
import EditColumnForeignKeyModal from "../Modals/EditColumnForeignKeyModal.vue";
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
  showFeatureModal: boolean;
}

const {
  readonly,
  events,
  options,
  addTab,
  markEditStatus,
  getSchemaStatus,
  getTableStatus,
  getColumnStatus,
  queuePendingScrollToColumn,
  selectionEnabled,
} = useSchemaEditorContext();

const engine = computed((): Engine => {
  return props.db.instanceResource.engine;
});
const state = reactive<LocalState>({
  mode: "COLUMNS",
  showEditColumnForeignKeyModal: false,
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

const mocked = computed(() => {
  const { db, database, schema, table } = props;

  const mockedTable = cloneDeep(table);
  mockedTable.columns = mockedTable.columns.filter((column) => {
    if (!column.name) {
      return false;
    }
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
  return { metadata: mockedDatabase };
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
