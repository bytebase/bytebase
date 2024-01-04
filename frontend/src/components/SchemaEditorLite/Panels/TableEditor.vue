<template>
  <div
    class="flex flex-col pt-2 gap-y-2 w-full h-full overflow-y-hidden px-1 -mx-1"
  >
    <div class="w-full flex flex-row justify-between items-center">
      <div class="w-full flex justify-start items-center gap-x-2">
        <template v-if="state.mode === 'INDEXES'">
          <NButton size="small" @click="state.mode = 'COLUMNS'">
            <ArrowLeftIcon class="w-4 h-4" />
          </NButton>
          <template v-if="!readonly">
            <NButton size="small" @click="handleAddIndex">
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
              <FeatureBadge feature="bb.feature.schema-template" />
              <PlusIcon class="w-4 h-auto mr-1 text-gray-400" />
              {{ $t("schema-editor.actions.add-from-template") }}
            </NButton>
          </template>
          <NButton
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

    <div class="flex-1 overflow-y-hidden">
      <TableColumnEditor
        :show="state.mode === 'COLUMNS'"
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
      <IndexesEditor
        :show="state.mode === 'INDEXES'"
        :readonly="readonly"
        :db="db"
        :database="database"
        :schema="schema"
        :table="table"
        @update="markTableStatus('updated')"
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
import { PlusIcon } from "lucide-vue-next";
import { ArrowLeftIcon } from "lucide-vue-next";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { IndexIcon } from "@/components/Icon";
import { Drawer, DrawerContent } from "@/components/v2";
import { hasFeature, pushNotification } from "@/store/modules";
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaTemplateSetting_FieldTemplate } from "@/types/proto/v1/setting_service";
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
import { EditStatus } from "../types";
import ColumnSelectionSummary from "./ColumnSelectionSummary.vue";
import IndexesEditor from "./IndexesEditor";
import TableColumnEditor from "./TableColumnEditor";

type EditMode = "COLUMNS" | "INDEXES";

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
  mode: EditMode;
  showEditColumnForeignKeyModal: boolean;
  showSchemaTemplateDrawer: boolean;
  showFeatureModal: boolean;
}

const { t } = useI18n();
const {
  project,
  readonly,
  events,
  addTab,
  markEditStatus,
  removeEditStatus,
  getSchemaStatus,
  getTableStatus,
  getColumnStatus,
  upsertColumnConfig,
  queuePendingScrollToColumn,
  selectionEnabled,
} = useSchemaEditorContext();
const engine = computed(() => {
  return props.db.instanceEntity.engine;
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
    IndexMetadata.fromPartial({
      name: `${props.table.name}-index-${randomString(8).toLowerCase()}`,
    })
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
      database: props.database,
      schema: props.schema,
      table: props.table,
    },
    status
  );
};
</script>
