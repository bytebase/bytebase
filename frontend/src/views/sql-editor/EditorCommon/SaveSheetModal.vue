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
import { NButton, NInput } from "naive-ui";
import { ref } from "vue";
import { BBModal } from "@/bbkit";
import RequiredStar from "@/components/RequiredStar.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useWorkSheetStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { extractWorksheetUID } from "@/utils";
import FolderForm from "@/views/sql-editor/AsidePanel/WorksheetPane/SheetList/FolderForm.vue";
import { useSheetContextByView } from "@/views/sql-editor/Sheet";
import { useSQLEditorContext } from "../context";

const worksheetV1Store = useWorkSheetStore();
const editorContext = useSQLEditorContext();
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
  if (pendingEdit.value.title === "") {
    return;
  }

  const {
    worksheet,
    connection,
    statement,
    id: tabId,
  } = pendingEdit.value.rawTab;
  const folders = folderFormRef.value?.folders ?? [];

  const sheetId = Number(extractWorksheetUID(worksheet ?? ""));
  if (sheetId !== UNKNOWN_ID) {
    await editorContext.maybeUpdateWorksheet({
      tabId,
      worksheet,
      title: pendingEdit.value.title,
      database: connection.database,
      statement,
      folders,
    });
  } else {
    await editorContext.createWorksheet({
      tabId,
      title: pendingEdit.value.title,
      statement: pendingEdit.value.rawTab.statement,
      database: connection.database,
      folders,
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

useEmitteryEventListener(
  editorContext.events,
  "save-sheet",
  ({ tab, editTitle }) => {
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
  }
);
</script>
