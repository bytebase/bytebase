<template>
  <BBModal
    v-if="state.showModal"
    :title="$t('sql-editor.save-sheet')"
    @close="close"
  >
    <SaveSheetForm @close="close" @save-sheet="trySaveSheet" />
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";

import type { SheetUpsert } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { useDatabaseStore, useSheetStore, useTabStore } from "@/store";
import { defaultTabName } from "@/utils";
import SaveSheetForm from "./SaveSheetForm.vue";

type LocalState = {
  showModal: boolean;
};

const tabStore = useTabStore();
const databaseStore = useDatabaseStore();
const sheetStore = useSheetStore();

const state = reactive<LocalState>({
  showModal: false,
});

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

const doSaveSheet = async (sheetName?: string) => {
  const { name, statement, sheetId } = tabStore.currentTab;
  sheetName = sheetName || name;

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

const trySaveSheet = (sheetName?: string) => {
  if (!allowSave.value) {
    return;
  }

  if (tabStore.currentTab.name === defaultTabName.value && !sheetName) {
    state.showModal = true;
    return;
  }
  state.showModal = false;

  doSaveSheet(sheetName);
};

const close = () => {
  state.showModal = false;
};

defineExpose({
  trySaveSheet,
});
</script>
