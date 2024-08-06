<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="getFunctionKey"
      :columns="columns"
      :data="layoutReady ? funcs : []"
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
import {
  NDataTable,
  NPerformantEllipsis,
  type DataTableColumn,
} from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  FunctionMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useAutoHeightDataTable } from "../../common";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  funcs: FunctionMetadata[];
}>();

const emit = defineEmits<{
  (
    event: "click",
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      func: FunctionMetadata;
    }
  ): void;
}>();

const { t } = useI18n();
const { containerElRef, tableBodyHeight, layoutReady } =
  useAutoHeightDataTable();

const getFunctionKey = (func: FunctionMetadata) => {
  return func.name;
};

const columns = computed(() => {
  const columns: (DataTableColumn<FunctionMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      className: "truncate",
      render: (func) => {
        return (
          <NPerformantEllipsis class="w-full leading-6">
            {func.name}
          </NPerformantEllipsis>
        );
      },
    },
  ];
  return columns;
});

const rowProps = (func: FunctionMetadata) => {
  return {
    onClick: () => {
      emit("click", {
        database: props.database,
        schema: props.schema,
        func,
      });
    },
  };
};
</script>

<style lang="postcss" scoped>
:deep(.n-data-table-th .n-data-table-resize-button::after) {
  @apply bg-control-bg h-2/3;
}
:deep(.n-data-table-td.input-cell) {
  @apply pl-0.5 pr-1 py-0;
}
</style>
