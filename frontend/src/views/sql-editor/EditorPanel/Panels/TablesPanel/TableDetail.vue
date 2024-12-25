<template>
  <div
    class="w-full h-[28px] flex flex-row gap-x-2 justify-between items-center"
  >
    <NTabs
      v-model:value="state.view"
      size="small"
      style="height: 28px; --n-tab-gap: 1rem"
    >
      <template #prefix>
        <NButton text @click="deselect">
          <ChevronLeftIcon class="w-5 h-5" />
          <div class="flex items-center gap-1">
            <TableIcon class="w-4 h-4" />
            <span>{{ table.name }}</span>
          </div>
        </NButton>
      </template>
      <template v-if="tabItems.length > 1">
        <NTab v-for="tab in tabItems" :key="tab.view" :name="tab.view">
          <div class="flex items-center gap-1">
            <component :is="{ render: tab.icon }" class="w-4 h-4" />
            {{ tab.text }}
          </div>
        </NTab>
      </template>
      <template #suffix>
        <SearchBox
          v-model:value="state.keyword"
          size="small"
          style="width: 10rem"
        />
      </template>
    </NTabs>
  </div>
  <ColumnsTable
    v-show="state.view === 'COLUMNS'"
    :db="db"
    :database="database"
    :schema="schema"
    :table="table"
    :keyword="state.keyword"
  />
  <IndexesTable
    v-show="state.view === 'INDEXES'"
    :db="db"
    :database="database"
    :schema="schema"
    :table="table"
    :keyword="state.keyword"
  />
  <ForeignKeysTable
    v-show="state.view === 'FOREIGN-KEYS'"
    :db="db"
    :database="database"
    :schema="schema"
    :table="table"
    :keyword="state.keyword"
  />
  <TriggersTable
    v-show="state.view === 'TRIGGERS'"
    :db="db"
    :database="database"
    :schema="schema"
    :table="table"
    :keyword="state.keyword"
    @click="
      ({ trigger, position }) =>
        updateViewState({
          detail: {
            table: table.name,
            trigger: keyWithPosition(trigger.name, position),
          },
        })
    "
  />
  <PartitionsTable
    v-show="state.view === 'PARTITIONS'"
    :db="db"
    :database="database"
    :schema="schema"
    :table="table"
    :keyword="state.keyword"
  />
</template>

<script setup lang="ts">
import { ChevronLeftIcon } from "lucide-vue-next";
import { NButton, NTab, NTabs } from "naive-ui";
import { computed, h, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  ColumnIcon,
  ForeignKeyIcon,
  IndexIcon,
  TableIcon,
  TablePartitionIcon,
  TriggerIcon,
} from "@/components/Icon";
import { SearchBox } from "@/components/v2";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon";
import { useEditorPanelContext } from "../../context";
import ColumnsTable from "./ColumnsTable.vue";
import ForeignKeysTable from "./ForeignKeysTable.vue";
import IndexesTable from "./IndexesTable.vue";
import PartitionsTable from "./PartitionsTable.vue";
import TriggersTable from "./TriggersTable.vue";

type View = "COLUMNS" | "INDEXES" | "FOREIGN-KEYS" | "PARTITIONS" | "TRIGGERS";
type LocalState = {
  view: View;
  keyword: string;
};

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}>();

const { t } = useI18n();
const { viewState, updateViewState } = useEditorPanelContext();
const state = reactive<LocalState>({
  view: "COLUMNS",
  keyword: "",
});

const tabItems = computed(() => {
  const items = [
    { view: "COLUMNS", text: t("database.columns"), icon: () => h(ColumnIcon) },
  ];
  const { table } = props;
  if (table.indexes.length > 0) {
    items.push({
      view: "INDEXES",
      text: t("schema-editor.index.indexes"),
      icon: () => h(IndexIcon),
    });
  }
  if (table.foreignKeys.length > 0) {
    items.push({
      view: "FOREIGN-KEYS",
      text: t("database.foreign-keys"),
      icon: () => h(ForeignKeyIcon),
    });
  }
  if (table.triggers.length > 0) {
    items.push({
      view: "TRIGGERS",
      text: t("db.triggers"),
      icon: () => h(TriggerIcon),
    });
  }
  if (table.partitions.length > 0) {
    items.push({
      view: "PARTITIONS",
      text: t("schema-editor.table-partition.partitions"),
      icon: () => h(TablePartitionIcon),
    });
  }
  return items;
});

const deselect = () => {
  updateViewState({
    detail: {},
  });
};

watch(
  [
    () => viewState.value?.detail.table,
    () => viewState.value?.detail.column,
    () => viewState.value?.detail.index,
    () => viewState.value?.detail.foreignKey,
    () => viewState.value?.detail.trigger,
    () => viewState.value?.detail.partition,
  ],
  ([table, column, index, foreignKey, trigger, partition]) => {
    if (!table) return;
    if (column) {
      state.view = "COLUMNS";
      return;
    }
    if (index) {
      state.view = "INDEXES";
      return;
    }
    if (foreignKey) {
      state.view = "FOREIGN-KEYS";
      return;
    }
    if (partition) {
      state.view = "PARTITIONS";
      return;
    }
    if (trigger) {
      state.view = "TRIGGERS";
      return;
    }
    // fallback
    state.view = "COLUMNS";
  },
  { immediate: true }
);
</script>
