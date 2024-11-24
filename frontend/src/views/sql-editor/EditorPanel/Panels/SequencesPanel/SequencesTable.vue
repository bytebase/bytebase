<template>
  <div ref="containerElRef" class="w-full h-full px-2 py-2 overflow-x-auto">
    <NDataTable
      v-bind="$attrs"
      ref="dataTableRef"
      size="small"
      :row-key="
        ({ sequence, position }) => keyWithPosition(sequence.name, position)
      "
      :columns="columns"
      :data="layoutReady ? filteredSequences : []"
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
import { NCheckbox, NDataTable, type DataTableColumn } from "naive-ui";
import { computed, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SequenceMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useAutoHeightDataTable } from "@/utils";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon";
import { EllipsisCell } from "../../common";
import { useEditorPanelContext } from "../../context";

type SequenceWithPosition = { sequence: SequenceMetadata; position: number };

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  sequences: SequenceMetadata[];
  keyword?: string;
  maxHeight?: number;
}>();

const emit = defineEmits<{
  (
    event: "click",
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      sequence: SequenceMetadata;
      position: number;
    }
  ): void;
}>();

const { t } = useI18n();
const { viewState } = useEditorPanelContext();

const funcsWithPosition = computed(() => {
  return props.sequences.map<SequenceWithPosition>((sequence, position) => ({
    sequence,
    position,
  }));
});

const filteredSequences = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return funcsWithPosition.value.filter(({ sequence: func }) =>
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
  filteredSequences,
  computed(() => ({
    maxHeight: props.maxHeight ? props.maxHeight : null,
  }))
);

const columns = computed(() => {
  const columns: (DataTableColumn<SequenceWithPosition> & {
    hide?: boolean;
  })[] = [
    {
      key: "name",
      title: t("schema-editor.database.name"),
      resizable: true,
      render: ({ sequence }) => {
        return h(EllipsisCell, {
          content: sequence.name,
          keyword: props.keyword,
        });
      },
    },
    {
      key: "dataType",
      title: t("db.sequence.data-type"),
      resizable: true,
      render: ({ sequence }) => {
        return sequence.dataType;
      },
    },
    {
      key: "start",
      title: t("db.sequence.start"),
      resizable: true,
      render: ({ sequence }) => {
        return sequence.start;
      },
    },
    {
      key: "minValue",
      title: t("db.sequence.min-value"),
      resizable: true,
      render: ({ sequence }) => {
        return sequence.minValue;
      },
    },
    {
      key: "maxValue",
      title: t("db.sequence.max-value"),
      resizable: true,
      render: ({ sequence }) => {
        return h(EllipsisCell, {
          content: sequence.maxValue,
        });
      },
    },
    {
      key: "increment",
      title: t("db.sequence.increment"),
      resizable: true,
      render: ({ sequence }) => {
        return sequence.increment;
      },
    },
    {
      key: "cycle",
      title: t("db.sequence.cycle"),
      resizable: true,
      render: ({ sequence }) => {
        return <NCheckbox checked={sequence.cycle} disabled={true} />;
      },
    },
    {
      key: "cacheSize",
      title: t("db.sequence.cacheSize"),
      resizable: true,
      render: ({ sequence }) => {
        return sequence.cacheSize;
      },
    },
    {
      key: "lastValue",
      title: t("db.sequence.lastValue"),
      resizable: true,
      render: ({ sequence }) => {
        return h(EllipsisCell, {
          content: sequence.lastValue,
        });
      },
    },
  ];
  return columns;
});

const rowProps = ({ sequence, position }: SequenceWithPosition) => {
  return {
    onClick: () => {
      emit("click", {
        database: props.database,
        schema: props.schema,
        sequence,
        position,
      });
    },
  };
};

watch(
  [() => viewState.value?.detail.sequence, virtualListRef],
  ([sequence, vl]) => {
    if (sequence && vl) {
      vl.scrollTo({ key: sequence });
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
