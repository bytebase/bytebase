<template>
  <div
    class="w-full h-11 px-2 py-2 border-b flex flex-row justify-between items-center"
  >
    <NTabs
      v-model:value="state.mode"
      size="small"
      style="height: 28px; --n-tab-gap: 1rem"
    >
      <template #prefix>
        <NButton text @click="deselect">
          <ChevronLeftIcon class="w-5 h-5" />
          <div class="flex items-center gap-1">
            <ViewIcon class="w-4 h-4" />
            <span>{{ view.name }}</span>
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
          v-if="state.mode !== 'DEFINITION'"
          v-model:value="state.keyword"
          size="small"
          style="width: 10rem"
        />

        <div v-if="state.mode === 'DEFINITION'" class="flex items-center gap-2">
          <NCheckbox
            v-if="state.mode === 'DEFINITION'"
            v-model:checked="format"
          >
            {{ $t("sql-editor.format") }}
          </NCheckbox>
          <OpenAIButton
            size="small"
            :statement="selectedStatement || view.definition"
            :actions="['explain-code']"
          />
        </div>
      </template>
    </NTabs>
  </div>
  <DefinitionViewer
    v-show="state.mode === 'DEFINITION'"
    :db="db"
    :code="view.definition"
    :format="format"
    @select-content="selectedStatement = $event"
  />
  <ColumnsTable
    v-show="state.mode === 'COLUMNS'"
    :db="db"
    :database="database"
    :schema="schema"
    :view="view"
    :keyword="state.keyword"
  />
  <DependencyColumnsTable
    v-show="state.mode === 'DEPENDENCY-COLUMNS'"
    :db="db"
    :database="database"
    :schema="schema"
    :view="view"
    :keyword="state.keyword"
  />
</template>

<script setup lang="ts">
import { useLocalStorage } from "@vueuse/core";
import { ChevronLeftIcon, CodeIcon, FileSymlinkIcon } from "lucide-vue-next";
import { NButton, NCheckbox, NTab, NTabs } from "naive-ui";
import { computed, h, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { ColumnIcon, ViewIcon } from "@/components/Icon";
import { SearchBox } from "@/components/v2";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { OpenAIButton } from "@/views/sql-editor/EditorCommon";
import { useCurrentTabViewStateContext } from "../../context/viewState";
import ColumnsTable from "./ColumnsTable.vue";
import DefinitionViewer from "./DefinitionViewer.vue";
import DependencyColumnsTable from "./DependencyColumnsTable.vue";

type Mode = "DEFINITION" | "COLUMNS" | "DEPENDENCY-COLUMNS";
type LocalState = {
  mode: Mode;
  keyword: string;
};

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  view: ViewMetadata;
}>();

const { t } = useI18n();
const { viewState, updateViewState } = useCurrentTabViewStateContext();
const state = reactive<LocalState>({
  mode: "COLUMNS",
  keyword: "",
});
const format = useLocalStorage<boolean>(
  "bb.sql-editor.editor-panel.code-viewer.format",
  false
);
const selectedStatement = ref("");

const tabItems = computed(() => {
  const items = [
    {
      view: "DEFINITION",
      text: t("common.definition"),
      icon: () => h(CodeIcon),
    },
  ];
  const { view } = props;
  if (view.columns.length > 0) {
    items.push({
      view: "COLUMNS",
      text: t("database.columns"),
      icon: () => h(ColumnIcon),
    });
  }
  if (view.dependencyColumns.length > 0) {
    items.push({
      view: "DEPENDENCY-COLUMNS",
      text: t("schema-editor.index.dependency-columns"),
      icon: () => h(FileSymlinkIcon),
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
    () => viewState.value?.detail.view,
    () => viewState.value?.detail.column,
    () => viewState.value?.detail.dependencyColumn,
  ],
  ([view, column, dependencyColumn]) => {
    if (!view) return;
    if (column) {
      state.mode = "COLUMNS";
      return;
    }
    if (dependencyColumn) {
      state.mode = "DEPENDENCY-COLUMNS";
      return;
    }
    // fallback
    state.mode = "DEFINITION";
  },
  { immediate: true }
);
</script>
