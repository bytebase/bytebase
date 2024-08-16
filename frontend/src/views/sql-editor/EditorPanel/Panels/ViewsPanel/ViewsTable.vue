<template>
  <div ref="containerElRef" class="w-full h-full overflow-x-auto">
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
    />
  </div>
</template>

<script setup lang="tsx">
import { NDataTable, type DataTableColumn, type DataTableInst } from "naive-ui";
import { computed, h, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  ViewMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { getHighlightHTMLByRegExp } from "@/utils";
import { useAutoHeightDataTable } from "../../common";
import { useEditorPanelContext } from "../../context";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  views: ViewMetadata[];
  keyword?: string;
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
const { containerElRef, tableBodyHeight, layoutReady } =
  useAutoHeightDataTable();
const dataTableRef = ref<DataTableInst>();
const vlRef = computed(() => {
  return (dataTableRef.value as any)?.$refs?.mainTableInstRef?.bodyInstRef
    ?.virtualListRef;
});
const { viewState } = useEditorPanelContext();

const filteredViews = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return props.views.filter((view) => view.name.includes(keyword));
  }
  return props.views;
});

const columns = computed(() => {
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
      className: "truncate",
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
  [() => viewState.value?.detail.view, vlRef],
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
  @apply bg-control-bg h-2/3;
}
:deep(.n-data-table-td.input-cell) {
  @apply pl-0.5 pr-1 py-0;
}
</style>
