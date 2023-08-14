<template>
  <main class="px-2 pt-2 w-full h-full flex flex-col overflow-y-auto">
    <template v-if="currentTab">
      <TabsContainer />
      <div :key="currentTab.id" class="w-full h-full relative overflow-y-auto">
        <TableEditor
          v-if="currentTab.type === SchemaDesignerTabType.TabForTable"
        />
      </div>
    </template>
    <EmptyTips v-else />
  </main>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { SchemaDesignerTabType, useSchemaDesignerContext } from "./common";
import TableEditor from "./Panels/TableEditor.vue"
import TabsContainer from "./TabsContainer.vue"

const { getCurrentTab } = useSchemaDesignerContext();
const currentTab = computed(() => getCurrentTab());
</script>
