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
      <div class="flex flex-col gap-y-1">
        <p>{{ $t("sql-editor.choose-folder") }}</p>
        <!-- TODO(ed): support creating new folders when saving the worksheet -->
        <NTreeSelect
          filterable
          clearable
          checkable
          show-path
          virtual-scroll
          :multiple="false"
          :options="[folderTree]"
          :check-strategy="'parent'"
          :render-prefix="renderPrefix"
          v-model:expanded-keys="expandedKeys"
          v-model:value="pendingEdit.folder"
        />
      </div>
      <div class="flex justify-end gap-x-2 mt-4">
        <NButton @click="close">{{ $t("common.close") }}</NButton>
        <NButton
          :disabled="!pendingEdit.title"
          type="primary"
          @click="() => doSaveSheet(pendingEdit)"
        >
          {{ $t("common.save") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import {
  FolderCodeIcon,
  FolderMinusIcon,
  FolderOpenIcon,
} from "lucide-vue-next";
import { NButton, NInput, NTreeSelect, type TreeOption } from "naive-ui";
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
import type { WorsheetFolderNode } from "@/views/sql-editor/Sheet";
import { useSheetContextByView } from "@/views/sql-editor/Sheet";
import { useSQLEditorContext } from "../context";

const tabStore = useSQLEditorTabStore();
const worksheetV1Store = useWorkSheetStore();
const { events: editorEvents } = useSQLEditorContext();
const { folderTree, folderContext, getPwdForWorksheet } =
  useSheetContextByView("my");

const pendingEdit = ref<{
  title: string;
  folder: string;
  rawTab?: SQLEditorTab;
}>({
  title: "",
  folder: "",
});
const expandedKeys = ref<string[]>([folderContext.rootPath.value]);
const showModal = ref(false);

const doSaveSheet = async ({
  title,
  folder,
  rawTab,
}: {
  title: string;
  folder: string;
  rawTab?: SQLEditorTab;
}) => {
  if (!rawTab) {
    return close();
  }
  const { statement, worksheet } = rawTab;

  if (title === "") {
    return;
  }

  const sheetId = Number(extractWorksheetUID(worksheet ?? ""));
  let worksheetEntity: Worksheet | undefined;

  if (sheetId !== UNKNOWN_ID) {
    const currentSheet = worksheetV1Store.getWorksheetByName(worksheet);
    if (!currentSheet) return;

    worksheetEntity = await worksheetV1Store.patchWorksheet(
      {
        ...currentSheet,
        title: title,
        database: rawTab.connection.database,
        content: new TextEncoder().encode(statement),
      },
      ["title", "content", "database"]
    );
  } else {
    worksheetEntity = await worksheetV1Store.createWorksheet(
      create(WorksheetSchema, {
        title,
        project: tabStore.project,
        content: new TextEncoder().encode(statement),
        database: rawTab.connection.database,
        visibility: Worksheet_Visibility.PRIVATE,
      })
    );
  }

  if (worksheetEntity) {
    if (folder && folder !== folderContext.rootPath.value) {
      const folders = folder
        .replace(folderContext.rootPath.value, "")
        .split("/")
        .filter((p) => p);
      await worksheetV1Store.upsertWorksheetOrganizer(
        {
          worksheet: worksheetEntity.name,
          starred: false,
          folders,
        },
        ["folders"]
      );
    }

    tabStore.updateTab(rawTab.id, {
      title,
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
    folder: folderContext.rootPath.value,
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
  doSaveSheet(pendingEdit.value);
});

const renderPrefix = ({ option }: { option: TreeOption }) => {
  const node = option as WorsheetFolderNode;
  if (node.worksheet) {
    return null;
  }
  if (expandedKeys.value.includes(node.key)) {
    // is opened folder
    return <FolderOpenIcon class="w-4 h-auto text-gray-600" />;
  }
  if (node.key === folderContext.rootPath.value) {
    // root folder icon
    return <FolderCodeIcon class="w-4 h-auto text-gray-600" />;
  }
  if (node.empty) {
    // empty folder icon
    return <FolderMinusIcon class="w-4 h-auto text-gray-600" />;
  }
  // fallback to normal folder icon
  return <FolderCodeIcon class="w-4 h-auto text-gray-600" />;
};
</script>
