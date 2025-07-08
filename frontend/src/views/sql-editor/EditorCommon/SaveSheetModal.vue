<template>
  <BBModal
    v-if="state.pendingEdit"
    :title="$t('sql-editor.save-sheet')"
    @close="close"
  >
    <SaveSheetForm
      :tab="state.pendingEdit.tab"
      :mask="state.pendingEdit.mask"
      @close="close"
      @confirm="doSaveSheet"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { reactive } from "vue";
import { BBModal } from "@/bbkit";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useDatabaseV1Store,
  useWorkSheetStore,
  useSQLEditorTabStore,
} from "@/store";
import type { SQLEditorTab } from "@/types";
import { UNKNOWN_ID } from "@/types";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";
import {
  WorksheetSchema,
  Worksheet_Visibility,
} from "@/types/proto-es/v1/worksheet_service_pb";
import { extractWorksheetUID } from "@/utils";
import { useSheetContext } from "../Sheet";
import { useSQLEditorContext } from "../context";
import SaveSheetForm from "./SaveSheetForm.vue";

type LocalState = {
  pendingEdit?: {
    tab: SQLEditorTab;
    mask?: Array<keyof Worksheet>;
  };
};

const tabStore = useSQLEditorTabStore();
const databaseStore = useDatabaseV1Store();
const worksheetV1Store = useWorkSheetStore();
const { events: sheetEvents } = useSheetContext();
const { events: editorEvents } = useSQLEditorContext();

const state = reactive<LocalState>({});

const doSaveSheet = async (
  tab: SQLEditorTab,
  mask?: Array<keyof Worksheet>
) => {
  const { title, statement, worksheet } = tab;

  if (title === "") {
    return;
  }

  const sheetId = Number(extractWorksheetUID(worksheet ?? ""));

  if (sheetId !== UNKNOWN_ID) {
    const currentSheet = await worksheetV1Store.getWorksheetByName(worksheet);
    if (!currentSheet) return;

    const updatedSheet = await worksheetV1Store.patchWorksheet(
      {
        ...currentSheet,
        title: title,
        database: tab.connection.database,
        content: new TextEncoder().encode(statement),
      },
      mask ?? ["title", "content", "database"]
    );
    if (updatedSheet) {
      const tab = tabStore.tabList.find(
        (t) => t.worksheet === updatedSheet.name
      );
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
    const createdSheet = await worksheetV1Store.createWorksheet(
      create(WorksheetSchema, {
        title,
        project: database.project,
        content: new TextEncoder().encode(statement),
        database: database.name,
        visibility: Worksheet_Visibility.PRIVATE,
      })
    );
    if (tabStore.currentTabId === tab.id) {
      tabStore.updateCurrentTab({
        worksheet: createdSheet.name,
        title,
        status: "CLEAN",
      });
    }
  }

  // Refresh "my" sheet list.
  sheetEvents.emit("refresh", { views: ["my"] });
  state.pendingEdit = undefined;
};

const needSheetTitle = (tab: SQLEditorTab) => {
  if (tab.worksheet) {
    // If the sheet is saved, we don't need to show the name popup.
    return false;
  }
  return true;
};

const trySaveSheet = (
  tab: SQLEditorTab,
  editTitle?: boolean,
  mask?: Array<keyof Worksheet>
) => {
  if (needSheetTitle(tab) || editTitle) {
    state.pendingEdit = {
      tab,
      mask,
    };
    return;
  }
  state.pendingEdit = undefined;

  doSaveSheet(tab, mask);
};

const close = () => {
  state.pendingEdit = undefined;
};

useEmitteryEventListener(
  editorEvents,
  "save-sheet",
  ({ tab, editTitle, mask }) => {
    trySaveSheet(tab, editTitle, mask);
  }
);
</script>
