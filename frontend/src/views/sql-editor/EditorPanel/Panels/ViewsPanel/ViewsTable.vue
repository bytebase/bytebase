<template>
  <div ref="containerElRef" class="w-full h-full px-2 py-2 overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="(view) => view.name"
      :columns="columns"
      :data="layoutReady ? filteredViews : []"
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
  SchemaMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getHighlightHTMLByRegExp, useAutoHeightDataTable } from "@/utils";
import { EllipsisCell } from "../../common";
import { useCurrentTabViewStateContext } from "../../context/viewState";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  views: ViewMetadata[];
  keyword?: string;
  maxHeight?: number;
}>();

const emit = defineEmits<{
  (
    event: "click",
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      view: ViewMetadata;
    }
  ): void;
}>();

const { t } = useI18n();
const { viewState } = useCurrentTabViewStateContext();

const filteredViews = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return props.views.filter((view) =>
      view.name.toLowerCase().includes(keyword)
    );
  }
  return props.views;
});

const {
  dataTableRef,
  containerElRef,
  virtualListRef,
  tableBodyHeight,
  layoutReady,
} = useAutoHeightDataTable(
  filteredViews,
  computed(() => ({
    maxHeight: props.maxHeight ? props.maxHeight : null,
  }))
);

const columns = computed(() => {
  const downGrade = filteredViews.value.length > 50;
  const columns: (DataTableColumn<ViewMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      className: "truncate",
      render: (view) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(view.name, props.keyword ?? ""),
        });
      },
    },
    {
      key: "comment",
      title: t("schema-editor.database.comment"),
      resizable: true,
      className: "overflow-hidden",
      render: (view) => {
        return h(EllipsisCell, {
          content: view.comment,
          downGrade,
        });
      },
    },
  ];
  return columns;
});

const rowProps = (view: ViewMetadata) => {
  return {
    onClick: () => {
      emit("click", {
        database: props.database,
        schema: props.schema,
        view,
      });
    },
  };
};

watch(
  [() => viewState.value?.detail.view, virtualListRef],
  ([view, vl]) => {
    if (view && vl) {
      vl.scrollTo({ key: view });
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
