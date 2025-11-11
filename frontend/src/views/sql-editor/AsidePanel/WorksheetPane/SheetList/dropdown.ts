import { type DropdownOption } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { t } from "@/plugins/i18n";
import { useCurrentUserV1 } from "@/store";
import { type Position } from "@/types";
import { isWorksheetWritableV1 } from "@/utils";
import { type WorsheetFolderNode } from "@/views/sql-editor/Sheet";

export type DropdownOptionType =
  | "share"
  | "rename"
  | "delete"
  | "addfolder"
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

export const useDropdown = () => {
  const me = useCurrentUserV1();

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
      // DO NOT change the showDropdown manually.
      context.showDropdown = !!node;
      if (!node) {
        context.showSharePanel = false;
      }
    }
  );

  const isCreator = computed(() => {
    return context.node?.worksheet?.creator === `users/${me.value.email}`;
  });

  const options = computed((): WorksheetDropdown[] => {
    if (!context.node) {
      return [];
    }

    const items: WorksheetDropdown[] = [];
    if (context.node.worksheet) {
      items.push({
        key: "duplicate",
        label: isCreator.value ? t("common.duplicate") : t("common.fork"),
      });
      if (isCreator.value) {
        items.push({
          key: "share",
          label: t("common.share"),
        });
      }
      const canWriteSheet = isWorksheetWritableV1(context.node.worksheet);
      if (canWriteSheet) {
        items.push(
          {
            key: "rename",
            label: t("sql-editor.tab.context-menu.actions.rename"),
          },
          {
            key: "delete",
            label: t("common.delete"),
          }
        );
      }
    } else {
      items.push({
        key: "addfolder",
        label: t("sql-editor.tab.context-menu.actions.add-folder"),
      });
      if (context.node.editable) {
        items.push(
          {
            key: "rename",
            label: t("sql-editor.tab.context-menu.actions.rename"),
          },
          {
            key: "delete",
            label: t("common.delete"),
          }
        );
      }
    }

    return items;
  });

  const handleShow = (e: MouseEvent, node: WorsheetFolderNode) => {
    e.preventDefault();
    e.stopPropagation();
    context.node = node;
    context.position = {
      x: e.clientX,
      y: e.clientY,
    };
  };

  const handleClickOutside = () => {
    context.node = undefined;
  };

  return {
    context,
    options,
    handleShow,
    handleClickOutside,
  };
};
