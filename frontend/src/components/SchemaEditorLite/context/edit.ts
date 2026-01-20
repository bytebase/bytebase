import { computed, ref } from "vue";
import type {
  ColumnMetadata,
  Database,
  FunctionMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type { EditStatus } from "../types";
import { keyForResource } from "./common";

export const useEditStatus = () => {
  const dirtyPaths = ref(new Map<string, EditStatus>());
  const dirtyPathsArray = computed(() => {
    return Array.from(dirtyPaths.value.keys());
  });

  const markEditStatus = (
    database: Database,
    metadata: {
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
    database: Database,
    metadata: {
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

  const getSchemaStatus = (
    database: Database,
    metadata: {
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
    database: Database,
    metadata: {
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

  const replaceTableName = (
    database: Database,
    metadata: {
      schema: SchemaMetadata;
      table: TableMetadata;
    },
    newName: string
  ) => {
    const key = keyForResource(database, metadata);

    const sections = key.split("/");
    sections.pop();
    sections.push(newName);
    const newKey = sections.join("/");
    if (key === newKey) {
      return;
    }

    const pathMap = new Map<string /* old path */, string /* new path */>();
    for (const path of dirtyPathsArray.value) {
      if (!dirtyPaths.value.has(path)) {
        continue;
      }
      if (path.startsWith(key)) {
        pathMap.set(path, path.replace(key, newKey));
      }
    }

    for (const [oldPath, newPath] of pathMap.entries()) {
      const status = dirtyPaths.value.get(oldPath)!;
      dirtyPaths.value.delete(oldPath);
      dirtyPaths.value.set(newPath, status);
    }
  };

  const getColumnStatus = (
    database: Database,
    metadata: {
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
    database: Database,
    metadata: {
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
    database: Database,
    metadata: {
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
    database: Database,
    metadata: {
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
    database: Database,
    metadata: {
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

  const isDirty = computed(() => dirtyPaths.value.size > 0);

  return {
    isDirty,
    markEditStatus,
    markEditStatusByKey,
    getEditStatusByKey,
    removeEditStatus,
    clearEditStatus,
    getSchemaStatus,
    getTableStatus,
    getColumnStatus,
    getPartitionStatus,
    getProcedureStatus,
    getFunctionStatus,
    getViewStatus,
    replaceTableName,
  };
};
