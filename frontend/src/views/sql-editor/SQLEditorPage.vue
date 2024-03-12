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
        <div class="w-full flex-1 overflow-hidden">
          <Splitpanes
            v-if="
              !currentTab ||
              currentTab.mode === 'READONLY' ||
              currentTab.mode === 'STANDARD'
            "
            horizontal
            class="default-theme"
            :dbl-click-splitter="false"
          >
            <Pane class="flex flex-row overflow-hidden">
              <div class="h-full flex-1 overflow-hidden">
                <Splitpanes
                  vertical
                  class="default-theme"
                  :dbl-click-splitter="false"
                >
                  <Pane>
                    <EditorPanel v-if="isDisconnected || allowReadonlyMode" />
                    <ReadonlyModeNotSupported v-else />
                  </Pane>
                  <Pane
                    v-if="showSecondarySidebar && windowWidth >= 1024"
                    :size="25"
                  >
                    <SecondarySidebar />
                  </Pane>
                </Splitpanes>
              </div>

              <div v-if="windowWidth >= 1024" class="h-full border-l shrink-0">
                <SecondaryGutterBar />
              </div>
            </Pane>
            <Pane
              v-if="!isDisconnected && allowReadonlyMode"
              class="relative"
              :size="40"
            >
              <ResultPanel />
            </Pane>
          </Splitpanes>

          <TerminalPanelV1 v-else-if="currentTab.mode === 'ADMIN'" />

          <AccessDenied v-else />
        </div>

        <div
          v-if="isFetchingSheet"
          class="flex items-center justify-center absolute inset-0 bg-white/50 z-20"
        >
          <BBSpin />
        </div>
      </Pane>
    </Splitpanes>

    <Quickstart v-if="showQuickstart" />

    <Drawer v-model:show="showSheetPanel">
      <DrawerContent :title="$t('sql-editor.sheet.self')">
        <SheetPanel @close="showSheetPanel = false" />
      </DrawerContent>
    </Drawer>

    <SchemaEditorModal
      v-if="alterSchemaState.showModal"
      :database-id-list="alterSchemaState.databaseIdList"
      :new-window="true"
      alter-type="SINGLE_DB"
      @close="alterSchemaState.showModal = false"
    />

    <teleport to="#sql-editor-debug">
      <li>[Page]isDisconnected: {{ isDisconnected }}</li>
      <li>[Page]currentTab.id: {{ currentTab?.id }}</li>
      <li>[Page]currentTab.connection: {{ currentTab?.connection }}</li>
    </teleport>
  </div>
</template>

<script lang="ts" setup>
import { useWindowSize } from "@vueuse/core";
import { storeToRefs } from "pinia";
import { Splitpanes, Pane } from "splitpanes";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import SchemaEditorModal from "@/components/AlterSchemaPrepForm/SchemaEditorModal.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  useActuatorV1Store,
  useConnectionOfCurrentSQLEditorTab,
  useDatabaseV1Store,
  useSQLEditorTabStore,
} from "@/store";
import {
  allowUsingSchemaEditorV1,
  extractProjectResourceName,
  instanceV1HasReadonlyMode,
} from "@/utils";
import AccessDenied from "./AccessDenied.vue";
import AsidePanel from "./AsidePanel/AsidePanel.vue";
import EditorPanel from "./EditorPanel/EditorPanel.vue";
import ReadonlyModeNotSupported from "./ReadonlyModeNotSupported.vue";
import ResultPanel from "./ResultPanel";
import {
  default as SecondarySidebar,
  useSecondarySidebarContext,
} from "./SecondarySidebar";
import { SecondaryGutterBar } from "./SecondarySidebar";
import { useSheetContext } from "./Sheet";
import SheetPanel from "./SheetPanel";
import TabList from "./TabList";
import TerminalPanelV1 from "./TerminalPanel/TerminalPanelV1.vue";
import { useSQLEditorContext } from "./context";

type LocalState = {
  sidebarExpanded: boolean;
};

type AlterSchemaState = {
  showModal: boolean;
  databaseIdList: string[];
};

const state = reactive<LocalState>({
  sidebarExpanded: false,
});

const router = useRouter();
const databaseStore = useDatabaseV1Store();
const actuatorStore = useActuatorV1Store();
const tabStore = useSQLEditorTabStore();
const { events: editorEvents } = useSQLEditorContext();
const { showPanel: showSheetPanel } = useSheetContext();
const { show: showSecondarySidebar } = useSecondarySidebarContext();

const { currentTab, isDisconnected } = storeToRefs(tabStore);
const showQuickstart = computed(() => actuatorStore.pageMode === "BUNDLED");
const isFetchingSheet = computed(() => false /* editorStore.isFetchingSheet */);

const { width: windowWidth } = useWindowSize();

const { instance } = useConnectionOfCurrentSQLEditorTab();

const allowReadonlyMode = computed(() => {
  if (isDisconnected.value) return false;

  return instanceV1HasReadonlyMode(instance.value);
});

const alterSchemaState = reactive<AlterSchemaState>({
  showModal: false,
  databaseIdList: [],
});

useEmitteryEventListener(
  editorEvents,
  "alter-schema",
  ({ databaseUID, schema, table }) => {
    const database = databaseStore.getDatabaseByUID(databaseUID);
    if (allowUsingSchemaEditorV1([database])) {
      // TODO: support open selected database tab directly in Schema Editor.
      alterSchemaState.databaseIdList = [databaseUID];
      alterSchemaState.showModal = true;
    } else {
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
        name: `[${database.name}] Alter schema`,
        project: database.projectEntity.uid,
        databaseList: databaseUID,
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

.secondary-sidebar-gutter .n-tabs-wrapper {
  @apply pt-0;
}
</style>

<style scoped lang="postcss">
.sqleditor--wrapper {
  @apply w-full flex-1 overflow-hidden flex flex-col;
}
</style>
