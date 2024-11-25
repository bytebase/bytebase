import { head } from "lodash-es";
import {
  CodeIcon,
  CopyIcon,
  ExternalLinkIcon,
  FileCodeIcon,
  FileDiffIcon,
  FileMinusIcon,
  FilePlusIcon,
  FileSearch2Icon,
  LinkIcon,
  SquarePenIcon,
} from "lucide-vue-next";
import { NButton, useDialog, type DropdownOption } from "naive-ui";
import { computed, h, nextTick, ref } from "vue";
import { useRouter } from "vue-router";
import TableIcon from "@/components/Icon/TableIcon.vue";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { t } from "@/plugins/i18n";
import { SQL_EDITOR_DATABASE_MODULE } from "@/router/sqlEditor";
import { pushNotification, useAppFeature, useSQLEditorTabStore } from "@/store";
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  dialectOfEngineV1,
  languageOfEngineV1,
  type ComposedDatabase,
  type Position,
  type SQLEditorConnection,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type {
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto/v1/database_service";
import { DataSource, DataSourceType } from "@/types/proto/v1/instance_service";
import {
  defer,
  extractInstanceResourceName,
  extractProjectResourceName,
  generateSimpleDeleteStatement,
  generateSimpleInsertStatement,
  generateSimpleSelectAllStatement,
  generateSimpleUpdateStatement,
  instanceV1HasAlterSchema,
  keyForDependentColumn,
  sortByDictionary,
  toClipboard,
} from "@/utils";
import { keyWithPosition } from "../../EditorCommon";
import {
  useEditorPanelContext,
  type EditorPanelViewState,
} from "../../EditorPanel";
import { useSQLEditorContext } from "../../context";
import type { NodeTarget, NodeType, TreeNode } from "./common";

type DropdownOptionWithTreeNode = DropdownOption & {
  onSelect?: () => void;
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
  _statement: string
): Promise<"CANCEL" | "OVERRIDE" | "COPY"> => {
  const { currentTab } = useSQLEditorTabStore();
  if (!currentTab) {
    return Promise.resolve("CANCEL");
  }
  if (currentTab.statement.trim().length === 0) {
    return Promise.resolve("OVERRIDE");
  }

  const d = defer<"CANCEL" | "OVERRIDE" | "COPY">();
  const dialog = $d.warning({
    title: t("common.warning"),
    content: t("sql-editor.current-editing-statement-is-not-empty"),
    contentClass: "whitespace-pre-wrap",
    style: "z-index: 100000",
    closable: false,
    closeOnEsc: false,
    maskClosable: false,
    action: () => {
      const buttons = [
        h(
          NButton,
          { size: "small", onClick: () => d.resolve("CANCEL") },
          { default: () => t("common.cancel") }
        ),
        h(
          NButton,
          {
            size: "small",
            type: "warning",
            onClick: () => d.resolve("OVERRIDE"),
          },
          { default: () => t("common.override") }
        ),
        h(
          NButton,
          {
            size: "small",
            type: "primary",
            onClick: () => d.resolve("COPY"),
          },
          { default: () => t("common.copy") }
        ),
      ];
      return h(
        "div",
        { class: "flex items-center justify-end gap-2" },
        buttons
      );
    },
  });
  d.promise.then(() => dialog.destroy());
  return d.promise;
};

export const useActions = () => {
  const { updateViewState, typeToView } = useEditorPanelContext();

  const selectAllFromTableOrView = async (node: TreeNode) => {
    const { target } = (node as TreeNode<"table" | "view">).meta;
    if (!targetSupportsGenerateSQL(target)) {
      return;
    }

    const schema = target.schema.name;
    const tableOrViewName = tableOrViewNameForNode(node);
    if (!tableOrViewName) {
      return;
    }

    const { db } = target;
    const { engine } = db.instanceResource;

    const query = await formatCode(
      generateSimpleSelectAllStatement(
        engine,
        schema,
        tableOrViewName,
        SELECT_ALL_LIMIT
      ),
      engine
    );
    updateViewState({
      view: "CODE",
    });
    runQuery(db, schema, tableOrViewName, query);
  };
  const viewDetail = async (node: TreeNode) => {
    const { type, target } = node.meta;
    const SUPPORTED_TYPES: NodeType[] = [
      "schema",
      "expandable-text",
      "table",
      "column",
      "external-table",
      "view",
      "dependent-column",
      "procedure",
      "package",
      "function",
      "sequence",
      "trigger",
      "index",
      "foreign-key",
      "partition-table",
    ] as const;
    if (!SUPPORTED_TYPES.includes(type)) {
      return;
    }
    if (type === "schema") {
      const schema = (node.meta.target as NodeTarget<"schema">).schema.name;
      updateViewState({
        schema,
      });
      return;
    }
    if (type === "expandable-text") {
      const { mockType } = target as NodeTarget<"expandable-text">;
      if (!mockType) return;
      try {
        const view = typeToView(mockType);
        const schema = schemaForNode(node);
        const vs: Partial<EditorPanelViewState> = {
          view,
        };
        if (typeof schema !== "undefined") {
          vs.schema = schema.name;
        }
        if (
          mockType === "column" ||
          mockType === "index" ||
          mockType === "foreign-key" ||
          mockType === "partition-table"
        ) {
          const table = tableForNode(node);
          if (typeof table !== "undefined") {
            vs.detail = { table: table.name };
            if (mockType === "column") {
              vs.detail.column = head(table.columns)?.name;
            }
            if (mockType === "index") {
              vs.detail.index = head(table.indexes)?.name;
            }
            if (mockType === "foreign-key") {
              vs.detail.foreignKey = head(table.foreignKeys)?.name;
            }
            if (mockType === "partition-table") {
              vs.detail.partition = head(table.partitions)?.name;
            }
          }
        }
        if (mockType === "column" || mockType === "dependent-column") {
          const view = viewForNode(node);
          if (typeof view !== "undefined") {
            vs.detail = { view: view.name };
            if (mockType === "column") {
              vs.detail.column = head(view.columns)?.name;
            }
            if (mockType === "dependent-column") {
              const dep = head(view.dependentColumns);
              if (dep) {
                vs.detail.dependentColumn = keyForDependentColumn(dep);
              }
            }
          }
        }
        updateViewState(vs);
      } catch {
        // nothing
      }
      return;
    }

    const { schema } = target as NodeTarget<
      | "table"
      | "column"
      | "view"
      | "procedure"
      | "package"
      | "function"
      | "sequence"
      | "trigger"
      | "external-table"
      | "index"
      | "foreign-key"
      | "partition-table"
      | "dependent-column"
    >;
    updateViewState({
      view: typeToView(type),
      schema: schema.name,
    });
    await nextTick();
    const detail: EditorPanelViewState["detail"] = {};
    if (
      type === "table" ||
      type === "index" ||
      type === "foreign-key" ||
      type === "partition-table"
    ) {
      detail.table = (target as NodeTarget<"table">).table.name;
    }
    if (type === "column") {
      const parentType = node.parent?.parent?.meta.type;
      if (parentType === "table") {
        detail.table = (target as NodeTarget<"table">).table.name;
        detail.column = (target as NodeTarget<"column">).column.name;
      }
      if (parentType === "external-table") {
        detail.externalTable = (
          target as NodeTarget<"external-table">
        ).externalTable.name;
        detail.column = (target as NodeTarget<"column">).column.name;
        updateViewState({
          view: "EXTERNAL_TABLES",
        });
      }
      if (parentType === "view") {
        detail.view = (target as NodeTarget<"view">).view.name;
        detail.column = (target as NodeTarget<"column">).column.name;
        updateViewState({
          view: "VIEWS",
        });
      }
    }
    if (type === "view") {
      detail.view = (target as NodeTarget<"view">).view.name;
    }
    if (type === "dependent-column") {
      const { database, dependentColumn } =
        target as NodeTarget<"dependent-column">;
      const depSchema = database.schemas.find(
        (s) => s.name === dependentColumn.schema
      );
      if (
        depSchema &&
        depSchema.views.find((v) => v.name === dependentColumn.table)
      ) {
        updateViewState({
          view: "VIEWS",
          schema: dependentColumn.schema,
        });
        detail.view = dependentColumn.table;
        detail.column = dependentColumn.column;
      } else {
        updateViewState({
          view: "TABLES",
          schema: dependentColumn.schema,
        });
        detail.table = dependentColumn.table;
        detail.column = dependentColumn.column;
      }
    }
    if (type === "procedure") {
      const { procedure, position } = target as NodeTarget<"procedure">;
      detail.procedure = keyWithPosition(procedure.name, position);
    }
    if (type === "package") {
      const { package: pack, position } = target as NodeTarget<"package">;
      detail.package = keyWithPosition(pack.name, position);
    }
    if (type === "function") {
      const { function: func, position } = target as NodeTarget<"function">;
      detail.func = keyWithPosition(func.name, position);
    }
    if (type === "sequence") {
      const { sequence, position } = target as NodeTarget<"sequence">;
      detail.sequence = keyWithPosition(sequence.name, position);
    }
    if (type === "trigger") {
      const { trigger, position } = target as NodeTarget<"trigger">;
      detail.trigger = keyWithPosition(trigger.name, position);
    }
    if (type === "external-table") {
      detail.externalTable = (
        target as NodeTarget<"external-table">
      ).externalTable.name;
    }
    if (type === "index") {
      detail.index = (target as NodeTarget<"index">).index.name;
    }
    if (type === "foreign-key") {
      detail.foreignKey = (target as NodeTarget<"foreign-key">).foreignKey.name;
    }
    if (type === "partition-table") {
      detail.partition = (
        target as NodeTarget<"partition-table">
      ).partition.name;
    }
    updateViewState({
      detail,
    });
  };

  return { selectAllFromTableOrView, viewDetail };
};

export const useDropdown = () => {
  const router = useRouter();
  const { events: editorEvents, schemaViewer } = useSQLEditorContext();
  const { selectAllFromTableOrView, viewDetail } = useActions();
  const disallowEditSchema = useAppFeature(
    "bb.feature.sql-editor.disallow-edit-schema"
  );
  const disallowNavigateToConsole = useAppFeature(
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

        if (!disallowEditSchema.value && !disallowNavigateToConsole.value) {
          if (instanceV1HasAlterSchema(db.instanceResource)) {
            items.push({
              key: "edit-schema",
              label: t("database.edit-schema"),
              icon: () => <SquarePenIcon class="w-4 h-4" />,
              onSelect: () => {
                editorEvents.emit("alter-schema", {
                  databaseName: db.name,
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
      if (targetSupportsGenerateSQL(target)) {
        items.push({
          key: "preview-table-data",
          label: t("sql-editor.preview-table-data"),
          icon: () => <TableIcon class="w-4 h-4" />,
          onSelect: async () => {
            selectAllFromTableOrView(node);
          },
        });
      }
      const generateSQLChildren: DropdownOptionWithTreeNode[] = [];

      if (targetSupportsGenerateSQL(target)) {
        const engine = engineForTarget(target);
        generateSQLChildren.push({
          key: "generate-sql--select",
          label: "SELECT",
          icon: () => <FileSearch2Icon class="w-4 h-4" />,
          onSelect: async () => {
            const statement = await formatCode(
              generateSimpleSelectAllStatement(
                engine,
                schema,
                tableOrView,
                SELECT_ALL_LIMIT
              ),
              engine
            );
            applyContentToCurrentTabOrCopyToClipboard(statement, $d);
          },
        });
        if (type === "table") {
          const { schema, table } = target as NodeTarget<"table">;
          const columns = table.columns.map((column) => column.name);
          generateSQLChildren.push({
            key: "generate-sql--insert",
            label: "INSERT",
            icon: () => <FilePlusIcon class="w-4 h-4" />,
            onSelect: async () => {
              const statement = await formatCode(
                generateSimpleInsertStatement(
                  engine,
                  schema.name,
                  table.name,
                  columns
                ),
                engine
              );
              applyContentToCurrentTabOrCopyToClipboard(statement, $d);
            },
          });
          generateSQLChildren.push({
            key: "generate-sql--update",
            label: "UPDATE",
            icon: () => <FileDiffIcon class="w-4 h-4" />,
            onSelect: async () => {
              const statement = await formatCode(
                generateSimpleUpdateStatement(
                  engine,
                  schema.name,
                  table.name,
                  columns
                ),
                engine
              );
              applyContentToCurrentTabOrCopyToClipboard(statement, $d);
            },
          });
          generateSQLChildren.push({
            key: "generate-sql--delete",
            label: "DELETE",
            icon: () => <FileMinusIcon class="w-4 h-4" />,
            onSelect: async () => {
              const statement = await formatCode(
                generateSimpleDeleteStatement(engine, schema.name, table.name),
                engine
              );
              applyContentToCurrentTabOrCopyToClipboard(statement, $d);
            },
          });
        }
      }
      if (generateSQLChildren.length > 0) {
        items.push({
          key: "generate-sql",
          label: t("sql-editor.generate-sql"),
          icon: () => <FileCodeIcon class="w-4 h-4" />,
          children: generateSQLChildren,
        });
      }
    }
    if (
      type === "table" ||
      type === "view" ||
      type === "procedure" ||
      type === "function"
    ) {
      items.push({
        key: "view-detail",
        label: t("sql-editor.view-detail"),
        icon: () => <ExternalLinkIcon class="w-4 h-4" />,
        onSelect: () => {
          viewDetail(node);
        },
      });
    }
    const ORDERS = [
      "copy-name",
      "copy-all-column-names",
      "copy-select-statement",
      "preview-table-data",
      "generate-sql",
      "view-schema-text",
      "view-detail",
      "edit-schema",
      "copy-url",
    ];
    sortByDictionary(items, ORDERS, (item) => item.key as string);
    return items;
  });

  const flattenOptions = computed(() => {
    return options.value.flatMap<DropdownOptionWithTreeNode>((item) => {
      if (item.children) {
        return [item, ...item.children] as DropdownOptionWithTreeNode[];
      }
      return item;
    });
  });

  const handleSelect = (key: string) => {
    const option = flattenOptions.value.find((item) => item.key === key);
    if (!option) {
      return;
    }
    if (typeof option.onSelect === "function") {
      option.onSelect();
    }
    show.value = false;
  };

  const handleClickoutside = () => {
    show.value = false;
  };

  return {
    show,
    position,
    context,
    options,
    handleSelect,
    handleClickoutside,
    selectAllFromTableOrView,
  };
};

const schemaForNode = (
  node: TreeNode | undefined
): SchemaMetadata | undefined => {
  if (!node) return undefined;
  if (node.meta.type === "schema") {
    return (node.meta.target as NodeTarget<"schema">).schema;
  }
  return schemaForNode(node.parent);
};

const tableForNode = (
  node: TreeNode | undefined
): TableMetadata | undefined => {
  if (!node) return undefined;
  if (node.meta.type === "table") {
    return (node.meta.target as NodeTarget<"table">).table;
  }
  return tableForNode(node.parent);
};

const viewForNode = (node: TreeNode | undefined): ViewMetadata | undefined => {
  if (!node) return undefined;
  if (node.meta.type === "view") {
    return (node.meta.target as NodeTarget<"view">).view;
  }
  return viewForNode(node.parent);
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

const targetSupportsGenerateSQL = (target: NodeTarget) => {
  const engine = engineForTarget(target);
  if (engine === Engine.REDIS) {
    return false;
  }
  return true;
};

const applyContentToCurrentTabOrCopyToClipboard = async (
  content: string,
  $d: ReturnType<typeof useDialog>
) => {
  const tabStore = useSQLEditorTabStore();
  const tab = tabStore.currentTab;
  if (!tab) {
    copyToClipboard(content);
    return;
  }
  if (tab.statement.trim().length === 0) {
    tabStore.updateCurrentTab({
      statement: content,
    });
    return;
  }
  const choice = await confirmOverrideStatement($d, content);
  if (choice === "CANCEL") {
    return;
  }
  if (choice === "OVERRIDE") {
    tabStore.updateCurrentTab({
      statement: content,
    });
    return;
  }
  copyToClipboard(content);
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
  const tab = useSQLEditorTabStore().currentTab;
  if (!tab) {
    return;
  }
  if (tab.mode === "ADMIN") {
    useSQLEditorTabStore().updateCurrentTab({
      mode: DEFAULT_SQL_EDITOR_TAB_MODE,
    });
  }

  const { execute } = useExecuteSQL();
  const connection: SQLEditorConnection = {
    instance: database.instance,
    database: database.name,
    schema,
    table: tableOrViewName,
    dataSourceId: getDefaultQueryableDataSourceOfDatabase(database).id,
  };
  await nextTick();
  execute({
    statement,
    connection,
    explain: false,
    engine: database.instanceResource.engine,
    selection: tab.editorState.selection,
  });
};

const getDefaultQueryableDataSourceOfDatabase = (
  database: ComposedDatabase
) => {
  const dataSources = database.instanceResource.dataSources;
  const readonlyDataSources = dataSources.filter(
    (ds) => ds.type === DataSourceType.READ_ONLY
  );
  // First try to use readonly data source if available.
  return (head(readonlyDataSources) || head(dataSources)) as DataSource;
};

const formatCode = async (code: string, engine: Engine) => {
  const lang = languageOfEngineV1(engine);
  if (lang !== "sql") {
    return code;
  }
  try {
    const result = await formatSQL(code, dialectOfEngineV1(engine));
    if (!result.error) {
      return result.data;
    }
    return code; // fallback;
  } catch {
    return code; // fallback
  }
};
