<template>
  <main class="px-2 py-2 w-full h-full flex flex-col overflow-y-hidden">
    <template v-if="currentTab">
      <TabsContainer
        @on-table-search-pattern="handleTableSearchPattern"
        @on-column-search-pattern="handleColumnSearchPattern"
      />
      <div :key="currentTab.id" class="w-full flex-1 relative overflow-hidden">
        <DatabaseEditor
          v-if="currentTab.type === SchemaEditorTabType.TabForDatabase"
          :search-pattern="state.tableSearchPattern"
        />
        <TableEditor
          v-else-if="currentTab.type === SchemaEditorTabType.TabForTable"
          :search-pattern="state.columnSearchPattern"
        />
      </div>
    </template>
    <EmptyTips v-else />
  </main>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useSchemaEditorV1Store } from "@/store/modules/v1/schemaEditor";
import { SchemaEditorTabType } from "@/types/v1/schemaEditor";
import DatabaseEditor from "./Panels/DatabaseEditor.vue";
import TableEditor from "./Panels/TableEditor.vue";
import TabsContainer from "./TabsContainer.vue";

interface LocalState {
  tableSearchPattern: string;
  columnSearchPattern: string;
}

const schemaEditorV1Store = useSchemaEditorV1Store();
const currentTab = computed(() => schemaEditorV1Store.currentTab);

const state = reactive<LocalState>({
  tableSearchPattern: "",
  columnSearchPattern: "",
});

const handleTableSearchPattern = (tableSearchPattern: string) => {
  state.tableSearchPattern = tableSearchPattern;
};

const handleColumnSearchPattern = (columnSearchPattern: string) => {
  state.columnSearchPattern = columnSearchPattern;
};

watch([() => currentTab.value], () => {
  state.tableSearchPattern = "";
  state.columnSearchPattern = "";
});
</script>
