<template>
  <div v-if="!state.isLoading" class="w-full h-full">
    <Splitpanes
      class="default-theme w-full h-full flex flex-row overflow-hidden"
    >
      <Pane min-size="15" size="20">
        <AsidePanel />
      </Pane>
      <Pane min-size="60" size="80">
        <main class="pl-2 pt-2 w-full h-full flex flex-col overflow-y-auto">
          <template v-if="currentTab">
            <TabsContainer />
            <div
              :key="currentTab.id"
              class="w-full h-full relative overflow-y-auto"
            >
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
      </Pane>
    </Splitpanes>
  </div>
</template>

<script lang="ts" setup>
import { uniq } from "lodash-es";
import { Splitpanes, Pane } from "splitpanes";
import { computed, onMounted, PropType, reactive } from "vue";
import { useInstanceV1Store, useSchemaEditorStore } from "@/store";
import { SchemaEditorTabType } from "@/types";
import AsidePanel from "./AsidePanel.vue";
import EmptyTips from "./EmptyTips.vue";
import TabsContainer from "./TabsContainer.vue";
import DatabaseEditor from "./Panels/DatabaseEditor.vue";
import TableEditor from "./Panels/TableEditor.vue";

interface LocalState {
  isLoading: boolean;
}

const props = defineProps({
  databaseIdList: {
    type: Array as PropType<string[]>,
    required: true,
  },
});
const state = reactive<LocalState>({
  isLoading: true,
});

const editorStore = useSchemaEditorStore();
const instanceStore = useInstanceV1Store();
const currentTab = computed(() => {
  return editorStore.currentTab;
});

onMounted(async () => {
  // Reset Schema Editor state before fetching latest data when mounted.
  editorStore.$reset();

  // Prepare instance and database data.
  const databaseIdList = props.databaseIdList;
  const databaseList = await editorStore.fetchDatabaseList(databaseIdList);
  const instanceIdList = uniq(
    databaseList.map((database) => database.instanceEntity.uid)
  );
  for (const instanceId of instanceIdList) {
    await instanceStore.getOrFetchInstanceByUID(instanceId);
  }
  state.isLoading = false;
});
</script>

<style>
@import "splitpanes/dist/splitpanes.css";

/* splitpanes pane style */
.splitpanes.default-theme .splitpanes__pane {
  @apply bg-transparent !transition-none;
}

.splitpanes.default-theme .splitpanes__splitter {
  @apply bg-gray-100 border-none;
}

.splitpanes.default-theme .splitpanes__splitter:hover {
  @apply bg-indigo-300;
}

.splitpanes.default-theme .splitpanes__splitter::before,
.splitpanes.default-theme .splitpanes__splitter::after {
  @apply bg-gray-700 opacity-50 text-white;
}

.splitpanes.default-theme .splitpanes__splitter:hover::before,
.splitpanes.default-theme .splitpanes__splitter:hover::after {
  @apply bg-white opacity-100;
}
</style>
