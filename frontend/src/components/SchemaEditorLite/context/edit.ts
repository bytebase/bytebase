import { computed, ref } from "vue";
import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  FunctionMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
  ViewMetadata,
} from "@/types/proto/v1/database_service";
import type { EditStatus } from "../types";
import { keyForResource } from "./common";

export const useEditStatus = () => {
  const dirtyPaths = ref(new Map<string, EditStatus>());
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
      partition?: TablePartitionMetadata;
      procedure?: ProcedureMetadata;
      function?: FunctionMetadata;
      view?: ViewMetadata;
    },
    status: EditStatus
  ) => {
    const key = keyForResource(database, metadata);
    dirtyPaths.value.set(key, status);
  };

  const markEditStatusByKey = (key: string, status: EditStatus) => {
    dirtyPaths.value.set(key, status);
  };

  const getEditStatusByKey = (key: string) => {
    return dirtyPaths.value.get(key);
  };

  const removeEditStatus = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table?: TableMetadata;
      column?: ColumnMetadata;
      partition?: TablePartitionMetadata;
      procedure?: ProcedureMetadata;
      function?: FunctionMetadata;
      view?: ViewMetadata;
    },
    recursive: boolean
  ) => {
    const key = keyForResource(database, metadata);
    const keys = recursive
      ? dirtyPathsArray.value.filter(
          (path) => path === key || path.startsWith(`${key}/`)
        )
      : [key];

    keys.forEach((key) => dirtyPaths.value.delete(key));
  };

  const removeEditStatusByKey = (key: string, recursive: boolean) => {
    const keys = recursive
      ? dirtyPathsArray.value.filter(
          (path) => path === key || path.startsWith(`${key}/`)
        )
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
    if (dirtyPathsArray.value.some((path) => path.startsWith(`${key}/`))) {
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
    if (dirtyPathsArray.value.some((path) => path.startsWith(`${key}/`))) {
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
    if (dirtyPathsArray.value.some((path) => path.startsWith(`${key}/`))) {
      return "updated";
    }
    return "normal";
  };

  const getPartitionStatus = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
      partition: TablePartitionMetadata;
    }
  ): EditStatus => {
    const key = keyForResource(database, metadata);
    if (dirtyPaths.value.has(key)) {
      return dirtyPaths.value.get(key)!;
    }
    if (dirtyPathsArray.value.some((path) => path.startsWith(`${key}/`))) {
      return "updated";
    }
    return "normal";
  };

  const getProcedureStatus = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      procedure: ProcedureMetadata;
    }
  ): EditStatus => {
    const key = keyForResource(database, metadata);
    if (dirtyPaths.value.has(key)) {
      return dirtyPaths.value.get(key)!;
    }
    return "normal";
  };

  const getFunctionStatus = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      function: FunctionMetadata;
    }
  ): EditStatus => {
    const key = keyForResource(database, metadata);
    if (dirtyPaths.value.has(key)) {
      return dirtyPaths.value.get(key)!;
    }
    return "normal";
  };

  const getViewStatus = (
    database: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      view: ViewMetadata;
    }
  ): EditStatus => {
    const key = keyForResource(database, metadata);
    if (dirtyPaths.value.has(key)) {
      return dirtyPaths.value.get(key)!;
    }
    return "normal";
  };

  const clearEditStatus = () => {
    dirtyPaths.value.clear();
  };

  return {
    dirtyPaths,
    markEditStatus,
    markEditStatusByKey,
    getEditStatusByKey,
    removeEditStatus,
    removeEditStatusByKey,
    clearEditStatus,
    getSchemaStatus,
    getTableStatus,
    getColumnStatus,
    getPartitionStatus,
    getProcedureStatus,
    getFunctionStatus,
    getViewStatus,
  };
};
