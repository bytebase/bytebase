<template>
  <main class="px-2 py-2 w-full h-full flex flex-col overflow-y-hidden">
    <template v-if="currentTab">
      <TabsContainer
        @update:table-search-pattern="handleTableSearchPattern"
        @update:column-search-pattern="handleColumnSearchPattern"
      />
      <div
        class="w-full flex-1 relative overflow-hidden"
        :data-key="currentTab.id"
      >
        <DatabaseEditor
          v-if="currentTab.type === 'database'"
          :key="currentTab.id"
          v-model:selected-schema-name="currentTab.selectedSchema"
          :db="currentTab.database"
          :database="currentTab.metadata.database"
          :search-pattern="state.tableSearchPattern"
        />
        <TableEditor
          v-if="currentTab.type === 'table'"
          :key="currentTab.id"
          :db="currentTab.database"
          :database="currentTab.metadata.database"
          :schema="currentTab.metadata.schema"
          :table="currentTab.metadata.table"
          :search-pattern="state.columnSearchPattern"
        />
      </div>
    </template>
    <EmptyTips v-else />
  </main>
</template>

<script lang="ts" setup>
import { reactive, watch } from "vue";
import DatabaseEditor from "./Panels/DatabaseEditor.vue";
import TableEditor from "./Panels/TableEditor.vue";
import TabsContainer from "./TabsContainer.vue";
import { useSchemaEditorContext } from "./context";

interface LocalState {
  tableSearchPattern: string;
  columnSearchPattern: string;
}

const { currentTab } = useSchemaEditorContext();

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

watch([() => currentTab.value?.id], () => {
  state.tableSearchPattern = "";
  state.columnSearchPattern = "";
});
</script>
