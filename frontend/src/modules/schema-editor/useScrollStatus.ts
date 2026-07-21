import { useCallback, useEffect, useRef, useState } from "react";
import type {
  RichColumnMetadata,
  RichMetadataWithDB,
  RichSchemaMetadata,
  RichTableMetadata,
  ScrollStatusContext,
} from "./types";

export function useScrollStatus(): ScrollStatusContext {
  const [pendingScrollToTable, setPendingScrollToTable] = useState<
    RichMetadataWithDB<RichTableMetadata> | undefined
  >();
  const [pendingScrollToColumn, setPendingScrollToColumn] = useState<
    RichMetadataWithDB<RichColumnMetadata> | undefined
  >();

  const queuePendingScrollToTable = useCallback(
    (params: RichMetadataWithDB<RichTableMetadata>) => {
      requestAnimationFrame(() => {
        setPendingScrollToTable(params);
      });
    },
    []
  );

  const queuePendingScrollToColumn = useCallback(
    (params: RichMetadataWithDB<RichColumnMetadata>) => {
      requestAnimationFrame(() => {
        setPendingScrollToColumn(params);
      });
    },
    []
  );

  const consumePendingScrollToTable = useCallback(() => {
    setPendingScrollToTable(undefined);
  }, []);

  const consumePendingScrollToColumn = useCallback(() => {
    setPendingScrollToColumn(undefined);
  }, []);

  return {
    pendingScrollToTable,
    pendingScrollToColumn,
    queuePendingScrollToTable,
    queuePendingScrollToColumn,
    consumePendingScrollToTable,
    consumePendingScrollToColumn,
  };
}

/**
 * Hook to consume a pending scroll-to-table event when the condition matches.
 * Replaces the Vue `useConsumePendingScrollToTable` composable.
 */
export function useConsumePendingScrollToTable(
  condition: RichMetadataWithDB<RichSchemaMetadata> | undefined,
  pending: RichMetadataWithDB<RichTableMetadata> | undefined,
  consume: () => void,
  fn: (params: RichMetadataWithDB<RichTableMetadata>) => void
) {
  const mountedRef = useRef(false);

  useEffect(() => {
    mountedRef.current = true;
  }, []);

  useEffect(() => {
    if (!mountedRef.current) return;
    if (!pending) return;
    if (!condition) return;
    if (
      pending.db.name === condition.db.name &&
      pending.metadata.schema.name === condition.metadata.schema.name
    ) {
      fn(pending);
      consume();
    }
  }, [pending, condition, fn, consume]);
}

/**
 * Hook to consume a pending scroll-to-column event when the condition matches.
 * Replaces the Vue `useConsumePendingScrollToColumn` composable.
 */
export function useConsumePendingScrollToColumn(
  condition: RichMetadataWithDB<RichTableMetadata> | undefined,
  pending: RichMetadataWithDB<RichColumnMetadata> | undefined,
  consume: () => void,
  fn: (params: RichMetadataWithDB<RichColumnMetadata>) => void
) {
  const mountedRef = useRef(false);

  useEffect(() => {
    mountedRef.current = true;
  }, []);

  useEffect(() => {
    if (!mountedRef.current) return;
    if (!pending) return;
    if (!condition) return;
    if (
      pending.db.name === condition.db.name &&
      pending.metadata.schema.name === condition.metadata.schema.name &&
      pending.metadata.table.name === condition.metadata.table.name
    ) {
      fn(pending);
      consume();
    }
  }, [pending, condition, fn, consume]);
}
