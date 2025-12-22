import {
  ExternalLinkIcon,
  LinkIcon,
  SquarePenIcon,
  WrenchIcon,
} from "lucide-vue-next";
import type { DropdownOption } from "naive-ui";
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { t } from "@/plugins/i18n";
import { PROJECT_V1_ROUTE_DATABASE_DETAIL } from "@/router/dashboard/projectV1";
import { useSQLEditorTabStore } from "@/store";
import {
  type BatchQueryContext,
  type ComposedDatabase,
  DEFAULT_SQL_EDITOR_TAB_MODE,
  instanceOfSQLEditorTreeNode,
  isConnectableSQLEditorTreeNode,
  type Position,
  type SQLEditorConnection,
  type SQLEditorTabMode as TabMode,
  type SQLEditorTreeNode as TreeNode,
} from "@/types";
import {
  extractInstanceResourceName,
  extractProjectResourceName,
  hasWorkspacePermissionV2,
  instanceV1HasAlterSchema,
  instanceV1HasReadonlyMode,
  setDefaultDataSourceForConn,
} from "@/utils";
import { type SQLEditorContext, useSQLEditorContext } from "../../context";

type DropdownOptionWithTreeNode = DropdownOption & {
  onSelect: () => void;
};

export const useDropdown = () => {
  const router = useRouter();
  const editorContext = useSQLEditorContext();
  const { events: editorEvents, showConnectionPanel } = editorContext;

  const show = ref(false);
  const position = ref<Position>({
    x: 0,
    y: 0,
  });
  const context = ref<TreeNode>();

  const allowAdmin = computed(() => hasWorkspacePermissionV2("bb.sql.admin"));

  const options = computed((): DropdownOptionWithTreeNode[] => {
    const node = context.value;
    if (!node) {
      return [];
    }
    const { type, target } = node.meta;
    // Don't show any context menu actions for disabled
    // instances/databases
    if (node.disabled) {
      return [];
    }

    const items: DropdownOptionWithTreeNode[] = [];

    if (isConnectableSQLEditorTreeNode(node)) {
      const database = node.meta.target as ComposedDatabase;
      const instance = instanceOfSQLEditorTreeNode(node);
      if (instance && instanceV1HasReadonlyMode(instance)) {
        items.push({
          key: "connect",
          label: t("sql-editor.connect"),
          icon: () => <LinkIcon class="w-4 h-4" />,
          onSelect: () => {
            setConnection({
              database,
              context: editorContext,
              newTab: false,
            });
            showConnectionPanel.value = false;
          },
        });
        items.push({
          key: "connect-in-new-tab",
          label: t("sql-editor.connect-in-new-tab"),
          icon: () => <LinkIcon class="w-4 h-4" />,
          onSelect: () => {
            setConnection({
              database,
              newTab: true,
              context: editorContext,
            });
            showConnectionPanel.value = false;
          },
        });
      }
      if (allowAdmin.value) {
        items.push({
          key: "connect-in-admin-mode",
          label: t("sql-editor.connect-in-admin-mode"),
          icon: () => <WrenchIcon class="w-4 h-4" />,
          onSelect: () => {
            setConnection({
              database,
              mode: "ADMIN",
              context: editorContext,
              newTab: false,
            });
            showConnectionPanel.value = false;
          },
        });
      }
    }
    if (type === "database") {
      const database = target as ComposedDatabase;
      items.push({
        key: "view-database-detail",
        label: t("sql-editor.view-database-detail"),
        icon: () => <ExternalLinkIcon class="w-4 h-4" />,
        onSelect: () => {
          const route = router.resolve({
            name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
            params: {
              projectId: extractProjectResourceName(database.project),
              instanceId: extractInstanceResourceName(database.instance),
              databaseName: database.databaseName,
            },
          });
          const url = route.href;
          window.open(url, "_blank");
        },
      });
      if (instanceV1HasAlterSchema(database.instanceResource)) {
        items.push({
          key: "alter-schema",
          label: t("database.edit-schema"),
          icon: () => <SquarePenIcon class="w-4 h-4" />,
          onSelect: () => {
            const db = node.meta.target as ComposedDatabase;
            editorEvents.emit("alter-schema", {
              databaseName: db.name,
              schema: "",
              table: "",
            });
            showConnectionPanel.value = false;
          },
        });
      }
    }
    return items;
  });

  const handleSelect = (key: string) => {
    const option = options.value.find((item) => item.key === key);
    if (!option) {
      return;
    }
    option.onSelect();
    show.value = false;
  };

  const handleClickoutside = () => {
    show.value = false;
  };

  return { show, position, context, options, handleSelect, handleClickoutside };
};

// setConnection will:
// - when newTab == false && exist current tab: connect to the current tab
// - otherwise create a new worksheet then open the tab
export const setConnection = (options: {
  database?: ComposedDatabase;
  mode?: TabMode;
  newTab: boolean;
  context: SQLEditorContext;
  batchQueryContext?: BatchQueryContext;
}) => {
  const {
    database,
    mode = DEFAULT_SQL_EDITOR_TAB_MODE,
    newTab = false,
    context,
  } = options;
  const connection: SQLEditorConnection = {
    instance: database?.instance ?? "",
    database: database?.name ?? "",
  };
  if (database) {
    setDefaultDataSourceForConn(connection, database);
  }

  const tabStore = useSQLEditorTabStore();
  const batchQueryContext: BatchQueryContext = Object.assign(
    { databases: [] },
    tabStore.currentTab?.batchQueryContext,
    options.batchQueryContext
  );

  const createOrUpdate = () => {
    if (!newTab && tabStore.currentTab) {
      return context.maybeUpdateWorksheet({
        tabId: tabStore.currentTab.id,
        worksheet: tabStore.currentTab.worksheet,
        title: tabStore.currentTab.title,
        database: connection.database,
        statement: tabStore.currentTab.statement,
      });
    }

    // create new worksheet and set connection
    return context.createWorksheet({
      database: connection.database,
    });
  };

  createOrUpdate().then((tab) => {
    if (tab) {
      tabStore.updateTab(tab.id, { mode, batchQueryContext });
      context.asidePanelTab.value = "SCHEMA";
    }
  });
};
