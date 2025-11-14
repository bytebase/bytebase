<template>
  <main class="w-full h-full flex flex-col overflow-y-hidden">
    <template v-if="currentTab">
      <TabsContainer class="px-2 pt-2" />
      <div
        class="w-full flex-1 relative overflow-y-hidden"
        :data-key="currentTab.id"
      >
        <DatabaseEditor
          v-if="currentTab.type === 'database'"
          :key="currentTab.id"
          v-model:selected-schema-name="currentTab.selectedSchema"
          :db="currentTab.database"
          :database="currentTab.metadata.database"
          class="px-2 pb-2"
        />
        <TableEditor
          v-if="currentTab.type === 'table'"
          :key="currentTab.id"
          :db="currentTab.database"
          :database="currentTab.metadata.database"
          :schema="currentTab.metadata.schema"
          :table="currentTab.metadata.table"
        />
        <ProcedureEditor
          v-if="currentTab.type === 'procedure'"
          :key="currentTab.id"
          :db="currentTab.database"
          :database="currentTab.metadata.database"
          :schema="currentTab.metadata.schema"
          :procedure="currentTab.metadata.procedure"
          class="pb-2"
        />
        <FunctionEditor
          v-if="currentTab.type === 'function'"
          :key="currentTab.id"
          :db="currentTab.database"
          :database="currentTab.metadata.database"
          :schema="currentTab.metadata.schema"
          :func="currentTab.metadata.function"
          class="pb-2"
        />
        <ViewEditor
          v-if="currentTab.type === 'view'"
          :key="currentTab.id"
          :db="currentTab.database"
          :database="currentTab.metadata.database"
          :schema="currentTab.metadata.schema"
          :view="currentTab.metadata.view"
        />
      </div>
    </template>
    <EmptyTips v-else />
  </main>
</template>

<script lang="ts" setup>
import { useSchemaEditorContext } from "./context";
import EmptyTips from "./EmptyTips.vue";
import DatabaseEditor from "./Panels/DatabaseEditor.vue";
import FunctionEditor from "./Panels/FunctionEditor.vue";
import ProcedureEditor from "./Panels/ProcedureEditor.vue";
import TableEditor from "./Panels/TableEditor.vue";
import ViewEditor from "./Panels/ViewEditor.vue";
import TabsContainer from "./TabsContainer.vue";

const { currentTab } = useSchemaEditorContext();
</script>
