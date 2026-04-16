import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { engineSupportsMultiSchema } from "@/components/SchemaEditorLite/spec";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { getDatabaseEngine } from "@/utils";
import type { EditStatusContext } from "../types";
import type { TreeNode } from "./tree-builder";

export interface MenuState {
  show: boolean;
  x: number;
  y: number;
  node?: TreeNode;
}

export interface MenuOption {
  key: string;
  label: string;
}

export function useContextMenu(editStatus: EditStatusContext) {
  const { t } = useTranslation();
  const [menuState, setMenuState] = useState<MenuState>({
    show: false,
    x: 0,
    y: 0,
  });

  const menuOptions = useMemo((): MenuOption[] => {
    const { node } = menuState;
    if (!node) return [];
    if (!("db" in node)) return [];

    const engine = getDatabaseEngine(node.db as Database);

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
      if (engine === Engine.POSTGRES) {
        const status = editStatus.getSchemaStatus(node.db, node.metadata);
        if (status === "dropped") {
          return [
            {
              key: "restore-schema",
              label: t("schema-editor.actions.restore"),
            },
          ];
        }
        return [
          {
            key: "drop-schema",
            label: t("schema-editor.actions.drop-schema"),
          },
        ];
      }
      return [];
    }

    if (node.type === "group") {
      const createLabels: Record<string, string> = {
        table: t("schema-editor.actions.create-table"),
        view: t("schema-editor.actions.create-view"),
        procedure: t("schema-editor.actions.create-procedure"),
        function: t("schema-editor.actions.create-function"),
      };
      return [
        {
          key: `create-${node.group}`,
          label: createLabels[node.group] ?? "",
        },
      ];
    }

    if (node.type === "table") {
      const status = editStatus.getTableStatus(node.db, node.metadata);
      if (status === "dropped") {
        return [
          {
            key: "restore-table",
            label: t("schema-editor.actions.restore"),
          },
        ];
      }
      const options: MenuOption[] = [];
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
      return options;
    }

    if (node.type === "view") {
      const status = editStatus.getViewStatus(node.db, node.metadata);
      if (status === "dropped") {
        return [
          {
            key: "restore-view",
            label: t("schema-editor.actions.restore"),
          },
        ];
      }
      return [{ key: "drop-view", label: t("schema-editor.actions.drop") }];
    }

    if (node.type === "procedure") {
      const status = editStatus.getProcedureStatus(node.db, node.metadata);
      if (status === "dropped") {
        return [
          {
            key: "restore-procedure",
            label: t("schema-editor.actions.restore"),
          },
        ];
      }
      return [
        { key: "drop-procedure", label: t("schema-editor.actions.drop") },
      ];
    }

    if (node.type === "function") {
      const status = editStatus.getFunctionStatus(node.db, node.metadata);
      if (status === "dropped") {
        return [
          {
            key: "restore-function",
            label: t("schema-editor.actions.restore"),
          },
        ];
      }
      return [{ key: "drop-function", label: t("schema-editor.actions.drop") }];
    }

    return [];
  }, [menuState, editStatus, t]);

  const showMenu = useCallback((e: React.MouseEvent, node: TreeNode) => {
    e.preventDefault();
    e.stopPropagation();
    setMenuState({ show: true, x: e.clientX, y: e.clientY, node });
  }, []);

  const hideMenu = useCallback(() => {
    setMenuState((prev) => ({ ...prev, show: false }));
  }, []);

  return { menuState, menuOptions, showMenu, hideMenu };
}
