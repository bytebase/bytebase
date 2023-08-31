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
import { useDatabaseV1Store, useSheetV1Store, useTabStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import {
  Sheet_Visibility,
  Sheet_Source,
  Sheet_Type,
  Sheet,
} from "@/types/proto/v1/sheet_service";
import {
  extractSheetUID,
  getSuggestedTabNameFromConnection,
  isSimilarDefaultTabName,
} from "@/utils";
import { useSheetContext } from "../Sheet";
import SaveSheetForm from "./SaveSheetForm.vue";

type LocalState = {
  showModal: boolean;
};

const tabStore = useTabStore();
const databaseStore = useDatabaseV1Store();
const sheetV1Store = useSheetV1Store();
const { events: sheetEvents } = useSheetContext();

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
  if (tab.connection.databaseId === String(UNKNOWN_ID)) {
    return false;
  }
  return true;
});

const doSaveSheet = async (sheetTitle?: string) => {
  const { name, statement, sheetName } = tabStore.currentTab;
  sheetTitle = sheetTitle || name;

  const sheetId = Number(extractSheetUID(sheetName ?? ""));

  const conn = tabStore.currentTab.connection;
  const database = await databaseStore.getOrFetchDatabaseByUID(conn.databaseId);

  let sheet: Sheet | undefined;
  if (sheetId !== UNKNOWN_ID) {
    sheet = await sheetV1Store.patchSheet({
      name: sheetName,
      database: database.name,
      title: sheetTitle,
      content: new TextEncoder().encode(statement),
    });
  } else {
    sheet = await sheetV1Store.createSheet(database.project, {
      title: sheetTitle,
      content: new TextEncoder().encode(statement),
      database: database.name,
      visibility: Sheet_Visibility.VISIBILITY_PRIVATE,
      source: Sheet_Source.SOURCE_BYTEBASE,
      type: Sheet_Type.TYPE_SQL,
      payload: "{}",
    });
  }

  if (sheet) {
    tabStore.updateCurrentTab({
      sheetName: sheet.name,
      isSaved: true,
      name: sheetTitle,
    });

    // Refresh "my" sheet list.
    sheetEvents.emit("refresh", { views: ["my"] });
  }
};

const needSheetName = (sheetName: string | undefined) => {
  const tab = tabStore.currentTab;
  if (tab.sheetName) {
    // If the sheet is saved, we don't need to show the name popup.
    return false;
  }
  if (!sheetName) {
    const name = tab.name;
    if (
      name === getSuggestedTabNameFromConnection(tab.connection) ||
      isSimilarDefaultTabName(name)
    ) {
      // The tab is unsaved and its name is still the default one.
      return true;
    }
  }
  return false;
};

const trySaveSheet = (sheetTitle?: string) => {
  if (!allowSave.value) {
    return;
  }

  if (needSheetName(sheetTitle)) {
    state.showModal = true;
    return;
  }
  state.showModal = false;

  doSaveSheet(sheetTitle);
};

const close = () => {
  state.showModal = false;
};

defineExpose({
  trySaveSheet,
});
</script>
