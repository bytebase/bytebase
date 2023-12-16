import { useMounted } from "@vueuse/core";
import { Ref, computed, ref, watch } from "vue";
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

type RichSchemaMetadata = {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
};
type RichTableMetadata = RichSchemaMetadata & {
  table: TableMetadata;
};
type RichColumnMetadata = RichTableMetadata & {
  column: ColumnMetadata;
  // field?: "name" | "type"; // TODO
};
type RichMetadataWithDB<T> = {
  db: ComposedDatabase;
  metadata: T;
};

export const useScrollStatus = () => {
  const pendingScrollToTable = ref<RichMetadataWithDB<RichTableMetadata>>();
  const pendingScrollToColumn = ref<RichMetadataWithDB<RichColumnMetadata>>();
  const useConsumePendingScrollToTable = <TContext>(
    condition: Ref<RichMetadataWithDB<RichSchemaMetadata>>,
    context: Ref<TContext>,
    fn: (
      params: RichMetadataWithDB<RichTableMetadata>,
      context: TContext
    ) => void
  ) => {
    const mounted = useMounted();
    const matched = computed(() => {
      const pending = pendingScrollToTable.value;
      if (!pending) return false;
      return (
        pending.db.name === condition.value.db.name &&
        pending.metadata.schema.name === condition.value.metadata.schema.name
      );
    });
    watch(
      [mounted, matched, context, pendingScrollToTable],
      ([mounted, matched, context, pending]) => {
        if (!mounted) return;
        if (!matched) return;
        if (!context) return;
        if (!pending) return;
        fn(pending, context);
        pendingScrollToTable.value = undefined;
      },
      { immediate: true }
    );
  };
  const useConsumePendingScrollToColumn = <TContext>(
    condition: Ref<RichMetadataWithDB<RichTableMetadata>>,
    context: Ref<TContext>,
    fn: (
      params: RichMetadataWithDB<RichColumnMetadata>,
      context: TContext
    ) => void
  ) => {
    const mounted = useMounted();
    const matched = computed(() => {
      const pending = pendingScrollToColumn.value;
      if (!pending) return false;
      return (
        pending.db.name === condition.value.db.name &&
        pending.metadata.schema.name === condition.value.metadata.schema.name &&
        pending.metadata.table.name === condition.value.metadata.table.name
      );
    });
    watch(
      [mounted, matched, context, pendingScrollToColumn],
      ([mounted, matched, context, pending]) => {
        if (!mounted) return;
        if (!matched) return;
        if (!context) return;
        if (!pending) return;
        fn(pending, context);
        pendingScrollToColumn.value = undefined;
      },
      { immediate: true }
    );
  };
  const queuePendingScrollToTable = (
    params: RichMetadataWithDB<RichTableMetadata>
  ) => {
    requestAnimationFrame(() => {
      pendingScrollToTable.value = params;
    });
  };
  const queuePendingScrollToColumn = (
    params: RichMetadataWithDB<RichColumnMetadata>
  ) => {
    requestAnimationFrame(() => {
      pendingScrollToColumn.value = params;
    });
  };

  return {
    queuePendingScrollToTable,
    queuePendingScrollToColumn,
    useConsumePendingScrollToTable,
    useConsumePendingScrollToColumn,
  };
};
