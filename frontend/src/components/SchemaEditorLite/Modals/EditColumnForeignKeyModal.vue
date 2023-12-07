<template>
  <BBModal
    :title="$t('schema-editor.edit-foreign-key')"
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72 text-xs font-mono">
      {{ state }}
    </div>
    <div v-if="shouldShowSchemaSelector" class="w-72">
      <p class="mb-2">{{ $t("schema-editor.select-reference-schema") }}</p>
      <NSelect
        v-model:value="state.referencedSchemaName"
        :options="referenceSelectOptions"
        :placeholder="$t('schema-editor.schema.select')"
        @update:value="
          () => {
            state.referencedTableName = null;
            state.referencedColumnName = null;
          }
        "
      />
    </div>
    <div class="w-72">
      <p class="mt-4 mb-2">{{ $t("schema-editor.select-reference-table") }}</p>
      <NSelect
        v-model:value="state.referencedTableName"
        :options="referencedTableOptions"
        :placeholder="$t('schema-editor.table.select')"
        @update:value="state.referencedColumnName = null"
      />
    </div>
    <div class="w-72">
      <p class="mt-4 mb-2">{{ $t("schema-editor.select-reference-column") }}</p>
      <NSelect
        v-model:value="state.referencedColumnName"
        :options="referencedColumnOptions"
        :placeholder="$t('schema-editor.column.select')"
      />
    </div>

    <div class="w-full flex items-center justify-between mt-6">
      <div class="flex flex-row items-center justify-start">
        <NButton
          v-if="foreignKey !== undefined"
          type="error"
          @click="handleRemove"
        >
          {{ $t("common.delete") }}
        </NButton>
      </div>
      <div class="flex flex-row items-center justify-end gap-x-2">
        <NButton @click="dismissModal">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!allowConfirm"
          @click="handleConfirm"
        >
          {{ $t("common.save") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { NSelect, SelectOption } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { BBModal } from "@/bbkit";
import { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { randomString } from "@/utils";
import { useSchemaEditorContext } from "../context";
import {
  removeColumnFromForeignKey,
  upsertColumnFromForeignKey,
} from "../edit";

interface LocalState {
  referencedSchemaName: string;
  referencedTableName: string | null;
  referencedColumnName: string | null;
}

type SchemaSelectOption = SelectOption & {
  label: string;
  value: string;
  schema: SchemaMetadata;
};
type TableSelectOption = SchemaSelectOption & {
  table: TableMetadata;
};
type ColumnSelectOption = TableSelectOption & {
  column: ColumnMetadata;
};

const props = defineProps<{
  database: ComposedDatabase;
  metadata: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  column: ColumnMetadata;
  foreignKey?: ForeignKeyMetadata;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { markEditStatus, getColumnStatus } = useSchemaEditorContext();
const state = reactive<LocalState>({
  referencedSchemaName: props.schema.name,
  referencedTableName: null,
  referencedColumnName: null,
});
const engine = computed(() => {
  return props.database.instanceEntity.engine;
});

const referencedSchema = computed(() => {
  return props.metadata.schemas.find(
    (s) => s.name === state.referencedSchemaName
  );
});
const referencedTable = computed(() => {
  const schema = referencedSchema.value;
  if (!schema) return undefined;
  if (!state.referencedTableName) return undefined;
  const table = schema.tables.find((t) => t.name === state.referencedTableName);
  if (!table) return undefined;
  return { schema, table };
});
const referencedColumn = computed(() => {
  if (!referencedTable.value) return undefined;
  const { schema, table } = referencedTable.value;
  if (!state.referencedColumnName) return undefined;
  const column = table?.columns.find(
    (c) => c.name === state.referencedColumnName
  );
  if (!column) return undefined;
  return { schema, table, column };
});
const referenceSelectOptions = computed(() => {
  return props.metadata.schemas.map<SchemaSelectOption>((schema) => ({
    label: schema.name,
    value: schema.name,
    schema,
  }));
});
const referencedTableOptions = computed(() => {
  const schema = referencedSchema.value;
  if (!schema) return [];
  return schema.tables.map<TableSelectOption>((table) => ({
    label: table.name,
    value: table.name,
    schema,
    table,
  }));
});
const referencedColumnOptions = computed(() => {
  if (!referencedTable.value) return [];
  const { schema, table } = referencedTable.value;
  return table.columns.map<ColumnSelectOption>((column) => ({
    label: column.name,
    value: column.name,
    schema,
    table,
    column,
  }));
});
const allowConfirm = computed(() => {
  return referencedColumn.value !== undefined;
});

const shouldShowSchemaSelector = computed(() => {
  return engine.value === Engine.POSTGRES;
});

onMounted(() => {
  const { foreignKey } = props;
  if (!foreignKey) return;
  const position = foreignKey.columns.indexOf(props.column.name);
  if (position < 0) return;
  state.referencedSchemaName = foreignKey.referencedSchema;
  state.referencedTableName = foreignKey.referencedTable;
  state.referencedColumnName = foreignKey.referencedColumns[position];
});

const metadataForColumn = () => {
  return {
    database: props.metadata,
    schema: props.schema,
    table: props.table,
    column: props.column,
  };
};
const statusForColumn = () => {
  return getColumnStatus(props.database, metadataForColumn());
};
const markColumnUpdatedIfNeeded = () => {
  if (statusForColumn() !== "created") {
    markEditStatus(props.database, metadataForColumn(), "updated");
  }
};

const handleRemove = async () => {
  const { foreignKey } = props;
  if (!foreignKey) return;

  removeColumnFromForeignKey(props.table, foreignKey, props.column.name);

  markColumnUpdatedIfNeeded();
  dismissModal();
};

const handleConfirm = async () => {
  if (!referencedColumn.value) {
    return;
  }

  const { foreignKey, table, column } = props;
  const refed = referencedColumn.value;
  if (foreignKey) {
    // upsert column in existed foreignKey
    if (
      foreignKey.referencedSchema !== refed.schema.name ||
      foreignKey.referencedTable !== refed.table.name
    ) {
      // refed schema/table changed, clear the ref column list
      foreignKey.referencedSchema = refed.schema.name;
      foreignKey.referencedTable = refed.table.name;
      foreignKey.columns = [];
      foreignKey.referencedColumns = [];
    }
    upsertColumnFromForeignKey(foreignKey, column.name, refed.column.name);
  } else {
    const fk = ForeignKeyMetadata.fromPartial({
      name: `${table.name}-fk-${randomString(8).toLowerCase()}`,
      columns: [props.column.name],
      referencedSchema: refed.schema.name,
      referencedTable: refed.table.name,
      referencedColumns: [refed.column.name],
    });
    table.foreignKeys.push(fk);
  }
  markColumnUpdatedIfNeeded();
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
