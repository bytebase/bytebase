<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="(procedure) => procedure.name"
      :columns="columns"
      :data="layoutReady ? procedures : []"
      :row-props="rowProps"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
    />
  </div>
</template>

<script setup lang="tsx">
import { NDataTable, type DataTableColumn, type DataTableInst } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  ProcedureMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { nextAnimationFrame } from "@/utils";
import { useAutoHeightDataTable } from "../../common";
import { useEditorPanelContext } from "../../context";
import type { RichMetadataWithDB, RichProcedureMetadata } from "../../types";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  procedures: ProcedureMetadata[];
}>();

const emit = defineEmits<{
  (
    event: "click",
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      procedure: ProcedureMetadata;
    }
  ): void;
}>();

const { t } = useI18n();
const { containerElRef, tableBodyHeight, layoutReady } =
  useAutoHeightDataTable();
const dataTableRef = ref<DataTableInst>();
const vlRef = computed(() => {
  return (dataTableRef.value as any)?.$refs?.mainTableInstRef?.bodyInstRef
    ?.virtualListRef;
});
const { useConsumePendingScrollToTarget } = useEditorPanelContext();

const columns = computed(() => {
  const columns: (DataTableColumn<ProcedureMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      className: "truncate",
    },
  ];
  return columns;
});

const rowProps = (procedure: ProcedureMetadata) => {
  return {
    onClick: () => {
      emit("click", {
        database: props.database,
        schema: props.schema,
        procedure,
      });
    },
  };
};

useConsumePendingScrollToTarget(
  (target: RichMetadataWithDB<"procedure">) => {
    if (target.db.name !== props.db.name) {
      return false;
    }
    if (target.metadata.type === "procedure") {
      const metadata = target.metadata as RichProcedureMetadata;
      return metadata.schema.name === props.schema.name;
    }
    return false;
  },
  vlRef,
  async (target, vl) => {
    const key = target.metadata.procedure.name;
    if (!key) return false;
    await nextAnimationFrame();
    try {
      console.debug("scroll-to-procedure", vl, target, key);
      vl.scrollTo({ key });
    } catch {
      // Do nothing
    }
    return true;
  }
);
</script>

<style lang="postcss" scoped>
:deep(.n-data-table-th .n-data-table-resize-button::after) {
  @apply bg-control-bg h-2/3;
}
:deep(.n-data-table-td.input-cell) {
  @apply pl-0.5 pr-1 py-0;
}
</style>
