<template>
  <div ref="containerElRef" class="w-full h-full px-2 py-2 overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="({ func, position }) => keyWithPosition(func.name, position)"
      :columns="columns"
      :data="layoutReady ? filteredFuncs : []"
      :row-props="rowProps"
      :max-height="tableBodyHeight"
      :virtual-scroll="true"
      :striped="true"
      :bordered="true"
      :bottom-bordered="true"
      row-class-name="cursor-pointer"
    />
  </div>
</template>

<script setup lang="tsx">
import { type DataTableColumn, NDataTable } from "naive-ui";
import { computed, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  FunctionMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getHighlightHTMLByRegExp, useAutoHeightDataTable } from "@/utils";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon";
import { useCurrentTabViewStateContext } from "../../context/viewState";

type FunctionWithPosition = { func: FunctionMetadata; position: number };

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
      position: number;
    }
  ): void;
}>();

const { t } = useI18n();
const { viewState } = useCurrentTabViewStateContext();

const funcsWithPosition = computed(() => {
  return props.funcs.map<FunctionWithPosition>((func, position) => ({
    func,
    position,
  }));
});

const filteredFuncs = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return funcsWithPosition.value.filter(({ func }) =>
      func.name.toLowerCase().includes(keyword)
    );
  }
  return funcsWithPosition.value;
});

const {
  dataTableRef,
  containerElRef,
  tableBodyHeight,
  layoutReady,
  virtualListRef,
} = useAutoHeightDataTable(
  filteredFuncs,
  computed(() => ({
    maxHeight: props.maxHeight ? props.maxHeight : null,
  }))
);

const columns = computed(() => {
  const columns: (DataTableColumn<FunctionWithPosition> & {
    hide?: boolean;
  })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      className: "truncate",
      render: ({ func }) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(func.name, props.keyword ?? ""),
        });
      },
    },
  ];
  return columns;
});

const rowProps = ({ func, position }: FunctionWithPosition) => {
  return {
    onClick: () => {
      emit("click", {
        database: props.database,
        schema: props.schema,
        func,
        position,
      });
    },
  };
};

watch(
  [() => viewState.value?.detail.func, virtualListRef],
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
  background-color: rgb(var(--color-control-bg));
  height: 66.666667%;
}
:deep(.n-data-table-td.input-cell) {
  padding-left: 0.125rem;
  padding-right: 0.25rem;
  padding-top: 0;
  padding-bottom: 0;
}
</style>
