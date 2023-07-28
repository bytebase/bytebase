<template>
  <div class="sqleditor--wrapper">
    <TabList />
    <Splitpanes class="default-theme flex flex-col flex-1 overflow-hidden">
      <Pane v-if="windowWidth >= 800" size="20">
        <AsidePanel @alter-schema="handleAlterSchema" />
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
          <NDrawer
            v-model:show="state.sidebarExpanded"
            width="80vw"
            placement="left"
          >
            <AsidePanel @alter-schema="handleAlterSchema" />
          </NDrawer>
        </teleport>
      </template>
      <Pane class="relative">
        <template v-if="allowAccess">
          <template v-if="tabStore.currentTab.mode === TabMode.ReadOnly">
            <Splitpanes
              v-if="allowReadOnlyMode"
              horizontal
              class="default-theme"
            >
              <Pane>
                <EditorPanel />
              </Pane>
              <Pane v-if="!isDisconnected" :size="40">
                <ResultPanel />
              </Pane>
            </Splitpanes>

            <div
              v-else
              class="w-full h-full flex flex-col items-center justify-center gap-y-2"
            >
              <img
                src="../../assets/illustration/403.webp"
                class="max-h-[40%]"
              />
              <i18n-t
                class="textinfolabel flex items-center"
                keypath="sql-editor.allow-admin-mode-only"
                tag="div"
              >
                <template #instance>
                  <InstanceV1Name :instance="instance" :link="false" />
                </template>
              </i18n-t>
              <AdminModeButton />
            </div>
          </template>

          <TerminalPanelV1 v-if="tabStore.currentTab.mode === TabMode.Admin" />
        </template>
        <div
          v-else
          class="w-full h-full flex flex-col items-center justify-center"
        >
          <img src="../../assets/illustration/403.webp" class="max-h-[40%]" />
          <div class="textinfolabel">
            {{ $t("database.access-denied") }}
          </div>
        </div>

        <div
          v-if="isFetchingSheet"
          class="flex items-center justify-center absolute inset-0 bg-white/50 z-20"
        >
          <BBSpin />
        </div>
      </Pane>
    </Splitpanes>

    <Quickstart />

    <SchemaEditorModal
      v-if="alterSchemaState.showModal"
      :database-id-list="alterSchemaState.databaseIdList.map((id) => `${id}`)"
      :new-window="true"
      alter-type="SINGLE_DB"
      @close="alterSchemaState.showModal = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { Splitpanes, Pane } from "splitpanes";
import { stringify } from "qs";
import { NDrawer } from "naive-ui";

import { DatabaseId, TabMode, UNKNOWN_ID } from "@/types";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useInstanceV1ByUID,
  useSQLEditorStore,
  useTabStore,
} from "@/store";
import AsidePanel from "./AsidePanel/AsidePanel.vue";
import EditorPanel from "./EditorPanel/EditorPanel.vue";
import TerminalPanelV1 from "./TerminalPanel/TerminalPanelV1.vue";
import TabList from "./TabList";
import ResultPanel from "./ResultPanel";
import {
  allowUsingSchemaEditorV1,
  instanceV1HasReadonlyMode,
  isDatabaseV1Queryable,
} from "@/utils";
import AdminModeButton from "./EditorCommon/AdminModeButton.vue";
import SchemaEditorModal from "@/components/AlterSchemaPrepForm/SchemaEditorModal.vue";
import { useWindowSize } from "@vueuse/core";
import { InstanceV1Name } from "@/components/v2";
import { provideSheetPanelContext } from "./TabList/SheetPanel/common";

type LocalState = {
  sidebarExpanded: boolean;
};

type AlterSchemaState = {
  showModal: boolean;
  databaseIdList: DatabaseId[];
};

const state = reactive<LocalState>({
  sidebarExpanded: false,
});

const tabStore = useTabStore();
const databaseStore = useDatabaseV1Store();
const sqlEditorStore = useSQLEditorStore();
const currentUserV1 = useCurrentUserV1();

const isDisconnected = computed(() => tabStore.isDisconnected);
const isFetchingSheet = computed(() => sqlEditorStore.isFetchingSheet);

const { width: windowWidth } = useWindowSize();

const allowAccess = computed(() => {
  const { databaseId } = tabStore.currentTab.connection;
  const database = databaseStore.getDatabaseByUID(databaseId);
  if (database.uid === String(UNKNOWN_ID)) {
    // Allowed if connected to an instance
    return true;
  }
  return isDatabaseV1Queryable(database, currentUserV1.value);
});

const { instance } = useInstanceV1ByUID(
  computed(() => tabStore.currentTab.connection.instanceId)
);

const allowReadOnlyMode = computed(() => {
  if (isDisconnected.value) return true;

  return instanceV1HasReadonlyMode(instance.value);
});

const alterSchemaState = reactive<AlterSchemaState>({
  showModal: false,
  databaseIdList: [],
});

const handleAlterSchema = async (params: {
  databaseId: string;
  schema: string;
  table: string;
}) => {
  const { databaseId, schema, table } = params;
  const database = databaseStore.getDatabaseByUID(databaseId);
  if (allowUsingSchemaEditorV1([database])) {
    // await useProjectV1Store().getOrFetchProjectByUID(
    //   String(database.projectEntity.uid)
    // );
    // TODO: support open selected database tab directly in Schema Editor.
    alterSchemaState.databaseIdList = [databaseId].map((uid) => Number(uid));
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
      databaseList: databaseId,
      sql: exampleSQL.join(" "),
    };
    const url = `/issue/new?${stringify(query)}`;
    window.open(url, "_blank");
  }
};

// provide context for SheetPanel
provideSheetPanelContext();
</script>

<style>
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
  @apply bg-indigo-400;
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

<style scoped>
.sqleditor--wrapper {
  color: var(--base);
  --base: #444;
  --font-code: "Source Code Pro", monospace;
  --color-branding: #4f46e5;
  --border-color: rgba(200, 200, 200, 0.2);

  @apply flex-1 overflow-hidden flex flex-col pt-2;
}
</style>
