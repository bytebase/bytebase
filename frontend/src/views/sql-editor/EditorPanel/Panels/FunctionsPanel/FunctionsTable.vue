<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="(func) => keyForFunction(func)"
      :columns="columns"
      :data="layoutReady ? filteredFuncs : []"
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
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  FunctionMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { getHighlightHTMLByRegExp, keyForFunction } from "@/utils";
import { useAutoHeightDataTable } from "../../common";
import { useEditorPanelContext } from "../../context";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  funcs: FunctionMetadata[];
  keyword?: string;
  maxHeight?: number;
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
const { viewState } = useEditorPanelContext();

const filteredFuncs = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return props.funcs.filter((func) =>
      func.name.toLowerCase().includes(keyword)
    );
  }
  return props.funcs;
});

const { dataTableRef, containerElRef, tableBodyHeight, layoutReady } =
  useAutoHeightDataTable(
    filteredFuncs,
    computed(() => ({
      maxHeight: props.maxHeight ? props.maxHeight : null,
    }))
  );
const vlRef = computed(() => {
  return (dataTableRef.value as any)?.$refs?.mainTableInstRef?.bodyInstRef
    ?.virtualListRef;
});

const columns = computed(() => {
  const columns: (DataTableColumn<FunctionMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      className: "truncate",
      render: (func) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(func.name, props.keyword ?? ""),
        });
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

watch(
  [() => viewState.value?.detail.func, vlRef],
  ([func, vl]) => {
    if (func && vl) {
      vl.scrollTo({ key: func });
    }
  },
  { immediate: true }
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
