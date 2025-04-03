import { head, cloneDeep } from "lodash-es";
import {
  CodeIcon,
  CopyIcon,
  InfoIcon,
  ExternalLinkIcon,
  FileCodeIcon,
  FileDiffIcon,
  FileMinusIcon,
  FilePlusIcon,
  FileSearch2Icon,
  LinkIcon,
  SquarePenIcon,
} from "lucide-vue-next";
import {
  FunctionIcon,
  TableIcon,
  ViewIcon,
  ProcedureIcon,
  ExternalTableIcon,
  PackageIcon,
  SequenceIcon,
} from "@/components/Icon";
import { SchemaDiagramIcon } from "@/components/SchemaDiagram";
import { NButton, useDialog, type DropdownOption } from "naive-ui";
import { computed, h, nextTick, ref } from "vue";
import type { VNodeChild } from "vue";
import { useRouter } from "vue-router";
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
import {
  GetSchemaStringRequest_ObjectType,
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
  sortByDictionary,
  supportGetStringSchema,
  toClipboard,
  defaultSQLEditorTab,
} from "@/utils";
import { keyWithPosition } from "../../EditorCommon";
import {
  type EditorPanelView,
  useEditorPanelContext,
  type EditorPanelViewState,
} from "../../EditorPanel";
import { useSQLEditorContext } from "../../context";
import type { NodeTarget, NodeType, TreeNode } from "./common";

type DropdownOptionWithTreeNode = DropdownOption & {
  onSelect?: () => void;
};
const SELECT_ALL_LIMIT = 50; // default pagesize of SQL Editor

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

  const openNewTab = ({ title, view }: { title?: string; view?: EditorPanelView }) => {
    const tabStore = useSQLEditorTabStore();
    const fromTab = tabStore.currentTab;
    const clonedTab = defaultSQLEditorTab();
    if (fromTab) {
      clonedTab.connection = cloneDeep(fromTab.connection);
      clonedTab.treeState = cloneDeep(fromTab.treeState);
    }
    clonedTab.status = "CLEAN";
    clonedTab.title = title ?? "";
    tabStore.addTab(clonedTab);
    updateViewState({ view });
  }

  const viewDetail = async (node: TreeNode) => {
    const { type, target } = node.meta;
    const SUPPORTED_TYPES: NodeType[] = [
      "schema",
      "table",
      "external-table",
      "view",
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

    openNewTab({
      title: "View detail"
    });

    if (type === "schema") {
      const schema = (node.meta.target as NodeTarget<"schema">).schema.name;
      updateViewState({
        schema,
      });
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
      | "dependency-column"
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
      type === "trigger" ||
      type === "partition-table"
    ) {
      detail.table = (target as NodeTarget<"table">).table.name;
    }
    if (type === "trigger") {
      const { trigger, position } = target as NodeTarget<"trigger">;
      detail.trigger = keyWithPosition(trigger.name, position);
    }
    if (type === "view") {
      detail.view = (target as NodeTarget<"view">).view.name;
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

  return { selectAllFromTableOrView, viewDetail, openNewTab };
};

export const useDropdown = () => {
  const router = useRouter();
  const { events: editorEvents, schemaViewer } = useSQLEditorContext();
  const { selectAllFromTableOrView, viewDetail, openNewTab } = useActions();
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
    if (type === "database" || type === "schema") {
      const actions: { view: EditorPanelView; title: string; icon: () => VNodeChild }[] = [
        {
          view: "INFO",
          title: t("common.info"),
          icon: () => <InfoIcon class="w-4 h-4" />
        },
        {
          view: "TABLES",
          title: t("db.tables"),
          icon: () => <TableIcon class="w-4 h-4" />
        },
        {
          view: "VIEWS",
          title: t("db.views"),
          icon: () => <ViewIcon class="w-4 h-4" />
        },
        {
          view: "FUNCTIONS",
          title: t("db.functions"),
          icon: () => <FunctionIcon class="w-4 h-4" />
        },
        {
          view: "PROCEDURES",
          title: t("db.procedures"),
          icon: () => <ProcedureIcon class="w-4 h-4" />
        },
        {
          view: "SEQUENCES",
          title: t("db.sequences"),
          icon: () => <SequenceIcon class="w-4 h-4" />
        },
        {
          view: "PACKAGES",
          title: t("db.packages"),
          icon: () => <PackageIcon class="w-4 h-4" />
        },
        {
          view: "EXTERNAL_TABLES",
          title: t("db.external-tables"),
          icon: () => <ExternalTableIcon class="w-4 h-4" />
        },
        {
          view: "DIAGRAM",
          title: t("schema-diagram.self"),
          icon: () => <SchemaDiagramIcon class="w-4 h-4" />
        },
      ]
      for (const action of actions) {
        items.push({
          key: action.view,
          label: action.title,
          icon: action.icon,
          onSelect: () => {
            openNewTab({
              title: action.title,
              view: action.view,
            })
          }
        })
      }
    }

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
      if (type === 'view') {
       const {db, schema, view} = target as NodeTarget<"view">;
       if (supportGetStringSchema(db.instanceResource.engine)) {
          items.push({
            key: "view-schema-text",
            label: t("sql-editor.view-schema-text"),
            icon: () => <CodeIcon class="w-4 h-4" />,
            onSelect: () => {
              schemaViewer.value = {
                database: db,
                schema: schema.name,
                object: view.name,
                type: GetSchemaStringRequest_ObjectType.VIEW
              };
            },
          });
        }
      }

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

        if (supportGetStringSchema(db.instanceResource.engine)) {
          items.push({
            key: "view-schema-text",
            label: t("sql-editor.view-schema-text"),
            icon: () => <CodeIcon class="w-4 h-4" />,
            onSelect: () => {
              schemaViewer.value = {
                database: db,
                schema: schema.name,
                object: table.name,
                type: GetSchemaStringRequest_ObjectType.TABLE,
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
