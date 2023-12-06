import { pull, pullAt } from "lodash-es";
import { computed } from "vue";
import { ComposedDatabase } from "@/types";
import {
  ColumnConfig,
  ColumnMetadata,
  DatabaseMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableConfig,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { upsertArray } from "@/utils";
import { useSchemaEditorContext } from "./context";
import { EditStatus } from "./types";

const keyForResource = (
  database: ComposedDatabase,
  metadata: {
    schema: SchemaMetadata;
    table?: TableMetadata;
    column?: ColumnMetadata;
  }
) => {
  const { schema, table, column } = metadata;
  const parts = [database.name];
  if (schema) {
    parts.push(`schemas/${schema.name}`);
  }
  if (table) {
    parts.push(`tables/${table.name}`);
  }
  if (column) {
    parts.push(`columns/${column.name}`);
  }
  return parts.join("/");
};

export const useEditStatus = () => {
  const { dirtyPaths } = useSchemaEditorContext();
  const dirtyPathsArray = computed(() => {
    return Array.from(dirtyPaths.value.keys());
  });

  const markEditStatus = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table?: TableMetadata;
      column?: ColumnMetadata;
    },
    status: EditStatus
  ) => {
    const key = keyForResource(database, metadata);
    dirtyPaths.value.set(key, status);
  };

  const removeEditStatus = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table?: TableMetadata;
      column?: ColumnMetadata;
    },
    recursive: boolean
  ) => {
    const key = keyForResource(database, metadata);
    const keys = recursive
      ? dirtyPathsArray.value.filter((path) => path.startsWith(key))
      : [key];

    keys.forEach((key) => dirtyPaths.value.delete(key));
  };

  const getSchemaStatus = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
    }
  ): EditStatus => {
    const key = keyForResource(database, metadata);
    if (dirtyPaths.value.has(key)) {
      return dirtyPaths.value.get(key)!;
    }
    if (dirtyPathsArray.value.some((path) => path.startsWith(key))) {
      return "updated";
    }
    return "normal";
  };

  const getTableStatus = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    }
  ): EditStatus => {
    const key = keyForResource(database, metadata);
    if (dirtyPaths.value.has(key)) {
      return dirtyPaths.value.get(key)!;
    }
    if (dirtyPathsArray.value.some((path) => path.startsWith(key))) {
      return "updated";
    }
    return "normal";
  };

  const getColumnStatus = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
      column: ColumnMetadata;
    }
  ): EditStatus => {
    const key = keyForResource(database, metadata);
    if (dirtyPaths.value.has(key)) {
      return dirtyPaths.value.get(key)!;
    }
    if (dirtyPathsArray.value.some((path) => path.startsWith(key))) {
      return "updated";
    }
    return "normal";
  };

  return {
    markEditStatus,
    removeEditStatus,
    getSchemaStatus,
    getTableStatus,
    getColumnStatus,
  };
};

export const upsertTableConfig = (
  database: DatabaseMetadata,
  schema: SchemaMetadata,
  table: TableMetadata,
  config: TableConfig | undefined
) => {
  // TODO
  console.log("upsertTableConfig", database, schema, table, config);
};
export const upsertColumnConfig = (
  database: DatabaseMetadata,
  schema: SchemaMetadata,
  table: TableMetadata,
  column: ColumnMetadata,
  config: ColumnConfig | undefined
) => {
  // TODO
  console.log("upsertColumnConfig", database, schema, table, column, config);
};

export const upsertColumnPrimaryKey = (
  table: TableMetadata,
  columnName: string
) => {
  const pkIndex = table.indexes.findIndex((idx) => idx.primary);
  if (pkIndex < 0) {
    table.indexes.push(
      IndexMetadata.fromPartial({
        primary: true,
        name: "PRIMARY",
        expressions: [columnName],
      })
    );
  } else {
    const pk = table.indexes[pkIndex];
    upsertArray(pk.expressions, columnName);
  }
};
export const removeColumnPrimaryKey = (
  table: TableMetadata,
  columnName: string
) => {
  const pkIndex = table.indexes.findIndex((idx) => idx.primary);
  if (pkIndex < 0) {
    return;
  }
  const pk = table.indexes[pkIndex];
  pull(pk.expressions, columnName);
  if (pk.expressions.length === 0) {
    pullAt(table.indexes, pkIndex);
  }
};
export const removeColumnForeignKey = (
  table: TableMetadata,
  columnName: string
) => {
  for (let i = 0; i < table.foreignKeys.length; i++) {
    const fk = table.foreignKeys[i];
    const columnIndex = fk.columns.indexOf(columnName);
    if (columnIndex < 0) continue;
    pullAt(fk.columns, columnIndex);
    pullAt(fk.referencedColumns, columnIndex);
  }
  table.foreignKeys = table.foreignKeys.filter((fk) => fk.columns.length > 0);
};
