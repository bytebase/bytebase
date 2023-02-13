<template>
  <div class="sqleditor--wrapper">
    <TabList />
    <Splitpanes class="default-theme flex flex-col flex-1 overflow-hidden">
      <Pane size="20">
        <AsidePanel @alter-schema="handleAlterSchema" />
      </Pane>
      <Pane size="80" class="relative">
        <template v-if="allowAccess">
          <template v-if="tabStore.currentTab.mode === TabMode.ReadOnly">
            <Splitpanes
              v-if="allowReadOnlyMode"
              horizontal
              class="default-theme"
            >
              <Pane :size="isDisconnected ? 100 : 60">
                <EditorPanel />
              </Pane>
              <Pane :size="isDisconnected ? 0 : 40">
                <TablePanel />
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
                keypath="sql-editor.read-only-mode-not-allowed"
                tag="div"
              >
                <template #instance>
                  <span class="inline-flex items-center mx-1">
                    <InstanceEngineIcon :instance="instance" />
                    <span>{{ instanceName(instance) }}</span>
                  </span>
                </template>
              </i18n-t>
              <AdminModeButton />
            </div>
          </template>

          <TerminalPanel v-if="tabStore.currentTab.mode === TabMode.Admin" />
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

    <SchemaEditorModal
      v-if="alterSchemaState.showModal"
      :database-id-list="alterSchemaState.databaseIdList"
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

import { DatabaseId, TabMode, UNKNOWN_ID } from "@/types";
import {
  useConnectionTreeStore,
  useCurrentUser,
  useDatabaseStore,
  useInstanceById,
  useProjectStore,
  useSQLEditorStore,
  useTabStore,
} from "@/store";
import AsidePanel from "./AsidePanel/AsidePanel.vue";
import EditorPanel from "./EditorPanel/EditorPanel.vue";
import TerminalPanel from "./TerminalPanel/TerminalPanel.vue";
import TabList from "./TabList";
import TablePanel from "./TablePanel/TablePanel.vue";
import { allowUsingSchemaEditor, isDatabaseAccessible } from "@/utils";
import AdminModeButton from "./EditorCommon/AdminModeButton.vue";
import SchemaEditorModal from "@/components/AlterSchemaPrepForm/SchemaEditorModal.vue";

type AlterSchemaState = {
  showModal: boolean;
  databaseIdList: DatabaseId[];
};

const tabStore = useTabStore();
const databaseStore = useDatabaseStore();
const connectionTreeStore = useConnectionTreeStore();
const sqlEditorStore = useSQLEditorStore();
const currentUser = useCurrentUser();

const isDisconnected = computed(() => tabStore.isDisconnected);
const isFetchingSheet = computed(() => sqlEditorStore.isFetchingSheet);

const allowAccess = computed(() => {
  const { databaseId } = tabStore.currentTab.connection;
  const database = databaseStore.getDatabaseById(databaseId);
  if (database.id === UNKNOWN_ID) {
    // Allowed if connected to an instance
    return true;
  }
  const { accessControlPolicyList } = connectionTreeStore;
  return isDatabaseAccessible(
    database,
    accessControlPolicyList,
    currentUser.value
  );
});

const instance = useInstanceById(
  computed(() => tabStore.currentTab.connection.instanceId)
);

const allowReadOnlyMode = computed(() => {
  if (isDisconnected.value) return true;

  if (instance.value.engine === "MONGODB") {
    return false;
  }
  return true;
});

const alterSchemaState = reactive<AlterSchemaState>({
  showModal: false,
  databaseIdList: [],
});

const handleAlterSchema = async (params: {
  databaseId: DatabaseId;
  schema: string;
  table: string;
}) => {
  const { databaseId, schema, table } = params;
  const database = databaseStore.getDatabaseById(databaseId);
  if (allowUsingSchemaEditor([database])) {
    await useProjectStore().getOrFetchProjectById(database.project.id);
    // TODO: support open selected database tab directly in Schema Editor.
    alterSchemaState.databaseIdList = [databaseId];
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
      project: database.project.id,
      databaseList: databaseId,
      sql: exampleSQL.join(" "),
    };
    const url = `/issue/new?${stringify(query)}`;
    window.open(url, "_blank");
  }
};
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

  @apply flex-1 overflow-hidden flex flex-col;
}
</style>
