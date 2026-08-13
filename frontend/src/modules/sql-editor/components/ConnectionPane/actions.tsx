import {
  ExternalLink,
  Link as LinkIcon,
  SquarePen,
  Wrench,
} from "lucide-react";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { useSQLEditorAllowAdmin } from "@/modules/sql-editor/hooks/useSQLEditorState";
import { sqlEditorEvents } from "@/modules/sql-editor/model/events";
import { useSQLEditorStore } from "@/modules/sql-editor/store";
import {
  getSQLEditorEditorState,
  useSQLEditorEditorState,
} from "@/modules/sql-editor/store/editor";
import { getSQLEditorTabsState } from "@/modules/sql-editor/store/tab";
import type {
  BatchQueryContext,
  SQLEditorConnection,
  SQLEditorTabMode,
  SQLEditorTreeNode,
} from "@/types";
import {
  instanceOfSQLEditorTreeNode,
  isConnectableSQLEditorTreeNode,
} from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  autoDatabaseRoute,
  canCreateSavedQueryInProject,
  extractDatabaseResourceName,
  getInstanceResource,
  instanceV1HasAlterSchema,
  instanceV1HasReadonlyMode,
} from "@/utils";

type ActionKey =
  | "connect"
  | "connect-in-new-tab"
  | "connect-in-admin-mode"
  | "view-database-detail"
  | "alter-schema";

export type ConnectionMenuItem = {
  readonly key: ActionKey;
  readonly label: string;
  readonly icon: React.ReactNode;
  readonly onSelect: () => void;
};

const DEFAULT_TAB_MODE: SQLEditorTabMode = "SAVED_QUERY";

/**
 * Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/actions.tsx's `setConnection`.
 * Connects the current tab or creates a new saved query, then sets the tab
 * mode + batch-query context and flips the aside panel to `SCHEMA`.
 */
export function setConnection(options: {
  database?: Database;
  mode?: SQLEditorTabMode;
  newTab: boolean;
  batchQueryContext?: BatchQueryContext;
}) {
  const { database, mode = DEFAULT_TAB_MODE, newTab } = options;
  const connection: SQLEditorConnection = {
    instance: database
      ? extractDatabaseResourceName(database.name).instance
      : "",
    database: database?.name ?? "",
  };

  const tabsState = getSQLEditorTabsState();
  const currentTab = tabsState.tabsById.get(tabsState.currentTabId);
  const { maybeUpdateSavedQuery, createSavedQuery } =
    useSQLEditorStore.getState();

  const batchQueryContext: BatchQueryContext = Object.assign(
    { databases: [] } as BatchQueryContext,
    currentTab?.batchQueryContext,
    options.batchQueryContext
  );

  const createOrUpdate = () => {
    if (!newTab && currentTab) {
      return maybeUpdateSavedQuery({
        tabId: currentTab.id,
        savedQuery: currentTab.savedQuery,
        title: currentTab.title,
        database: connection.database,
        statement: currentTab.statement,
      });
    }
    if (!canCreateSavedQueryInProject(getSQLEditorEditorState().project)) {
      // Same fallback as the tab bar's "+": open the tab locally rather than
      // failing the request, so the connection the user picked still opens.
      return Promise.resolve(getSQLEditorTabsState().addTab({ connection }));
    }
    return createSavedQuery({
      database: connection.database,
    });
  };

  void createOrUpdate().then((tab) => {
    if (tab) {
      getSQLEditorTabsState().updateTab(tab.id, { mode, batchQueryContext });
      useSQLEditorStore.getState().setAsidePanelTab("SCHEMA");
    }
  });
}

/**
 * Replaces `useDropdown` from the Vue `actions.tsx`.
 * Returns the dynamic menu items for a right-clicked tree node. Consumers
 * render the resulting items in their own context menu UI.
 */
export function useConnectionMenu(node: SQLEditorTreeNode | null) {
  const { t } = useTranslation();
  const project = useSQLEditorEditorState((s) => s.project);
  const setShowConnectionPanel = useSQLEditorStore(
    (s) => s.setShowConnectionPanel
  );
  const allowAdmin = useSQLEditorAllowAdmin(project);

  const items = useMemo<ConnectionMenuItem[]>(() => {
    if (!node || node.disabled) return [];

    const { type, target } = node.meta;
    const out: ConnectionMenuItem[] = [];

    if (isConnectableSQLEditorTreeNode(node)) {
      const database = target as Database;
      const instance = instanceOfSQLEditorTreeNode(node);
      if (instance && instanceV1HasReadonlyMode(instance)) {
        out.push({
          key: "connect",
          label: t("database.select"),
          icon: <LinkIcon className="size-4" />,
          onSelect: () => {
            setConnection({ database, newTab: false });
            setShowConnectionPanel(false);
          },
        });
        out.push({
          key: "connect-in-new-tab",
          label: t("sql-editor.open-in-new-tab"),
          icon: <LinkIcon className="size-4" />,
          onSelect: () => {
            setConnection({ database, newTab: true });
            setShowConnectionPanel(false);
          },
        });
      }
      if (allowAdmin) {
        out.push({
          key: "connect-in-admin-mode",
          label: t("sql-editor.connect-in-admin-mode"),
          icon: <Wrench className="size-4" />,
          onSelect: () => {
            setConnection({ database, mode: "ADMIN", newTab: false });
            setShowConnectionPanel(false);
          },
        });
      }
    }

    if (type === "database") {
      const database = target as Database;
      out.push({
        key: "view-database-detail",
        label: t("sql-editor.view-database-detail"),
        icon: <ExternalLink className="size-4" />,
        onSelect: () => {
          const route = router.resolve(autoDatabaseRoute(database));
          window.open(route.href, "_blank");
        },
      });
      if (instanceV1HasAlterSchema(getInstanceResource(database))) {
        out.push({
          key: "alter-schema",
          label: t("database.edit-schema"),
          icon: <SquarePen className="size-4" />,
          onSelect: () => {
            void sqlEditorEvents.emit("alter-schema", {
              databaseName: database.name,
              schema: "",
              table: "",
            });
            setShowConnectionPanel(false);
          },
        });
      }
    }
    return out;
  }, [node, allowAdmin, t, setShowConnectionPanel]);

  const handleSelect = useCallback(
    (key: ActionKey) => {
      items.find((item) => item.key === key)?.onSelect();
    },
    [items]
  );

  return { items, handleSelect };
}
