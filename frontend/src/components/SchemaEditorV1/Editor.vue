<template>
  <main class="px-2 pt-2 w-full h-full flex flex-col overflow-y-auto">
    <template v-if="currentTab">
      <TabsContainer />
      <div :key="currentTab.id" class="w-full h-full relative overflow-y-auto">
        <DatabaseEditor
          v-if="currentTab.type === SchemaEditorTabType.TabForDatabase"
        />
        <TableEditor
          v-else-if="currentTab.type === SchemaEditorTabType.TabForTable"
        />
      </div>
    </template>
    <EmptyTips v-else />
  </main>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useSchemaEditorV1Store } from "@/store/modules/v1/schemaEditor";
import { SchemaEditorTabType } from "@/types/v1/schemaEditor";
import DatabaseEditor from "./Panels/DatabaseEditor.vue";
import TableEditor from "./Panels/TableEditor.vue";
import TabsContainer from "./TabsContainer.vue";

const schemaEditorV1Store = useSchemaEditorV1Store();
const currentTab = computed(() => schemaEditorV1Store.currentTab);
</script>
