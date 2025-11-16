import {
  FileCodeIcon,
  FilesIcon,
  FolderIcon,
  ListCheckIcon,
  type LucideProps,
  PencilLineIcon,
  Share2Icon,
  TrashIcon,
} from "lucide-vue-next";
import { type DropdownOption } from "naive-ui";
import { computed, type FunctionalComponent, h, reactive, watch } from "vue";
import { t } from "@/plugins/i18n";
import { useCurrentUserV1, useWorkSheetStore } from "@/store";
import { type Position } from "@/types";
import { isWorksheetWritableV1 } from "@/utils";
import type {
  SheetViewMode,
  WorksheetFolderNode,
} from "@/views/sql-editor/Sheet";

export type DropdownOptionType =
  | "share"
  | "rename"
  | "delete"
  | "add-folder"
  | "add-worksheet"
  | "multi-select"
  | "duplicate";

type WorksheetDropdown = DropdownOption & {
  key: DropdownOptionType;
};

interface DropdownContext {
  position: Position;
  showDropdown: boolean;
  showSharePanel: boolean;
  node?: WorksheetFolderNode;
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

    const items: {
      icon: FunctionalComponent<
        LucideProps,
        Record<string, any>,
        any,
        Record<string, any>
      >;
      key: DropdownOptionType;
      label: string;
    }[] = [];
    if (context.node.worksheet) {
      if (!worksheetEntity.value) {
        return [];
      }
      const isCreator =
        worksheetEntity.value.creator === `users/${me.value.email}`;
      items.push({
        icon: FilesIcon,
        key: "duplicate",
        label: isCreator ? t("common.duplicate") : t("common.fork"),
      });
      if (isCreator) {
        items.push({
          icon: Share2Icon,
          key: "share",
          label: t("common.share"),
        });
      }
      const canWriteSheet = isWorksheetWritableV1(worksheetEntity.value);
      if (canWriteSheet) {
        items.push(
          {
            icon: PencilLineIcon,
            key: "rename",
            label: t("sql-editor.tab.context-menu.actions.rename"),
          },
          {
            icon: TrashIcon,
            key: "delete",
            label: t("common.delete"),
          }
        );
      }
    } else {
      items.push({
        icon: FolderIcon,
        key: "add-folder",
        label: t("sql-editor.tab.context-menu.actions.add-folder"),
      });
      if (viewMode === "my") {
        items.push(
          {
            icon: FileCodeIcon,
            key: "add-worksheet",
            label: t("sql-editor.tab.context-menu.actions.add-worksheet"),
          },
          {
            icon: ListCheckIcon,
            key: "multi-select",
            label: t("sql-editor.tab.context-menu.actions.multi-select"),
          }
        );
      }
      if (context.node.editable) {
        items.push(
          {
            icon: PencilLineIcon,
            key: "rename",
            label: t("sql-editor.tab.context-menu.actions.rename"),
          },
          {
            icon: TrashIcon,
            key: "delete",
            label: t("common.delete"),
          }
        );
      }
    }

    return items.map((item) => ({
      icon: () => h(item.icon, { class: "w-4 text-gray-600" }),
      key: item.key,
      label: item.label,
    }));
  });

  const resetData = (e: MouseEvent, node: WorksheetFolderNode) => {
    e.preventDefault();
    e.stopPropagation();
    context.node = node;
    context.position = {
      x: e.clientX,
      y: e.clientY,
    };
  };

  const handleMenuShow = (e: MouseEvent, node: WorksheetFolderNode) => {
    resetData(e, node);
    context.showDropdown = true;
  };

  const handleSharePanelShow = (e: MouseEvent, node: WorksheetFolderNode) => {
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
