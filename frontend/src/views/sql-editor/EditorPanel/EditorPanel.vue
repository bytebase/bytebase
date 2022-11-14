<template>
  <div class="flex h-full w-full flex-col justify-start items-start">
    <EditorAction @save-sheet="handleSaveSheet" />

    <div
      v-if="!tabStore.isDisconnected"
      class="w-full py-1 px-4 flex justify-between items-center border-b"
      :class="[isProtectedEnvironment && 'bg-yellow-50']"
    >
      <div :class="[isProtectedEnvironment ? 'text-yellow-700' : 'text-main']">
        <template v-if="isProtectedEnvironment">
          {{ $t("sql-editor.sql-execute-in-protected-environment") }}
        </template>
      </div>

      <div class="action-right flex justify-end items-center">
        <NPopover
          v-if="selectedInstance.id !== UNKNOWN_ID && !hasReadonlyDataSource"
          trigger="hover"
        >
          <template #trigger>
            <heroicons-outline:exclamation
              class="h-6 w-6 flex-shrink-0 mr-2"
              :class="[
                isProtectedEnvironment ? 'text-yellow-500' : 'text-yellow-500',
              ]"
            />
          </template>
          <p class="py-1">
            {{ $t("instance.no-read-only-data-source-warn") }}
            <span
              class="underline text-accent cursor-pointer hover:opacity-80"
              @click="gotoInstanceDetailPage"
            >
              {{ $t("sql-editor.create-read-only-data-source") }}
            </span>
          </p>
        </NPopover>

        <NPopover trigger="hover" placement="bottom" :show-arrow="false">
          <template #trigger>
            <label class="flex items-center text-sm space-x-1">
              <div
                v-if="selectedInstance.id !== UNKNOWN_ID"
                class="flex items-center"
              >
                <span class="">{{ selectedInstance.environment.name }}</span>
                <ProtectedEnvironmentIcon
                  :environment="selectedInstance.environment"
                  class="ml-1"
                  :class="[isProtectedEnvironment && '~!text-yellow-700']"
                />
              </div>
              <div
                v-if="selectedInstance.id !== UNKNOWN_ID"
                class="flex items-center"
              >
                <span class="mx-2">/</span>
                <InstanceEngineIcon :instance="selectedInstance" show-status />
                <span class="ml-2">{{ selectedInstance.name }}</span>
              </div>
              <div
                v-if="selectedDatabase.id !== UNKNOWN_ID"
                class="flex items-center"
              >
                <span class="mx-2">/</span>
                <heroicons-outline:database />
                <span class="ml-2">{{ selectedDatabase.name }}</span>
              </div>
            </label>
          </template>
          <section>
            <div class="space-y-2">
              <div
                v-if="selectedInstance.id !== UNKNOWN_ID"
                class="flex flex-col"
              >
                <h1 class="text-gray-400">{{ $t("common.environment") }}</h1>
                <span class="flex items-center">
                  {{ selectedInstance.environment.name }}
                  <ProtectedEnvironmentIcon
                    :environment="selectedInstance.environment"
                    class="ml-1"
                  />
                </span>
              </div>
              <div
                v-if="selectedInstance.id !== UNKNOWN_ID"
                class="flex flex-col"
              >
                <h1 class="text-gray-400">{{ $t("common.instance") }}</h1>
                <span>{{ selectedInstance.name }}</span>
              </div>
              <div
                v-if="selectedDatabase.id !== UNKNOWN_ID"
                class="flex flex-col"
              >
                <h1 class="text-gray-400">{{ $t("common.database") }}</h1>
                <span>{{ selectedDatabase.name }}</span>
              </div>
            </div>
          </section>
        </NPopover>
      </div>
    </div>

    <template v-if="!tabStore.isDisconnected">
      <SQLEditor @save-sheet="handleSaveSheet" />
    </template>
    <template v-else>
      <ConnectionHolder />
    </template>

    <BBModal
      v-if="sqlEditorStore.isShowExecutingHint"
      :title="$t('common.tips')"
      @close="handleClose"
    >
      <ExecuteHint @close="handleClose" />
    </BBModal>
    <BBModal
      v-if="isShowSaveSheetModal"
      :title="$t('sql-editor.save-sheet')"
      @close="handleCloseModal"
    >
      <SaveSheetModal @close="handleCloseModal" @save-sheet="handleSaveSheet" />
    </BBModal>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { NPopover } from "naive-ui";
import { useRouter } from "vue-router";

import {
  useTabStore,
  useSQLEditorStore,
  useSheetStore,
  useDatabaseStore,
  useInstanceById,
  useDatabaseById,
} from "@/store";
import { defaultTabName } from "@/utils/tab";
import EditorAction from "./EditorAction.vue";
import SQLEditor from "./SQLEditor.vue";
import ExecuteHint from "./ExecuteHint.vue";
import ConnectionHolder from "./ConnectionHolder.vue";
import SaveSheetModal from "./SaveSheetModal.vue";
import type { SheetUpsert } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { instanceSlug } from "@/utils";

const router = useRouter();
const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();
const sheetStore = useSheetStore();
const databaseStore = useDatabaseStore();

const isShowSaveSheetModal = ref(false);

const connection = computed(() => tabStore.currentTab.connection);

const selectedInstance = useInstanceById(
  computed(() => connection.value.instanceId)
);
const selectedDatabase = useDatabaseById(
  computed(() => connection.value.databaseId)
);
const isProtectedEnvironment = computed(() => {
  const instance = selectedInstance.value;
  return instance.environment.tier === "PROTECTED";
});

const hasReadonlyDataSource = computed(() => {
  for (const ds of selectedInstance.value.dataSourceList) {
    if (ds.type === "RO") {
      return true;
    }
  }
  return false;
});

const gotoInstanceDetailPage = () => {
  const route = router.resolve({
    name: "workspace.instance.detail",
    params: {
      instanceSlug: instanceSlug(selectedInstance.value),
    },
  });
  window.open(route.href);
};

const allowSave = computed((): boolean => {
  const tab = tabStore.currentTab;
  if (tab.statement === "") {
    return false;
  }
  if (tab.isSaved) {
    return false;
  }
  // Temporarily disable saving and sharing if we are connected to an instance
  // but not a database.
  if (tab.connection.databaseId === UNKNOWN_ID) {
    return false;
  }
  return true;
});

const handleClose = () => {
  sqlEditorStore.setSQLEditorState({
    isShowExecutingHint: false,
  });
};

const handleSaveSheet = async (sheetName?: string) => {
  if (!allowSave.value) {
    return;
  }

  if (tabStore.currentTab.name === defaultTabName.value && !sheetName) {
    isShowSaveSheetModal.value = true;
    return;
  }
  isShowSaveSheetModal.value = false;

  const { name, statement, sheetId } = tabStore.currentTab;
  sheetName = sheetName ? sheetName : name;

  const conn = tabStore.currentTab.connection;
  const database = await databaseStore.getOrFetchDatabaseById(conn.databaseId);
  const sheetUpsert: SheetUpsert = {
    id: sheetId,
    projectId: database.project.id,
    databaseId: conn.databaseId,
    name: sheetName,
    statement: statement,
  };
  const sheet = await sheetStore.upsertSheet(sheetUpsert);

  tabStore.updateCurrentTab({
    sheetId: sheet.id,
    isSaved: true,
    name: sheetName,
  });
};

const handleCloseModal = () => {
  isShowSaveSheetModal.value = false;
};
</script>
