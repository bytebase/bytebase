import { computed, ref } from "vue";
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { EditStatus } from "../types";
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
    },
    status: EditStatus
  ) => {
    const key = keyForResource(database, metadata);
    dirtyPaths.value.set(key, status);
  };

  const markEditStatusByKey = (key: string, status: EditStatus) => {
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

  const clearEditStatus = () => {
    dirtyPaths.value.clear();
  };

  return {
    markEditStatus,
    markEditStatusByKey,
    removeEditStatus,
    clearEditStatus,
    getSchemaStatus,
    getTableStatus,
    getColumnStatus,
  };
};
