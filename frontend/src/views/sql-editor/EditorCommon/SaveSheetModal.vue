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
import {
  useDatabaseV1Store,
  useWorkSheetStore,
  useSQLEditorTabStore,
} from "@/store";
import type { SQLEditorTab } from "@/types";
import { UNKNOWN_ID } from "@/types";
import {
  Worksheet,
  Worksheet_Visibility,
} from "@/types/proto/v1/worksheet_service";
import { extractWorksheetUID } from "@/utils";
import { useSheetContext } from "../Sheet";
import { useSQLEditorContext } from "../context";
import SaveSheetForm from "./SaveSheetForm.vue";

type LocalState = {
  pendingEditTab?: SQLEditorTab;
};

const tabStore = useSQLEditorTabStore();
const databaseStore = useDatabaseV1Store();
const worksheetV1Store = useWorkSheetStore();
const { events: sheetEvents } = useSheetContext();
const { events: editorEvents } = useSQLEditorContext();

const state = reactive<LocalState>({});

const doSaveSheet = async (tab: SQLEditorTab) => {
  const { title, statement, sheet } = tab;

  if (title === "" || statement === "") {
    return;
  }

  const sheetId = Number(extractWorksheetUID(sheet ?? ""));

  if (sheetId !== UNKNOWN_ID) {
    const updatedSheet = await worksheetV1Store.patchSheet(
      {
        name: sheet,
        title: title,
        content: new TextEncoder().encode(statement),
      },
      ["title", "content"]
    );
    if (updatedSheet) {
      const tab = tabStore.tabList.find((t) => t.sheet === updatedSheet.name);
      if (tab) {
        tabStore.updateTab(tab.id, {
          title,
          status: "CLEAN",
        });
      }
    }
  } else {
    if (!tab.connection.database) {
      return false;
    }
    const database = await databaseStore.getOrFetchDatabaseByName(
      tab.connection.database,
      true /* silent */
    );
    const createdSheet = await worksheetV1Store.createSheet(
      Worksheet.fromPartial({
        title,
        project: database.project,
        content: new TextEncoder().encode(statement),
        database: database.name,
        visibility: Worksheet_Visibility.VISIBILITY_PRIVATE,
      })
    );
    if (tabStore.currentTabId === tab.id) {
      tabStore.updateCurrentTab({
        sheet: createdSheet.name,
        title,
        status: "CLEAN",
      });
    }
  }

  // Refresh "my" sheet list.
  sheetEvents.emit("refresh", { views: ["my"] });
  state.pendingEditTab = undefined;
};

const needSheetTitle = (tab: SQLEditorTab) => {
  if (tab.sheet) {
    // If the sheet is saved, we don't need to show the name popup.
    return false;
  }
  return true;
};

const trySaveSheet = (tab: SQLEditorTab, editTitle?: boolean) => {
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
