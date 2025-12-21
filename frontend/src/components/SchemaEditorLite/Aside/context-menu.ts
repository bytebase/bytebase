import Emittery from "emittery";
import type { DropdownOption } from "naive-ui";
import { computed, type Ref, reactive } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { isDescendantOf } from "@/utils";
import { useSchemaEditorContext } from "../context";
import { engineSupportsMultiSchema } from "../spec";
import type {
  TreeNode,
  TreeNodeForDatabase,
  TreeNodeForFunction,
  TreeNodeForGroup,
  TreeNodeForProcedure,
  TreeNodeForSchema,
  TreeNodeForTable,
  TreeNodeForView,
} from "./common";

interface TreeContextMenu {
  show: boolean;
  clientX: number;
  clientY: number;
  node?: TreeNode;
}

type TreeContextMenuEvents = Emittery<{
  "create-schema": TreeNodeForDatabase;
  "restore-schema": TreeNodeForSchema;
  "drop-schema": TreeNodeForSchema;
  "create-table": TreeNodeForGroup<"table">;
  "rename-table": TreeNodeForTable;
  "restore-table": TreeNodeForTable;
  "drop-table": TreeNodeForTable;
  "create-procedure": TreeNodeForGroup<"procedure">;
  "restore-procedure": TreeNodeForProcedure;
  "drop-procedure": TreeNodeForProcedure;
  "create-view": TreeNodeForGroup<"view">;
  "restore-view": TreeNodeForView;
  "drop-view": TreeNodeForView;
  "create-function": TreeNodeForGroup<"function">;
  "restore-function": TreeNodeForFunction;
  "drop-function": TreeNodeForFunction;
  show: {
    e: MouseEvent;
    node: TreeNode;
  };
  hide: undefined;
}>;

export type ContextMenuContext = {
  menu: Ref<TreeContextMenu>;
  options: Ref<DropdownOption[]>;
  events: TreeContextMenuEvents;
  handleShow: (e: MouseEvent, node: TreeNode) => void;
  handleSelect: (key: string) => void;
  handleClickOutside: (e: MouseEvent) => void;
};

export const useContextMenu = (): ContextMenuContext => {
  const menu = reactive<TreeContextMenu>({
    show: false,
    clientX: 0,
    clientY: 0,
    node: undefined,
  });
  const events: TreeContextMenuEvents = new Emittery();
  const { t } = useI18n();
  const {
    getSchemaStatus,
    getTableStatus,
    getViewStatus,
    getProcedureStatus,
    getFunctionStatus,
  } = useSchemaEditorContext();

  const options = computed((): DropdownOption[] => {
    const { node } = menu;
    if (!node) return [];
    if (typeof node.db === "undefined") return [];

    const { engine } = (node.db as ComposedDatabase).instanceResource;
    if (node.type === "database") {
      if (engineSupportsMultiSchema(engine)) {
        return [
          {
            key: "create-schema",
            label: t("schema-editor.actions.create-schema"),
          },
        ];
      }
      return [];
    }
    if (node.type === "schema") {
      const options: DropdownOption[] = [];
      if (engine === Engine.POSTGRES) {
        const status = getSchemaStatus(node.db, node.metadata);
        if (status === "dropped") {
          options.push({
            key: "restore-schema",
            label: t("schema-editor.actions.restore"),
          });
        } else {
          options.push({
            key: "drop-schema",
            label: t("schema-editor.actions.drop-schema"),
          });
        }
      }
      return options;
    }
    if (node.type === "group") {
      if (node.group === "table") {
        return [
          {
            key: "create-table",
            label: t("schema-editor.actions.create-table"),
          },
        ];
      }
      if (node.group === "view") {
        return [
          {
            key: "create-view",
            label: t("schema-editor.actions.create-view"),
          },
        ];
      }
      if (node.group === "procedure") {
        return [
          {
            key: "create-procedure",
            label: t("schema-editor.actions.create-procedure"),
          },
        ];
      }
      if (node.group === "function") {
        return [
          {
            key: "create-function",
            label: t("schema-editor.actions.create-function"),
          },
        ];
      }
    }
    if (node.type === "table") {
      const options: DropdownOption[] = [];
      const status = getTableStatus(node.db, node.metadata);
      if (status === "dropped") {
        options.push({
          key: "restore-table",
          label: t("schema-editor.actions.restore"),
        });
      } else {
        if (status !== "normal") {
          options.push({
            key: "rename-table",
            label: t("schema-editor.actions.rename"),
          });
        }
        options.push({
          key: "drop-table",
          label: t("schema-editor.actions.drop-table"),
        });
      }
      return options;
    }
    if (node.type === "view") {
      const options: DropdownOption[] = [];
      const status = getViewStatus(node.db, node.metadata);
      if (status === "dropped") {
        options.push({
          key: "restore-view",
          label: t("schema-editor.actions.restore"),
        });
      } else {
        options.push({
          key: "drop-view",
          label: t("schema-editor.actions.drop"),
        });
      }
      return options;
    }
    if (node.type === "procedure") {
      const options: DropdownOption[] = [];
      const status = getProcedureStatus(node.db, node.metadata);
      if (status === "dropped") {
        options.push({
          key: "restore-procedure",
          label: t("schema-editor.actions.restore"),
        });
      } else {
        options.push({
          key: "drop-procedure",
          label: t("schema-editor.actions.drop"),
        });
      }
      return options;
    }
    if (node.type === "function") {
      const options: DropdownOption[] = [];
      const status = getFunctionStatus(node.db, node.metadata);
      if (status === "dropped") {
        options.push({
          key: "restore-function",
          label: t("schema-editor.actions.restore"),
        });
      } else {
        options.push({
          key: "drop-function",
          label: t("schema-editor.actions.drop"),
        });
      }
      return options;
    }

    return [];
  });

  const handleShow = (e: MouseEvent, node: TreeNode) => {
    e.preventDefault();
    e.stopPropagation();
    menu.node = node;
    menu.show = true;
    menu.clientX = e.clientX;
    menu.clientY = e.clientY;
    events.emit("show", { e, node });
  };

  const handleSelect = async (key: string) => {
    const node = menu.node;
    if (!node) return;
    const emit = events.emit.bind(events) as (
      event: string,
      data: TreeNode
    ) => Promise<void>;
    emit(key, node);
    menu.show = false;
  };

  const handleClickOutside = (e: MouseEvent) => {
    if (
      !isDescendantOf(e.target as Element, ".n-tree-node-wrapper") ||
      e.button !== 2
    ) {
      events.emit("hide");
      menu.show = false;
    }
  };

  return {
    menu: computed(() => menu),
    options,
    events,
    handleShow,
    handleSelect,
    handleClickOutside,
  };
};
