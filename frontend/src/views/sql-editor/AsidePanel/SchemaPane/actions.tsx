import { head } from "lodash-es";
import {
  CodeIcon,
  CopyIcon,
  ExternalLinkIcon,
  LinkIcon,
  SquarePenIcon,
} from "lucide-vue-next";
import { useDialog, type DropdownOption } from "naive-ui";
import { computed, nextTick, ref } from "vue";
import { useRouter } from "vue-router";
import TableIcon from "@/components/Icon/TableIcon.vue";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { t } from "@/plugins/i18n";
import { PROJECT_V1_ROUTE_DATABASE_DETAIL } from "@/router/dashboard/projectV1";
import { SQL_EDITOR_DATABASE_MODULE } from "@/router/sqlEditor";
import { pushNotification, useAppFeature, useSQLEditorTabStore } from "@/store";
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  type ComposedDatabase,
  type CoreSQLEditorTab,
  type Position,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { DataSource, DataSourceType } from "@/types/proto/v1/instance_service";
import {
  defer,
  extractInstanceResourceName,
  extractProjectResourceName,
  generateSimpleSelectAllStatement,
  instanceV1HasAlterSchema,
  sortByDictionary,
  suggestedTabTitleForSQLEditorConnection,
  toClipboard,
} from "@/utils";
import { useSQLEditorContext } from "../../context";
import type { NodeTarget, TreeNode } from "./common";

type DropdownOptionWithTreeNode = DropdownOption & {
  onSelect: () => void;
};
const SELECT_ALL_LIMIT = 50; // default pagesize of SQL Editor
const VIEW_SCHEMA_ACTION_ENABLED_ENGINES = [
  Engine.MYSQL,
  Engine.OCEANBASE,
  Engine.POSTGRES,
  Engine.TIDB,
];

const confirmOverrideStatement = async (
  $d: ReturnType<typeof useDialog>,
  statement: string
) => {
  const { currentTab } = useSQLEditorTabStore();
  if (currentTab && currentTab.statement.trim().length > 0) {
    const d = defer<boolean>();

    $d.warning({
      title: t("common.warning"),
      content: t("sql-editor.will-override-current-editing-statement"),
      contentClass: "whitespace-pre-wrap",
      style: "z-index: 100000",
      negativeText: t("common.cancel"),
      positiveText: t("common.confirm"),
      onNegativeClick: () => {
        d.resolve(false);
      },
      onPositiveClick: () => {
        d.resolve(true);
      },
    });
    return d.promise;
  }

  return Promise.resolve(true);
};

export const useDropdown = () => {
  const router = useRouter();
  const { events: editorEvents, schemaViewer } = useSQLEditorContext();
  const disallowNavigateAwaySQLEditor = useAppFeature(
    "bb.feature.disallow-navigate-to-console"
  );
  const $d = useDialog();

  const show = ref(false);
  const position = ref<Position>({
    x: 0,
    y: 0,
  });
  const context = ref<TreeNode>();

  const options = computed((): DropdownOptionWithTreeNode[] => {
    const node = context.value;
    if (!node) {
      return [];
    }
    const { type, target } = node.meta;

    // Don't show any context menu actions for disabled nodes
    if (node.disabled) {
      return [];
    }

    const items: DropdownOptionWithTreeNode[] = [];
    if (type === "table" || type === "view") {
      const schema = (target as NodeTarget<"table" | "view">).schema.name;
      const tableOrView = tableOrViewNameForNode(node);
      items.push({
        key: "copy-name",
        label: t("sql-editor.copy-name"),
        icon: () => <CopyIcon class="w-4 h-4" />,
        onSelect: () => {
          const name = schema ? `${schema}.${tableOrView}` : tableOrView;
          copyToClipboard(name);
        },
      });
      if (type === "table") {
        const { db, schema, table } = target as NodeTarget<"table">;

        items.push({
          key: "copy-all-column-names",
          label: t("sql-editor.copy-all-column-names"),
          icon: () => <CopyIcon class="w-4 h-4" />,
          onSelect: () => {
            const names = table.columns.map((col) => col.name).join(", ");
            copyToClipboard(names);
          },
        });

        if (
          VIEW_SCHEMA_ACTION_ENABLED_ENGINES.includes(
            db.instanceResource.engine
          )
        ) {
          items.push({
            key: "view-schema-text",
            label: t("sql-editor.view-schema-text"),
            icon: () => <CodeIcon class="w-4 h-4" />,
            onSelect: () => {
              schemaViewer.value = {
                database: db,
                schema: schema.name,
                table: table.name,
              };
            },
          });
        }

        if (!disallowNavigateAwaySQLEditor.value) {
          items.push({
            key: "view-table-detail",
            label: t("sql-editor.view-table-detail"),
            icon: () => <ExternalLinkIcon class="w-4 h-4" />,
            onSelect: () => {
              const route = router.resolve({
                name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
                params: {
                  projectId: extractProjectResourceName(db.project),
                  instanceId: extractInstanceResourceName(db.instance),
                  databaseName: db.databaseName,
                },
                query: {
                  schema: schema.name ? schema.name : undefined,
                  table: table.name,
                },
              });
              const url = route.href;
              window.open(url, "_blank");
            },
          });

          if (instanceV1HasAlterSchema(db.instanceResource)) {
            items.push({
              key: "edit-schema",
              label: t("database.edit-schema"),
              icon: () => <SquarePenIcon class="w-4 h-4" />,
              onSelect: () => {
                editorEvents.emit("alter-schema", {
                  databaseUID: db.uid,
                  schema: schema.name,
                  table: table.name,
                });
              },
            });
          }

          items.push({
            key: "copy-url",
            label: t("sql-editor.copy-url"),
            icon: () => <LinkIcon class="w-4 h-4" />,
            onSelect: () => {
              const route = router.resolve({
                name: SQL_EDITOR_DATABASE_MODULE,
                params: {
                  project: extractProjectResourceName(db.project),
                  instance: extractInstanceResourceName(db.instance),
                  database: db.databaseName,
                },
                query: {
                  table: table.name,
                  schema: schema.name,
                },
              });
              const url = new URL(route.href, window.location.origin).href;
              copyToClipboard(url);
            },
          });
        }
      }
      if (targetSupportsSelectAll(target)) {
        items.push({
          key: "copy-select-statement",
          label: t("sql-editor.copy-select-statement"),
          icon: () => <CopyIcon class="w-4 h-4" />,
          onSelect: () => {
            const statement = generateSimpleSelectAllStatement(
              engineForTarget(target),
              schema,
              tableOrView,
              SELECT_ALL_LIMIT
            );
            copyToClipboard(statement);
          },
        });
        items.push({
          key: "preview-table-data",
          label: t("sql-editor.preview-table-data"),
          icon: () => <TableIcon class="w-4 h-4" />,
          onSelect: async () => {
            const statement = generateSimpleSelectAllStatement(
              engineForTarget(target),
              schema,
              tableOrView,
              SELECT_ALL_LIMIT
            );
            const confirmed = await confirmOverrideStatement($d, statement);
            if (!confirmed) {
              return;
            }
            runQuery(
              (target as NodeTarget<"database">).db,
              schema,
              tableOrView,
              statement
            );
          },
        });
      }
    }
    const ORDERS = [
      "copy-name",
      "copy-all-column-names",
      "copy-select-statement",
      "preview-table-data",
      "view-schema-text",
      "view-table-detail",
      "edit-schema",
      "copy-url",
    ];
    sortByDictionary(items, ORDERS, (item) => item.key as string);
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

const tableOrViewNameForNode = (node: TreeNode) => {
  const { type, target } = node.meta;
  return type === "table"
    ? (target as NodeTarget<"table">).table.name
    : type === "view"
      ? (target as NodeTarget<"view">).view.name
      : "";
};

const engineForTarget = (target: NodeTarget) => {
  return (target as NodeTarget<"database">).db.instanceResource.engine;
};

const targetSupportsSelectAll = (target: NodeTarget) => {
  const engine = engineForTarget(target);
  if (engine === Engine.REDIS) {
    return false;
  }
  return true;
};

const copyToClipboard = (content: string) => {
  toClipboard(content).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  });
};

const runQuery = async (
  database: ComposedDatabase,
  schema: string,
  tableOrViewName: string,
  statement: string
) => {
  const tabStore = useSQLEditorTabStore();
  const { execute } = useExecuteSQL();
  const tab: CoreSQLEditorTab = {
    connection: {
      instance: database.instance,
      database: database.name,
      schema,
      table: tableOrViewName,
      dataSourceId: getDefaultQueriableDataSourceOfDatabase(database).id,
    },
    mode: DEFAULT_SQL_EDITOR_TAB_MODE,
    worksheet: "",
  };
  if (
    tabStore.currentTab &&
    (tabStore.currentTab.status === "NEW" || !tabStore.currentTab.worksheet)
  ) {
    // If the current tab is "fresh new" or unsaved, update its connection directly.
    tabStore.updateCurrentTab({
      ...tab,
      title: suggestedTabTitleForSQLEditorConnection(tab.connection),
      status: "DIRTY",
      statement,
    });
  } else {
    // Otherwise select or add a new tab and set its connection
    tabStore.addTab(
      {
        ...tab,
        title: suggestedTabTitleForSQLEditorConnection(tab.connection),
        statement,
        status: "DIRTY",
      },
      /* beside */ true
    );
  }
  await nextTick();
  execute({
    statement,
    connection: { ...tab.connection },
    explain: false,
    engine: database.instanceResource.engine,
  });
};

export const selectAllFromTableOrView = async (
  $d: ReturnType<typeof useDialog>,
  node: TreeNode
) => {
  const { target } = (node as TreeNode<"table" | "view">).meta;
  if (!targetSupportsSelectAll(target)) {
    return;
  }

  const schema = target.schema.name;
  const tableOrViewName = tableOrViewNameForNode(node);
  if (!tableOrViewName) {
    return;
  }

  const { db } = target;
  const { engine } = db.instanceResource;

  const query = generateSimpleSelectAllStatement(
    engine,
    schema,
    tableOrViewName,
    SELECT_ALL_LIMIT
  );
  const confirmed = await confirmOverrideStatement($d, query);
  if (!confirmed) {
    return;
  }
  runQuery(db, schema, tableOrViewName, query);
};

const getDefaultQueriableDataSourceOfDatabase = (
  database: ComposedDatabase
) => {
  const dataSources = database.instanceResource.dataSources;
  const readonlyDataSources = dataSources.filter(
    (ds) => ds.type === DataSourceType.READ_ONLY
  );
  // First try to use readonly data source if available.
  return (head(readonlyDataSources) || head(dataSources)) as DataSource;
};
