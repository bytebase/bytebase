<template>
  <div class="sqleditor--wrapper">
    <teleport v-if="hideSidebar" to="body">
      <div
        id="fff"
        class="fixed rounded-full border border-control-border shadow-lg w-10 h-10 bottom-16 flex items-center justify-center bg-white hover:bg-control-bg cursor-pointer z-99999999 transition-all"
        :class="[
          state.sidebarExpanded ? 'left-[80%] -translate-x-5' : 'left-4',
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
    <NSplit
      direction="horizontal"
      :disabled="hideSidebar"
      :size="hideSidebar ? 0 : state.sizebarSize"
      :min="0.1"
      :max="0.4"
      :resize-trigger-size="1"
      @update:size="size => state.sizebarSize = size"
    >
      <template #1>
        <AsidePanel />
      </template>
      <template #2>
        <div class="h-full relative flex flex-col">
          <TabList />
          <EditorPanel />
        </div>
      </template>
    </NSplit>

    <Quickstart />

    <teleport to="#sql-editor-debug">
      <li>[Page]isDisconnected: {{ isDisconnected }}</li>
      <li>[Page]currentTab.id: {{ currentTab?.id }}</li>
      <li>[Page]currentTab.connection: {{ currentTab?.connection }}</li>
    </teleport>

    <ConnectionPanel v-model:show="showConnectionPanel" />
  </div>

  <IAMRemindModal v-if="projectContextReady" :project-name="projectName" />
</template>

<script lang="ts" setup>
import { useWindowSize } from "@vueuse/core";
import { NSplit } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import IAMRemindModal from "@/components/IAMRemindModal.vue";
import Quickstart from "@/components/Quickstart.vue";
import { Drawer } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  useDatabaseV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
} from "@/store";
import { extractProjectResourceName } from "@/utils";
import AsidePanel from "./AsidePanel";
import ConnectionPanel from "./ConnectionPanel";
import { useSQLEditorContext } from "./context";
import EditorPanel from "./EditorPanel";
import { provideCurrentTabViewStateContext } from "./EditorPanel/context/viewState";
import TabList from "./TabList";

type LocalState = {
  sizebarSize: number;
  sidebarExpanded: boolean;
};

const state = reactive<LocalState>({
  sizebarSize: 0.25,
  sidebarExpanded: false,
});

const router = useRouter();
const databaseStore = useDatabaseV1Store();
const tabStore = useSQLEditorTabStore();
const editorStore = useSQLEditorStore();

const {
  events: editorEvents,
  showConnectionPanel,
  pendingInsertAtCaret,
} = useSQLEditorContext();
const { project: projectName, projectContextReady } = storeToRefs(editorStore);

const { currentTab, isDisconnected } = storeToRefs(tabStore);

const { width: windowWidth } = useWindowSize();
const hideSidebar = computed(() => windowWidth.value < 800);

useEmitteryEventListener(
  editorEvents,
  "alter-schema",
  async ({ databaseName, schema, table }) => {
    const { enabledNewLayout } = useIssueLayoutVersion();
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
      template: "bb.issue.database.update",
      name: `[${database.databaseName}] Edit schema`,
      databaseList: database.name,
      sql: exampleSQL.join(" "),
    };
    const route = enabledNewLayout.value
      ? router.resolve({
          name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
          params: {
            projectId: extractProjectResourceName(database.project),
            planId: "create",
            specId: "placeholder",
          },
          query,
        })
      : router.resolve({
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

useEmitteryEventListener(editorEvents, "insert-at-caret", ({ content }) => {
  if (!tabStore.currentTab) return;
  editorPanelContext.updateViewState({
    view: "CODE",
  });
  requestAnimationFrame(() => {
    pendingInsertAtCaret.value = content;
  });
});
</script>

<style scoped lang="postcss">
.sqleditor--wrapper {
  width: 100%;
  flex: 1 1 0%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
