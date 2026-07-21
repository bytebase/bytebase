import { useCallback, useMemo, useRef, useState } from "react";
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
import { keyForResource } from "./core/keyForResource";
import type { EditStatus } from "./core/types";
import type { EditStatusContext, SchemaResourceMetadata } from "./types";

export function useEditStatus(): EditStatusContext {
  const dirtyPathsRef = useRef(new Map<string, EditStatus>());
  const [version, setVersion] = useState(0);
  const bump = useCallback(() => setVersion((v) => v + 1), []);

  // version is used as a dependency to re-derive when dirtyPaths mutates
  const dirtyPathsArray = useMemo(
    () => Array.from(dirtyPathsRef.current.keys()),
    [version]
  );

  const isDirty = useMemo(() => dirtyPathsRef.current.size > 0, [version]);

  const markEditStatus = useCallback(
    (
      database: Database,
      metadata: SchemaResourceMetadata,
      status: EditStatus
    ) => {
      const key = keyForResource(database, metadata);
      dirtyPathsRef.current.set(key, status);
      bump();
    },
    [bump]
  );

  const markEditStatusByKey = useCallback(
    (key: string, status: EditStatus) => {
      dirtyPathsRef.current.set(key, status);
      bump();
    },
    [bump]
  );

  const getEditStatusByKey = useCallback((key: string) => {
    return dirtyPathsRef.current.get(key);
  }, []);

  const removeEditStatus = useCallback(
    (
      database: Database,
      metadata: SchemaResourceMetadata,
      recursive: boolean
    ) => {
      const key = keyForResource(database, metadata);
      if (recursive) {
        const keys = Array.from(dirtyPathsRef.current.keys()).filter(
          (path) => path === key || path.startsWith(`${key}/`)
        );
        keys.forEach((k) => dirtyPathsRef.current.delete(k));
      } else {
        dirtyPathsRef.current.delete(key);
      }
      bump();
    },
    [bump]
  );

  const clearEditStatus = useCallback(() => {
    dirtyPathsRef.current.clear();
    bump();
  }, [bump]);

  const getResourceStatus = useCallback(
    (key: string, checkChildren: boolean): EditStatus => {
      const status = dirtyPathsRef.current.get(key);
      if (status) return status;
      if (
        checkChildren &&
        dirtyPathsArray.some((path) => path.startsWith(`${key}/`))
      ) {
        return "updated";
      }
      return "normal";
    },
    [dirtyPathsArray]
  );

  const getSchemaStatus = useCallback(
    (database: Database, metadata: { schema: SchemaMetadata }): EditStatus => {
      return getResourceStatus(keyForResource(database, metadata), true);
    },
    [getResourceStatus]
  );

  const getTableStatus = useCallback(
    (
      database: Database,
      metadata: { schema: SchemaMetadata; table: TableMetadata }
    ): EditStatus => {
      return getResourceStatus(keyForResource(database, metadata), true);
    },
    [getResourceStatus]
  );

  const getColumnStatus = useCallback(
    (
      database: Database,
      metadata: {
        schema: SchemaMetadata;
        table: TableMetadata;
        column: ColumnMetadata;
      }
    ): EditStatus => {
      return getResourceStatus(keyForResource(database, metadata), true);
    },
    [getResourceStatus]
  );

  const getPartitionStatus = useCallback(
    (
      database: Database,
      metadata: {
        schema: SchemaMetadata;
        table: TableMetadata;
        partition: TablePartitionMetadata;
      }
    ): EditStatus => {
      return getResourceStatus(keyForResource(database, metadata), true);
    },
    [getResourceStatus]
  );

  const getProcedureStatus = useCallback(
    (
      database: Database,
      metadata: { schema: SchemaMetadata; procedure: ProcedureMetadata }
    ): EditStatus => {
      return getResourceStatus(keyForResource(database, metadata), false);
    },
    [getResourceStatus]
  );

  const getFunctionStatus = useCallback(
    (
      database: Database,
      metadata: { schema: SchemaMetadata; function: FunctionMetadata }
    ): EditStatus => {
      return getResourceStatus(keyForResource(database, metadata), false);
    },
    [getResourceStatus]
  );

  const getViewStatus = useCallback(
    (
      database: Database,
      metadata: { schema: SchemaMetadata; view: ViewMetadata }
    ): EditStatus => {
      return getResourceStatus(keyForResource(database, metadata), false);
    },
    [getResourceStatus]
  );

  const replaceTableName = useCallback(
    (
      database: Database,
      metadata: { schema: SchemaMetadata; table: TableMetadata },
      newName: string
    ) => {
      const key = keyForResource(database, metadata);
      const sections = key.split("/");
      sections.pop();
      sections.push(newName);
      const newKey = sections.join("/");
      if (key === newKey) return;

      const pathMap = new Map<string, string>();
      for (const path of dirtyPathsRef.current.keys()) {
        if (path.startsWith(key)) {
          pathMap.set(path, path.replace(key, newKey));
        }
      }

      for (const [oldPath, newPath] of pathMap.entries()) {
        const status = dirtyPathsRef.current.get(oldPath)!;
        dirtyPathsRef.current.delete(oldPath);
        dirtyPathsRef.current.set(newPath, status);
      }
      bump();
    },
    [bump]
  );

  return {
    isDirty,
    version,
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
}
