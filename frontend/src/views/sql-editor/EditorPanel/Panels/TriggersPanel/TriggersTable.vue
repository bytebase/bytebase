<template>
  <div ref="containerElRef" class="w-full h-full px-2 py-2 overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="
        ({ trigger, position }) => keyWithPosition(trigger.name, position)
      "
      :columns="columns"
      :data="layoutReady ? filteredTriggers : []"
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
  TableMetadata,
  TriggerMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getHighlightHTMLByRegExp, useAutoHeightDataTable } from "@/utils";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon";
import { useCurrentTabViewStateContext } from "../../context/viewState";

type TriggerWithPosition = {
  trigger: TriggerMetadata;
  position: number;
};

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table?: TableMetadata;
  triggers?: TriggerMetadata[];
  keyword?: string;
  maxHeight?: number;
}>();

const emit = defineEmits<{
  (
    event: "click",
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table?: TableMetadata;
      trigger: TriggerMetadata;
      position: number;
    }
  ): void;
}>();

const { t } = useI18n();
const { viewState } = useCurrentTabViewStateContext();

const triggersWithPosition = computed(() => {
  return props.triggers?.map<TriggerWithPosition>((trigger, position) => ({
    trigger,
    position,
  }));
});

const filteredTriggers = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return triggersWithPosition.value?.filter(({ trigger }) =>
      trigger.name.toLowerCase().includes(keyword)
    );
  }
  return triggersWithPosition.value;
});

const {
  dataTableRef,
  containerElRef,
  virtualListRef,
  tableBodyHeight,
  layoutReady,
} = useAutoHeightDataTable(
  filteredTriggers.value,
  computed(() => ({
    maxHeight: props.maxHeight ? props.maxHeight : null,
  }))
);

const columns = computed(() => {
  const columns: (DataTableColumn<TriggerWithPosition> & {
    hide?: boolean;
  })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      className: "truncate",
      render: ({ trigger }) => {
        return h("span", {
          innerHTML: getHighlightHTMLByRegExp(
            trigger.name,
            props.keyword ?? ""
          ),
        });
      },
    },
  ];
  return columns;
});

const rowProps = ({ trigger, position }: TriggerWithPosition) => {
  return {
    onClick: () => {
      emit("click", {
        database: props.database,
        schema: props.schema,
        table: props.table,
        trigger,
        position,
      });
    },
  };
};

watch(
  [() => viewState.value?.detail.trigger, virtualListRef],
  ([trigger, vl]) => {
    if (trigger && vl) {
      vl.scrollTo({ key: trigger });
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
