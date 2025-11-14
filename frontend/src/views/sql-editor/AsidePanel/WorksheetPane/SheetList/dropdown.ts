import {
  FileCodeIcon,
  FilesIcon,
  FolderIcon,
  PencilLineIcon,
  Share2Icon,
  TrashIcon,
} from "lucide-vue-next";
import { type DropdownOption } from "naive-ui";
import { computed, h, reactive, watch } from "vue";
import { t } from "@/plugins/i18n";
import { useCurrentUserV1, useWorkSheetStore } from "@/store";
import { type Position } from "@/types";
import { isWorksheetWritableV1 } from "@/utils";
import type {
  SheetViewMode,
  WorsheetFolderNode,
} from "@/views/sql-editor/Sheet";

export type DropdownOptionType =
  | "share"
  | "rename"
  | "delete"
  | "add-folder"
  | "add-worksheet"
  | "duplicate";

type WorksheetDropdown = DropdownOption & {
  key: DropdownOptionType;
};

interface DropdownContext {
  position: Position;
  showDropdown: boolean;
  showSharePanel: boolean;
  node?: WorsheetFolderNode;
}

export const useDropdown = (viewMode: SheetViewMode) => {
  const me = useCurrentUserV1();
  const sheetStore = useWorkSheetStore();

  const context = reactive<DropdownContext>({
    position: {
      x: 0,
      y: 0,
    },
    showDropdown: false,
    showSharePanel: false,
  });

  watch(
    () => context.node,
    (node) => {
      if (!node) {
        context.showDropdown = false;
        context.showSharePanel = false;
      }
    }
  );

  watch(
    () => context.showSharePanel,
    (show) => {
      if (show) {
        context.showDropdown = false;
      }
    }
  );

  watch(
    () => context.showDropdown,
    (show) => {
      if (show) {
        context.showSharePanel = false;
      }
    }
  );

  const worksheetEntity = computed(() => {
    if (viewMode === "draft" || !context.node?.worksheet) {
      return undefined;
    }
    return sheetStore.getWorksheetByName(context.node.worksheet.name);
  });

  const options = computed((): WorksheetDropdown[] => {
    if (viewMode === "draft" || !context.node) {
      return [];
    }

    const items: WorksheetDropdown[] = [];
    if (context.node.worksheet) {
      if (!worksheetEntity.value) {
        return [];
      }
      const isCreator =
        worksheetEntity.value.creator === `users/${me.value.email}`;
      items.push({
        icon: () => h(FilesIcon, { class: "w-4 text-gray-600" }),
        key: "duplicate",
        label: isCreator ? t("common.duplicate") : t("common.fork"),
      });
      if (isCreator) {
        items.push({
          icon: () => h(Share2Icon, { class: "w-4 text-gray-600" }),
          key: "share",
          label: t("common.share"),
        });
      }
      const canWriteSheet = isWorksheetWritableV1(worksheetEntity.value);
      if (canWriteSheet) {
        items.push(
          {
            icon: () => h(PencilLineIcon, { class: "w-4 text-gray-600" }),
            key: "rename",
            label: t("sql-editor.tab.context-menu.actions.rename"),
          },
          {
            icon: () => h(TrashIcon, { class: "w-4 text-gray-600" }),
            key: "delete",
            label: t("common.delete"),
          }
        );
      }
    } else {
      items.push({
        icon: () => h(FolderIcon, { class: "w-4 text-gray-600" }),
        key: "add-folder",
        label: t("sql-editor.tab.context-menu.actions.add-folder"),
      });
      if (viewMode === "my") {
        items.push({
          icon: () => h(FileCodeIcon, { class: "w-4 text-gray-600" }),
          key: "add-worksheet",
          label: t("sql-editor.tab.context-menu.actions.add-worksheet"),
        });
      }
      if (context.node.editable) {
        items.push(
          {
            icon: () => h(PencilLineIcon, { class: "w-4 text-gray-600" }),
            key: "rename",
            label: t("sql-editor.tab.context-menu.actions.rename"),
          },
          {
            icon: () => h(TrashIcon, { class: "w-4 text-gray-600" }),
            key: "delete",
            label: t("common.delete"),
          }
        );
      }
    }

    return items;
  });

  const resetData = (e: MouseEvent, node: WorsheetFolderNode) => {
    e.preventDefault();
    e.stopPropagation();
    context.node = node;
    context.position = {
      x: e.clientX,
      y: e.clientY,
    };
  };

  const handleMenuShow = (e: MouseEvent, node: WorsheetFolderNode) => {
    resetData(e, node);
    context.showDropdown = true;
  };

  const handleSharePanelShow = (e: MouseEvent, node: WorsheetFolderNode) => {
    resetData(e, node);
    context.showSharePanel = true;
  };

  const handleClickOutside = () => {
    context.node = undefined;
  };

  return {
    context,
    options,
    worksheetEntity,
    handleMenuShow,
    handleSharePanelShow,
    handleClickOutside,
  };
};
