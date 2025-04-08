<template>
  <div class="sqleditor--wrapper">
    <Splitpanes
      class="default-theme flex flex-col flex-1 overflow-hidden"
      :dbl-click-splitter="false"
    >
      <Pane v-if="windowWidth >= 800" size="20">
        <AsidePanel />
      </Pane>
      <template v-else>
        <teleport to="body">
          <div
            id="fff"
            class="fixed rounded-full border border-control-border shadow-lg w-10 h-10 bottom-[4rem] flex items-center justify-center bg-white hover:bg-control-bg cursor-pointer z-[99999999] transition-all"
            :class="[
              state.sidebarExpanded
                ? 'left-[80%] -translate-x-5'
                : 'left-[1rem]',
            ]"
            style="
              transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
              transition-duration: 300ms;
            "
            @click="state.sidebarExpanded = !state.sidebarExpanded"
          >
            <heroicons-outline:chevron-left
              class="w-6 h-6 transition-transform"
              :class="[state.sidebarExpanded ? '' : '-scale-100']"
            />
          </div>
          <Drawer
            v-model:show="state.sidebarExpanded"
            width="80vw"
            placement="left"
          >
            <AsidePanel />
          </Drawer>
        </teleport>
      </template>
      <Pane class="relative flex flex-col">
        <TabList />

        <EditorPanel />

        <div
          v-if="isFetchingSheet"
          class="flex items-center justify-center absolute inset-0 bg-white/50 z-20"
        >
          <BBSpin />
        </div>
      </Pane>
    </Splitpanes>

    <Quickstart v-if="!hideQuickStart" />

    <Drawer v-model:show="showSheetPanel">
      <DrawerContent :title="$t('sql-editor.sheet.self')">
        <SheetPanel @close="showSheetPanel = false" />
      </DrawerContent>
    </Drawer>

    <teleport to="#sql-editor-debug">
      <li>[Page]isDisconnected: {{ isDisconnected }}</li>
      <li>[Page]currentTab.id: {{ currentTab?.id }}</li>
      <li>[Page]currentTab.connection: {{ currentTab?.connection }}</li>
    </teleport>

    <ConnectionPanel v-model:show="showConnectionPanel" />
  </div>
</template>

<script lang="ts" setup>
import { useWindowSize } from "@vueuse/core";
import { storeToRefs } from "pinia";
import { Splitpanes, Pane } from "splitpanes";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import Quickstart from "@/components/Quickstart.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  useAppFeature,
  useDatabaseV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { extractProjectResourceName } from "@/utils";
import AsidePanel from "./AsidePanel";
import ConnectionPanel from "./ConnectionPanel";
import EditorPanel from "./EditorPanel";
import { provideCurrentTabViewStateContext } from "./EditorPanel/context";
import { useSheetContext } from "./Sheet";
import SheetPanel from "./SheetPanel";
import TabList from "./TabList";
import { useSQLEditorContext } from "./context";

type LocalState = {
  sidebarExpanded: boolean;
};

const state = reactive<LocalState>({
  sidebarExpanded: false,
});

const router = useRouter();
const databaseStore = useDatabaseV1Store();
const tabStore = useSQLEditorTabStore();
const {
  events: editorEvents,
  showConnectionPanel,
  pendingInsertAtCaret,
} = useSQLEditorContext();
const { showPanel: showSheetPanel } = useSheetContext();

const { currentTab, isDisconnected } = storeToRefs(tabStore);
const hideQuickStart = useAppFeature("bb.feature.hide-quick-start");
const isFetchingSheet = computed(() => false /* editorStore.isFetchingSheet */);

const { width: windowWidth } = useWindowSize();

useEmitteryEventListener(
  editorEvents,
  "alter-schema",
  async ({ databaseName, schema, table }) => {
    const database = await databaseStore.getOrFetchDatabaseByName(databaseName);
    const exampleSQL = ["ALTER TABLE"];
    if (table) {
      if (schema) {
        exampleSQL.push(`${schema}.${table}`);
      } else {
        exampleSQL.push(`${table}`);
      }
    }
    const query = {
      template: "bb.issue.database.schema.update",
      name: `[${database.databaseName}] Edit schema`,
      databaseList: database.name,
      sql: exampleSQL.join(" "),
    };
    const route = router.resolve({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(database.project),
        issueSlug: "create",
      },
      query,
    });
    window.open(route.fullPath, "_blank");
  }
);

const editorPanelContext = provideCurrentTabViewStateContext();

useEmitteryEventListener(
  editorEvents,
  "insert-at-caret",
  async ({ content }) => {
    if (!tabStore.currentTab) return;
    editorPanelContext.updateViewState({
      view: "CODE",
    });
    requestAnimationFrame(() => {
      pendingInsertAtCaret.value = content;
    });
  }
);
</script>

<style lang="postcss">
@import "splitpanes/dist/splitpanes.css";

/* splitpanes pane style */
.splitpanes.default-theme .splitpanes__pane {
  @apply bg-transparent;
}

.splitpanes.default-theme .splitpanes__splitter {
  @apply bg-gray-100;
  min-height: 8px;
  min-width: 8px;
}

.splitpanes.default-theme .splitpanes__splitter:hover {
  @apply bg-accent;
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

<style scoped lang="postcss">
.sqleditor--wrapper {
  @apply w-full flex-1 overflow-hidden flex flex-col;
}
</style>
