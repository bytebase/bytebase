<template>
  <Splitpanes
    horizontal
    class="w-full h-full overflow-x-auto relative flex flex-col"
  >
    <Pane class="flex-1 overflow-y-hidden" :size="detail ? 60 : 100">
      <div ref="containerElRef" class="w-full h-full">
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
    </Pane>
    <Pane
      v-if="detail"
      class="flex flex-col items-stretch shrink-0 overflow-hidden"
      :size="40"
    >
      <CodeViewer
        :db="db"
        :title="detail.trigger.name"
        :code="detail.trigger.body"
        header-class="!p-0 h-[34px]"
      >
        <template #title-prefix>
          <NButton text @click="deselect">
            <ChevronDownIcon class="w-5 h-5" />
            <div class="flex items-center gap-1">
              <TriggerIcon class="w-4 h-4" />
              <span>{{ detail.trigger.name }}</span>
            </div>
          </NButton>
        </template>
      </CodeViewer>
    </Pane>
  </Splitpanes>
</template>

<script setup lang="tsx">
import { ChevronDownIcon } from "lucide-vue-next";
import { NButton, NDataTable, type DataTableColumn } from "naive-ui";
import { Pane, Splitpanes } from "splitpanes";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { TriggerIcon } from "@/components/Icon";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  TriggerMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { useAutoHeightDataTable } from "@/utils";
import {
  extractKeyWithPosition,
  keyWithPosition,
} from "@/views/sql-editor/EditorCommon";
import { EllipsisCell } from "../../common";
import { useEditorPanelContext } from "../../context";
import { CodeViewer } from "../common";

type TriggerWithPosition = {
  trigger: TriggerMetadata;
  position: number;
};

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  keyword?: string;
  maxHeight?: number;
}>();

const emit = defineEmits<{
  (
    event: "click",
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
      trigger: TriggerMetadata;
      position: number;
    }
  ): void;
}>();

const { t } = useI18n();
const { viewState, updateViewState } = useEditorPanelContext();

const detail = ref<TriggerWithPosition>();

const triggers = computed(() => {
  return props.table.triggers;
});

const triggersWithPosition = computed(() => {
  return triggers.value.map<TriggerWithPosition>((trigger, position) => ({
    trigger,
    position,
  }));
});

const filteredTriggers = computed(() => {
  const keyword = props.keyword?.trim().toLowerCase();
  if (keyword) {
    return triggersWithPosition.value.filter(({ trigger }) =>
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
        table: props.table,
        trigger,
        position,
      });
    },
  };
};

const deselect = () => {
  updateViewState({
    detail: {
      ...viewState.value?.detail,
      trigger: "-1", // Used as a placeholder to prevent the tab going (fallback) to "Columns"
    },
  });
};

watch(
  [() => viewState.value?.detail.trigger, virtualListRef],
  ([trigger, vl]) => {
    if (trigger) {
      const [name, position] = extractKeyWithPosition(trigger);
      const found = triggersWithPosition.value.find(
        (item) => item.trigger.name === name && item.position === position
      );
      detail.value = found;
    }
    if (trigger && vl) {
      vl.scrollTo({ key: trigger });
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
