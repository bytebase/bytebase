<template>
  <BBModal v-if="showModal" :title="$t('sql-editor.save-sheet')" @close="close">
    <div
      class="save-sheet-modal flex flex-col gap-y-3 w-lg max-w-[calc(100vw-8rem)]"
    >
      <div class="flex flex-col gap-y-1">
        <p>
          {{ $t("common.title") }}
          <RequiredStar />
        </p>
        <NInput
          ref="sheetTitleInputRef"
          v-model:value="pendingEdit.title"
          :placeholder="$t('sql-editor.save-sheet-input-placeholder')"
          :maxlength="200"
        />
      </div>
      <FolderForm ref="folderFormRef" :folder="pendingEdit.folder" />
      <div class="flex justify-end gap-x-2 mt-4">
        <NButton @click="close">{{ $t("common.close") }}</NButton>
        <NButton
          :disabled="!pendingEdit.title"
          type="primary"
          @click="doSaveSheet"
        >
          {{ $t("common.save") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import { NButton, NInput } from "naive-ui";
import { ref } from "vue";
import { BBModal } from "@/bbkit";
import RequiredStar from "@/components/RequiredStar.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useSQLEditorTabStore, useWorkSheetStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { UNKNOWN_ID } from "@/types";
import {
  type Worksheet,
  Worksheet_Visibility,
  WorksheetSchema,
} from "@/types/proto-es/v1/worksheet_service_pb";
import { extractWorksheetUID } from "@/utils";
import FolderForm from "@/views/sql-editor/AsidePanel/WorksheetPane/SheetList/FolderForm.vue";
import { useSheetContextByView } from "@/views/sql-editor/Sheet";
import { useSQLEditorContext } from "../context";

const tabStore = useSQLEditorTabStore();
const worksheetV1Store = useWorkSheetStore();
const { events: editorEvents } = useSQLEditorContext();
const { getPwdForWorksheet } = useSheetContextByView("my");
const folderFormRef = ref<InstanceType<typeof FolderForm>>();

const pendingEdit = ref<{
  title: string;
  folder: string;
  rawTab?: SQLEditorTab;
}>({
  title: "",
  folder: "",
});
const showModal = ref(false);

const doSaveSheet = async () => {
  if (!pendingEdit.value.rawTab) {
    return close();
  }
  const { statement, worksheet, connection } = pendingEdit.value.rawTab;

  if (pendingEdit.value.title === "") {
    return;
  }

  const sheetId = Number(extractWorksheetUID(worksheet ?? ""));
  let worksheetEntity: Worksheet | undefined;

  if (sheetId !== UNKNOWN_ID) {
    const currentSheet = worksheetV1Store.getWorksheetByName(worksheet);
    if (!currentSheet) {
      return;
    }

    worksheetEntity = await worksheetV1Store.patchWorksheet(
      {
        ...currentSheet,
        title: pendingEdit.value.title,
        database: connection.database,
        content: new TextEncoder().encode(statement),
      },
      ["title", "content", "database"]
    );
  } else {
    worksheetEntity = await worksheetV1Store.createWorksheet(
      create(WorksheetSchema, {
        title: pendingEdit.value.title,
        project: tabStore.project,
        content: new TextEncoder().encode(statement),
        database: connection.database,
        visibility: Worksheet_Visibility.PRIVATE,
      })
    );
  }

  if (worksheetEntity) {
    const folders = folderFormRef.value?.folders ?? [];
    if (folders.length > 0) {
      await worksheetV1Store.upsertWorksheetOrganizer(
        {
          worksheet: worksheetEntity.name,
          starred: false,
          folders,
        },
        ["folders"]
      );
    }

    tabStore.updateTab(pendingEdit.value.rawTab.id, {
      title: pendingEdit.value.title,
      status: "CLEAN",
      worksheet: worksheetEntity.name,
    });
  }

  showModal.value = false;
};

const needShowModal = (tab: SQLEditorTab) => {
  // If the sheet is saved, we don't need to show the name popup.
  return !tab.worksheet;
};

const close = () => {
  showModal.value = false;
};

useEmitteryEventListener(editorEvents, "save-sheet", ({ tab, editTitle }) => {
  pendingEdit.value = {
    title: tab.title,
    folder: "",
    rawTab: tab,
  };

  if (needShowModal(tab) || editTitle) {
    const worksheet = worksheetV1Store.getWorksheetByName(tab.worksheet);
    if (worksheet) {
      pendingEdit.value.folder = getPwdForWorksheet(worksheet);
    }
    showModal.value = true;
    return;
  }
  doSaveSheet();
});
</script>
