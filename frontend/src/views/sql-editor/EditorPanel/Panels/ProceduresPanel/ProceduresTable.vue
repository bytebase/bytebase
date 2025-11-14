<template>
  <div ref="containerElRef" class="w-full h-full px-2 py-2 overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="
        ({ procedure, position }) => keyWithPosition(procedure.name, position)
      "
      :columns="columns"
      :data="layoutReady ? filteredProcedures : []"
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
  ProcedureMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getHighlightHTMLByRegExp, useAutoHeightDataTable } from "@/utils";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon";
import { useCurrentTabViewStateContext } from "../../context/viewState";

type ProcedureWithPosition = {
  procedure: ProcedureMetadata;
  position: number;
};

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  procedures: ProcedureMetadata[];
  keyword?: string;
  maxHeight?: number;
}>();

const emit = defineEmits<{
  (
    event: "click",
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      procedure: ProcedureMetadata;
      position: number;
    }
  ): void;
}>();

const { t } = useI18n();
const { viewState } = useCurrentTabViewStateContext();

const proceduresWithPosition = computed(() => {
  return props.procedures.map<ProcedureWithPosition>((procedure, position) => ({
    procedure,
    position,
  }));
});

const filteredProcedures = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return proceduresWithPosition.value.filter(({ procedure }) =>
      procedure.name.toLowerCase().includes(keyword)
    );
  }
  return proceduresWithPosition.value;
});

const {
  dataTableRef,
  containerElRef,
  virtualListRef,
  tableBodyHeight,
  layoutReady,
} = useAutoHeightDataTable(
  filteredProcedures,
  computed(() => ({
    maxHeight: props.maxHeight ? props.maxHeight : null,
  }))
);

const columns = computed(() => {
  const columns: (DataTableColumn<ProcedureWithPosition> & {
    hide?: boolean;
  })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      className: "truncate",
      render: ({ procedure }) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(
            procedure.name,
            props.keyword ?? ""
          ),
        });
      },
    },
  ];
  return columns;
});

const rowProps = ({ procedure, position }: ProcedureWithPosition) => {
  return {
    onClick: () => {
      emit("click", {
        database: props.database,
        schema: props.schema,
        procedure,
        position,
      });
    },
  };
};

watch(
  [() => viewState.value?.detail.procedure, virtualListRef],
  ([procedure, vl]) => {
    if (procedure && vl) {
      vl.scrollTo({ key: procedure });
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
