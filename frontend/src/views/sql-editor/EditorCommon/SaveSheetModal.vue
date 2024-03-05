<template>
  <BBModal
    v-if="state.pendingEditTab"
    :title="$t('sql-editor.save-sheet')"
    @close="close"
  >
    <SaveSheetForm
      :tab="state.pendingEditTab"
      @close="close"
      @confirm="doSaveSheet"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useDatabaseV1Store, useWorkSheetStore, useTabStore } from "@/store";
import { UNKNOWN_ID, TabInfo } from "@/types";
import {
  Worksheet,
  Worksheet_Visibility,
} from "@/types/proto/v1/worksheet_service";
import { extractWorksheetUID } from "@/utils";
import { useSheetContext } from "../Sheet";
import { useSQLEditorContext } from "../context";
import SaveSheetForm from "./SaveSheetForm.vue";

type LocalState = {
  pendingEditTab?: TabInfo;
};

const tabStore = useTabStore();
const databaseStore = useDatabaseV1Store();
const worksheetV1Store = useWorkSheetStore();
const { events: sheetEvents } = useSheetContext();
const { events: editorEvents } = useSQLEditorContext();

const state = reactive<LocalState>({});

const doSaveSheet = async (tab: TabInfo) => {
  const { name, statement, sheetName } = tab;

  if (name === "" || statement === "") {
    return;
  }

  const sheetId = Number(extractWorksheetUID(sheetName ?? ""));

  if (sheetId !== UNKNOWN_ID) {
    const sheet = await worksheetV1Store.patchSheet(
      {
        name: sheetName,
        title: name,
        content: new TextEncoder().encode(statement),
      },
      ["title", "content"]
    );
    if (sheet) {
      const tab = tabStore.tabList.find((t) => t.sheetName === sheet.name);
      if (tab) {
        tabStore.updateTab(tab.id, {
          isSaved: true,
          name,
        });
      }
    }
  } else {
    if (tab.connection.databaseId === String(UNKNOWN_ID)) {
      return false;
    }
    const database = await databaseStore.getOrFetchDatabaseByUID(
      tab.connection.databaseId,
      true /* silent */
    );
    const sheet = await worksheetV1Store.createSheet(
      Worksheet.fromPartial({
        title: name,
        project: database.project,
        content: new TextEncoder().encode(statement),
        database: database.name,
        visibility: Worksheet_Visibility.VISIBILITY_PRIVATE,
      })
    );
    if (tabStore.currentTabId === tab.id) {
      tabStore.updateCurrentTab({
        sheetName: sheet.name,
        isSaved: true,
        name,
      });
    }
  }

  // Refresh "my" sheet list.
  sheetEvents.emit("refresh", { views: ["my"] });
  state.pendingEditTab = undefined;
};

const needSheetTitle = (tab: TabInfo) => {
  if (tab.sheetName) {
    // If the sheet is saved, we don't need to show the name popup.
    return false;
  }
  return true;
};

const trySaveSheet = (tab: TabInfo, editTitle?: boolean) => {
  if (needSheetTitle(tab) || editTitle) {
    state.pendingEditTab = tab;
    return;
  }
  state.pendingEditTab = undefined;

  doSaveSheet(tab);
};

const close = () => {
  state.pendingEditTab = undefined;
};

useEmitteryEventListener(editorEvents, "save-sheet", ({ tab, editTitle }) => {
  trySaveSheet(tab, editTitle);
});
</script>
