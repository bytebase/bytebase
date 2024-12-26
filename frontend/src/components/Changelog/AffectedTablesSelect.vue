<template>
  <NSelect
    :value="selectedKeys"
    :options="affectedTableOptions"
    filterable
    multiple
    tag
    :placeholder="$t('change-history.select-affection-tables')"
    :render-label="renderLabel"
    @update:value="updateSelectedKey"
  />
</template>

<script setup lang="tsx">
import { NSelect } from "naive-ui";
import type { SelectOption } from "naive-ui";
import { computed } from "vue";
import { useDBSchemaV1Store } from "@/store";
import { type ComposedDatabase, type Table } from "@/types";

type AffectedTablesSelectOption = SelectOption & {
  table: Table;
  value: string;
};

const props = defineProps<{
  database: ComposedDatabase;
  tables?: Table[];
}>();

const emit = defineEmits<{
  (event: "update:tables", tables?: Table[]): void;
}>();

const dbSchemaStore = useDBSchemaV1Store();

const metadata = computed(() => {
  return dbSchemaStore.getDatabaseMetadata(props.database.name);
});

const affectedTables = computed((): Table[] => {
  return metadata.value.schemas
    .map((schema) =>
      schema.tables.map((table) => ({
        schema: schema.name,
        table: table.name,
      }))
    )
    .flat();
});

const affectedTableOptions = computed(() => {
  return affectedTables.value.map<AffectedTablesSelectOption>((table) => {
    return {
      value: stringifyTable(table),
      table: table,
    };
  });
});

const renderLabel = (option: SelectOption) => {
  const { table, label } = option as AffectedTablesSelectOption;
  if (!table) return String(label);
  return <span class="truncate">{stringifyTable(table)}</span>;
};

const selectedKeys = computed(() => {
  return props.tables?.map((table) => stringifyTable(table)) || [];
});

const updateSelectedKey = (
  _: string,
  options: AffectedTablesSelectOption[]
) => {
  emit(
    "update:tables",
    options.map((option) => {
      const { table, label } = option;
      if (table) return table;

      let schema = "";
      let tableName = String(label);
      if (tableName.includes(".")) {
        schema = tableName.split(".")[0];
        tableName = tableName.split(".").slice(1).join(".");
      }
      return { schema, table: tableName };
    })
  );
};

const stringifyTable = (table: Table) => {
  const { schema, table: tableName } = table;
  if (schema !== "") {
    return `${schema}.${tableName}`;
  }
  return tableName;
};
</script>
