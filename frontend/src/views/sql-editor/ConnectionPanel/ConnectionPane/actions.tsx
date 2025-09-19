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
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  instanceOfSQLEditorTreeNode,
  isConnectableSQLEditorTreeNode,
  type ComposedDatabase,
  type CoreSQLEditorTab,
  type Position,
  type SQLEditorTabMode as TabMode,
  type SQLEditorTreeNode as TreeNode,
} from "@/types";
import {
  emptySQLEditorConnection,
  extractInstanceResourceName,
  extractProjectResourceName,
  hasWorkspacePermissionV2,
  instanceV1HasAlterSchema,
  instanceV1HasReadonlyMode,
  tryConnectToCoreSQLEditorTab,
} from "@/utils";
import { setDefaultDataSourceForConn } from "../../EditorCommon";
import { useSQLEditorContext, type SQLEditorContext } from "../../context";

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
    if (type === "project") {
      return [];
    } else {
      // Don't show any context menu actions for disabled
      // instances/databases
      if (node.disabled) {
        return [];
      }

      const items: DropdownOptionWithTreeNode[] = [];

      if (isConnectableSQLEditorTreeNode(node)) {
        const instance = instanceOfSQLEditorTreeNode(node);
        if (instance && instanceV1HasReadonlyMode(instance)) {
          items.push({
            key: "connect",
            label: t("sql-editor.connect"),
            icon: () => <LinkIcon class="w-4 h-4" />,
            onSelect: () => {
              setConnection(node, {
                context: editorContext,
              });
              showConnectionPanel.value = false;
            },
          });
          items.push({
            key: "connect-in-new-tab",
            label: t("sql-editor.connect-in-new-tab"),
            icon: () => <LinkIcon class="w-4 h-4" />,
            onSelect: () => {
              setConnection(node, {
                extra: { worksheet: "", mode: DEFAULT_SQL_EDITOR_TAB_MODE },
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
              setConnection(node, {
                extra: { worksheet: "", mode: "ADMIN" },
                context: editorContext,
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
    }
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

export const setConnection = (
  node: TreeNode<"database">,
  options: {
    extra?: { worksheet: string; mode: TabMode };
    newTab?: boolean;
    context?: SQLEditorContext;
  } = {}
) => {
  if (!node) {
    return;
  }
  if (!isConnectableSQLEditorTreeNode(node)) {
    // one more guard
    return;
  }
  const {
    extra = {
      worksheet: "",
      mode: DEFAULT_SQL_EDITOR_TAB_MODE,
    },
    newTab = false,
    context,
  } = options;
  const coreTab: CoreSQLEditorTab = {
    connection: emptySQLEditorConnection(),
    ...extra,
  };
  const conn = coreTab.connection;
  const database = node.meta.target;
  conn.instance = database.instance;
  conn.database = database.name;
  setDefaultDataSourceForConn(conn, database);
  tryConnectToCoreSQLEditorTab(coreTab, /* overrideTitle */ true, newTab);

  if (context) {
    context.asidePanelTab.value = "SCHEMA";
  }
};
