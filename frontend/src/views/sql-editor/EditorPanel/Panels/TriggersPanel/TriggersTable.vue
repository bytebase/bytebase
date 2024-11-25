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
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed, h, watch, withModifiers } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  TriggerMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useAutoHeightDataTable } from "@/utils";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon";
import { EllipsisCell } from "../../common";
import { useEditorPanelContext } from "../../context";

type TriggerWithPosition = {
  trigger: TriggerMetadata;
  position: number;
};

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  triggers: TriggerMetadata[];
  keyword?: string;
  maxHeight?: number;
}>();

const emit = defineEmits<{
  (
    event: "click",
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      trigger: TriggerMetadata;
      position: number;
    }
  ): void;
}>();

const { t } = useI18n();
const { viewState, updateViewState } = useEditorPanelContext();

const triggersWithPosition = computed(() => {
  return props.triggers.map<TriggerWithPosition>((trigger, position) => ({
    trigger,
    position,
  }));
});

const filteredTriggers = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return triggersWithPosition.value.filter(
      ({ trigger }) =>
        trigger.name.toLowerCase().includes(keyword) ||
        trigger.tableName.toLowerCase().includes(keyword)
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
  filteredTriggers,
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
      title: t("common.name"),
      resizable: true,
      render: ({ trigger }) => {
        return <EllipsisCell content={trigger.name} keyword={props.keyword} />;
      },
    },
    {
      key: "table-name",
      title: t("db.trigger.table-name"),
      resizable: true,
      render: ({ trigger }) => {
        return h(EllipsisCell, {
          content: trigger.tableName,
          keyword: props.keyword,
          class: "cursor-pointer hover:underline hover:text-accent",
          onClick: withModifiers(() => {
            const { schema } = props;
            updateViewState({
              view: "TABLES",
              schema: schema.name,
              detail: {
                table: trigger.tableName,
              },
            });
          }, ["stop", "prevent"]),
        });
      },
    },
    {
      key: "event",
      title: t("db.trigger.event"),
      resizable: true,
      render: ({ trigger }) => {
        return trigger.event;
      },
    },
    {
      key: "timing",
      title: t("db.trigger.timing"),
      resizable: true,
      render: ({ trigger }) => {
        return trigger.timing;
      },
    },
    {
      key: "body",
      title: t("db.trigger.body"),
      resizable: true,
      render: ({ trigger }) => {
        return (
          <EllipsisCell
            content={trigger.body}
            class="font-mono"
            tooltipClass="font-mono"
          />
        );
      },
    },
    {
      key: "sql-mode",
      title: "SQL mode",
      resizable: true,
      render: ({ trigger }) => {
        return (
          <EllipsisCell
            content={trigger.sqlMode}
            tooltip={trigger.sqlMode.replaceAll(",", ",\n")}
          />
        );
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
        trigger,
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
  @apply bg-control-bg h-2/3;
}
:deep(.n-data-table-td.input-cell) {
  @apply pl-0.5 pr-1 py-0;
}
</style>
