<template>
  <div v-if="!state.isLoading" class="w-full h-full">
    <Splitpanes
      class="default-theme w-full h-full flex flex-row overflow-hidden"
    >
      <Pane size="20">
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
              <TableEditor
                v-if="currentTab.type === SchemaDesignerTabType.TabForTable"
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
import { Splitpanes, Pane } from "splitpanes";
import { computed, onMounted, reactive, ref } from "vue";
import { useDBSchemaV1Store } from "@/store";
import AsidePanel from "./AsidePanel.vue";
import EmptyTips from "./EmptyTips.vue";
import TabsContainer from "./TabsContainer.vue";
import TableEditor from "./Panels/TableEditor.vue";
import {
  provideSchemaDesignerContext,
  useSchemaDesignerContext,
} from "./common";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { Engine } from "@/types/proto/v1/common";
import { SchemaDesignerTabState, SchemaDesignerTabType } from "./common/type";

interface LocalState {
  isLoading: boolean;
}

const props = defineProps({
  database: {
    type: String,
    required: true,
  },
});
const state = reactive<LocalState>({
  isLoading: true,
});

const { getCurrentTab } = useSchemaDesignerContext();
const dbSchemaStore = useDBSchemaV1Store();
const currentTab = computed(() => {
  return getCurrentTab();
});
const metadata = ref<DatabaseMetadata>(DatabaseMetadata.fromPartial({}));
const baselineMetadata = ref<DatabaseMetadata>(
  DatabaseMetadata.fromPartial({})
);
const tabState = ref<SchemaDesignerTabState>({
  tabMap: new Map(),
});

provideSchemaDesignerContext({
  metadata,
  baselineMetadata: baselineMetadata.value,
  engine: Engine.MYSQL,
  tabState: tabState,
});

onMounted(async () => {
  const databaseMetadata = await dbSchemaStore.getOrFetchDatabaseMetadata(
    props.database
  );
  baselineMetadata.value = databaseMetadata;
  metadata.value = databaseMetadata;
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
