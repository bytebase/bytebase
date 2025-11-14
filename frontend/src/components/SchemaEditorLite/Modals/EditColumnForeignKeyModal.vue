<template>
  <BBModal
    :title="$t('schema-editor.foreign-key.edit')"
    class="shadow-inner outline-solid outline-gray-200"
    @close="dismissModal"
  >
    <div>
      <p class="textlabel mb-2">
        {{ $t("database.foreign-key.reference") }}
        <span class="textinfolabel"> ({{ referenceTips }}) </span>
      </p>
      <NInputGroup>
        <NSelect
          v-if="shouldShowSchemaSelector"
          v-model:value="state.referencedSchemaName"
          :options="referenceSelectOptions"
          :placeholder="$t('schema-editor.schema.select')"
          :filterable="true"
          @update:value="
            () => {
              state.referencedTableName = null;
              state.referencedColumnName = null;
            }
          "
        />
        <NSelect
          v-model:value="state.referencedTableName"
          :options="referencedTableOptions"
          :placeholder="$t('schema-editor.table.select')"
          :filterable="true"
          @update:value="state.referencedColumnName = null"
        />
        <NSelect
          v-model:value="state.referencedColumnName"
          :options="referencedColumnOptions"
          :placeholder="$t('schema-editor.column.select')"
          :filterable="true"
        />
      </NInputGroup>
    </div>
    <div class="mt-4">
      <p class="textlabel mb-2">{{ $t("common.name") }}</p>
      <NInput
        v-model:value="state.foreignKeyName"
        :placeholder="$t('schema-editor.foreign-key.name-description')"
      />
    </div>

    <div class="w-full flex items-center justify-between mt-4">
      <div class="flex flex-row items-center justify-start">
        <NButton
          v-if="foreignKey !== undefined"
          type="error"
          text
          @click="handleRemove"
        >
          <template #icon>
            <TrashIcon class="w-4 h-auto" />
          </template>
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
import { create } from "@bufbuild/protobuf";
import { TrashIcon } from "lucide-vue-next";
import type { SelectOption } from "naive-ui";
import { NButton, NInput, NInputGroup, NSelect } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { BBModal } from "@/bbkit";
import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { ForeignKeyMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { hasSchemaProperty } from "@/utils";
import { useSchemaEditorContext } from "../context";
import {
  removeColumnFromForeignKey,
  upsertColumnFromForeignKey,
} from "../edit";

interface LocalState {
  referencedSchemaName: string;
  referencedTableName: string | null;
  referencedColumnName: string | null;
  foreignKeyName: string;
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
  foreignKeyName: "",
});
const engine = computed(() => {
  return props.database.instanceResource.engine;
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
  return state.foreignKeyName !== "" && referencedColumn.value !== undefined;
});
const shouldShowSchemaSelector = computed(() => {
  return hasSchemaProperty(engine.value);
});
const referenceTips = computed(() => {
  return shouldShowSchemaSelector.value
    ? "Schema.Table.Column"
    : "Table.Column";
});

onMounted(() => {
  const { foreignKey } = props;
  if (!foreignKey) return;
  const position = foreignKey.columns.indexOf(props.column.name);
  if (position < 0) return;
  state.referencedSchemaName = foreignKey.referencedSchema;
  state.referencedTableName = foreignKey.referencedTable;
  state.referencedColumnName = foreignKey.referencedColumns[position];
  state.foreignKeyName = foreignKey.name;
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
  if (state.foreignKeyName === "") {
    return;
  }
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
    foreignKey.name = state.foreignKeyName;
    upsertColumnFromForeignKey(foreignKey, column.name, refed.column.name);
  } else {
    const fk = create(ForeignKeyMetadataSchema, {
      name: state.foreignKeyName,
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
