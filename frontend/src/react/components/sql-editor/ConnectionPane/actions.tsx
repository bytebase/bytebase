import {
  ExternalLink,
  Link as LinkIcon,
  SquarePen,
  Wrench,
} from "lucide-react";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DATABASE_DETAIL } from "@/router/dashboard/projectV1";
import {
  useSQLEditorStore as useSQLEditorPiniaStore,
  useSQLEditorTabStore,
  useSQLEditorWorksheetStore,
} from "@/store";
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
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
  getInstanceResource,
  instanceV1HasAlterSchema,
  instanceV1HasReadonlyMode,
} from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";

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

const DEFAULT_TAB_MODE: SQLEditorTabMode = "WORKSHEET";

/**
 * Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/actions.tsx's `setConnection`.
 * Connects the current tab or creates a new worksheet, then sets the tab
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

  const tabStore = useSQLEditorTabStore();
  const worksheetStore = useSQLEditorWorksheetStore();

  const batchQueryContext: BatchQueryContext = Object.assign(
    { databases: [] } as BatchQueryContext,
    tabStore.currentTab?.batchQueryContext,
    options.batchQueryContext
  );

  const createOrUpdate = () => {
    if (!newTab && tabStore.currentTab) {
      return worksheetStore.maybeUpdateWorksheet({
        tabId: tabStore.currentTab.id,
        worksheet: tabStore.currentTab.worksheet,
        title: tabStore.currentTab.title,
        database: connection.database,
        statement: tabStore.currentTab.statement,
      });
    }
    return worksheetStore.createWorksheet({
      database: connection.database,
    });
  };

  void createOrUpdate().then((tab) => {
    if (tab) {
      tabStore.updateTab(tab.id, { mode, batchQueryContext });
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
  const editorStore = useSQLEditorPiniaStore();
  const setShowConnectionPanel = useSQLEditorStore(
    (s) => s.setShowConnectionPanel
  );
  const allowAdmin = useVueState(() => editorStore.allowAdmin);

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
      const { instance, databaseName } = extractDatabaseResourceName(
        database.name
      );
      out.push({
        key: "view-database-detail",
        label: t("sql-editor.view-database-detail"),
        icon: <ExternalLink className="size-4" />,
        onSelect: () => {
          const route = router.resolve({
            name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
            params: {
              projectId: extractProjectResourceName(database.project),
              instanceId: extractInstanceResourceName(instance),
              databaseName,
            },
          });
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
