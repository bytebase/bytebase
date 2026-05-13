import { cloneDeep } from "lodash-es";
import {
  Code,
  Copy,
  ExternalLink,
  FileCode,
  FileDiff,
  FileMinus,
  FilePlus,
  FileSearch2,
  Link as LinkIcon,
  SquarePen,
  Table as TableIcon,
} from "lucide-react";
import type { ReactNode } from "react";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { formatSQL } from "@/react/components/monaco/sqlFormatter";
import { router } from "@/router";
import { SQL_EDITOR_DATABASE_MODULE } from "@/router/sqlEditor";
import {
  pushNotification,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useSQLEditorTabStore,
} from "@/store";
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  dialectOfEngineV1,
  type EditorPanelView,
  type EditorPanelViewState,
  languageOfEngineV1,
  type SQLEditorConnection,
  type SQLEditorTab,
  typeToView,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  type Database,
  GetSchemaStringRequest_ObjectType,
} from "@/types/proto-es/v1/database_service_pb";
import { defaultViewState } from "@/types/sqlEditor/tabViewState";
import {
  defaultSQLEditorTab,
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
  generateSimpleDeleteStatement,
  generateSimpleInsertStatement,
  generateSimpleSelectAllStatement,
  generateSimpleUpdateStatement,
  getInstanceResource,
  instanceV1HasAlterSchema,
  isSameSQLEditorConnection,
  sortByDictionary,
  supportGetStringSchema,
} from "@/utils";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import {
  type NodeTarget,
  type NodeType,
  readableTextForNodeTarget,
  type TreeNode,
} from "./schemaTree";

const SELECT_ALL_LIMIT = 50;

/**
 * Schema-pane "view detail" panel actions. Mirrors Vue's
 * `useCurrentTabViewStateContext().availableActions` — the consumer
 * (SchemaPane.tsx) computes this from the active connection's instance
 * and passes it in. We don't recompute it here so the per-instance
 * support gating stays a single source of truth.
 */
export type AvailableAction = {
  readonly view: EditorPanelView;
  readonly title: string;
  readonly icon: ReactNode;
};

/**
 * Replaces `SchemaPane/actions.tsx` Vue exports. Two factories:
 *  - `useSchemaPaneActions()` — invokable handlers (selectAll, viewDetail,
 *    openNewTab).
 *  - `useSchemaPaneContextMenu(node, deps)` — nested menu items keyed by
 *    `node.meta.type`, mirroring the Vue dropdown 1:1 (item order +
 *    nesting + i18n keys).
 *
 * The schema-viewer modal's lifecycle stays in SchemaPane.tsx (it owns
 * the React state). The hook receives `setSchemaViewer` so menu items
 * can ask the panel to open the viewer.
 */

/**
 * Pure helper: resolve the engine for a target without standing up a
 * Vue reactive computed.
 */
const engineForDatabase = (database: string): Engine => {
  const db = useDatabaseV1Store().getDatabaseByName(database);
  return getInstanceResource(db).engine;
};

const targetSupportsGenerateSQL = (target: NodeTarget): boolean => {
  const { database } = target as NodeTarget<"database">;
  return engineForDatabase(database) !== Engine.REDIS;
};

const formatCode = async (code: string, engine: Engine): Promise<string> => {
  const lang = languageOfEngineV1(engine);
  if (lang !== "sql") return code;
  try {
    const result = await formatSQL(code, dialectOfEngineV1(engine));
    return result.error ? code : result.data;
  } catch {
    return code;
  }
};

/**
 * Push the generated SQL into the active editor tab. If there is no
 * active tab, fall back to copying to the clipboard so the user doesn't
 * lose the work.
 */
const applyContentToCurrentTabOrCopyToClipboard = async (
  content: string,
  notify: (key: string) => void
) => {
  const tab = useSQLEditorTabStore().currentTab;
  if (!tab) {
    await copyToClipboard(content, notify);
    return;
  }
  void sqlEditorEvents.emit("append-editor-content", {
    content,
    select: true,
  });
};

const copyToClipboard = async (
  content: string,
  notify: (key: string) => void
) => {
  if (typeof navigator === "undefined" || !navigator.clipboard) return;
  try {
    await navigator.clipboard.writeText(content);
    notify("common.copied");
  } catch {
    // Silent fail — matches Vue's `if (!isSupported.value) return` behavior.
  }
};

const updateCurrentTabViewState = (patch: Partial<EditorPanelViewState>) => {
  const tabStore = useSQLEditorTabStore();
  const tab = tabStore.currentTab;
  if (!tab) return;
  tabStore.updateTab(tab.id, {
    viewState: {
      ...defaultViewState(),
      ...tab.viewState,
      ...patch,
    },
  });
};

const runQuery = async (
  database: Database,
  schema: string | undefined,
  tableOrViewName: string,
  statement: string
) => {
  const tabStore = useSQLEditorTabStore();
  const tab = tabStore.currentTab;
  if (!tab) return;
  if (tab.mode === "ADMIN") {
    tabStore.updateCurrentTab({ mode: DEFAULT_SQL_EDITOR_TAB_MODE });
  }
  const { execute } = useExecuteSQL();
  const connection: SQLEditorConnection = {
    instance: extractDatabaseResourceName(database.name).instance,
    database: database.name,
    schema: schema ?? "",
    table: tableOrViewName,
  };
  // Yield once so any state updates above flush before execute reads
  // the tab — mirrors Vue's `await nextTick()` in the original.
  await Promise.resolve();
  execute({
    statement,
    connection,
    explain: false,
    engine: getInstanceResource(database).engine,
    selection: tab.editorState.selection,
  });
};

export function useSchemaPaneActions() {
  const databaseStore = useDatabaseV1Store();
  const tabStore = useSQLEditorTabStore();

  const openNewTab = useCallback(
    (params: {
      title: string;
      schema?: string;
      table?: string;
      view: EditorPanelView;
    }) => {
      const currentViewState = tabStore.currentTab?.viewState;
      const schema = params.schema ?? currentViewState?.schema;
      const table = params.table ?? currentViewState?.table;

      const fromTab = tabStore.currentTab;
      const clonedTab: SQLEditorTab = {
        ...defaultSQLEditorTab(),
        status: "CLEAN",
        title: params.title,
      };
      if (fromTab) {
        clonedTab.connection = cloneDeep(fromTab.connection);
        clonedTab.treeState = cloneDeep(fromTab.treeState);
      }

      const findExistedTab = tabStore.openTabList.find((tab) => {
        if (tab.status !== "CLEAN" || tab.id === fromTab?.id) return false;
        if (!isSameSQLEditorConnection(tab.connection, clonedTab.connection))
          return false;
        const viewState = tab.viewState;
        if (
          viewState.view !== params.view ||
          (schema && viewState.schema !== schema)
        ) {
          return false;
        }
        return true;
      });

      if (findExistedTab) {
        tabStore.setCurrentTabId(findExistedTab.id);
      } else {
        tabStore.addTab(clonedTab);
        updateCurrentTabViewState({ view: params.view, schema, table });
      }
    },
    [tabStore]
  );

  const selectAllFromTableOrView = useCallback(
    async (node: TreeNode) => {
      const { target, type } = (node as TreeNode<"table" | "view">).meta;
      if (!targetSupportsGenerateSQL(target)) return;

      const tableOrViewName = readableTextForNodeTarget(type, target);
      if (!tableOrViewName) return;

      const { database, schema } = target;
      const db = databaseStore.getDatabaseByName(database);
      const engine = getInstanceResource(db).engine;

      const query = await formatCode(
        generateSimpleSelectAllStatement(
          engine,
          schema,
          tableOrViewName,
          SELECT_ALL_LIMIT
        ),
        engine
      );
      updateCurrentTabViewState({ view: "CODE" });
      await runQuery(db, schema, tableOrViewName, query);
    },
    [databaseStore]
  );

  const viewDetail = useCallback(
    async (node: TreeNode) => {
      const { type, target } = node.meta;
      const SUPPORTED: NodeType[] = [
        "table",
        "view",
        "procedure",
        "function",
        "trigger",
      ];
      if (!SUPPORTED.includes(type)) return;

      const schema = (target as NodeTarget<"schema">).schema;
      const table = (target as NodeTarget<"table">).table;
      openNewTab({
        title: "View detail",
        view: typeToView(type),
        schema,
        table,
      });
      await Promise.resolve();

      const detail: EditorPanelViewState["detail"] = {};
      let name = "";
      switch (type) {
        case "table":
          name = (target as NodeTarget<"table">).table;
          detail.table = name;
          break;
        case "view":
          name = (target as NodeTarget<"view">).view;
          detail.view = name;
          break;
        case "procedure": {
          const { procedure, position } = target as NodeTarget<"procedure">;
          name = procedure;
          detail.procedure = keyWithPosition(procedure, position);
          break;
        }
        case "function": {
          const { function: func, position: funcPosition } =
            target as NodeTarget<"function">;
          name = func;
          detail.func = keyWithPosition(func, funcPosition);
          break;
        }
        case "trigger": {
          const { trigger, position: triggerPosition } =
            target as NodeTarget<"trigger">;
          name = trigger;
          detail.trigger = keyWithPosition(trigger, triggerPosition);
          break;
        }
      }
      updateCurrentTabViewState({ detail });
      if (name) {
        tabStore.updateCurrentTab({ title: `Detail for ${type} ${name}` });
      }
    },
    [openNewTab, tabStore]
  );

  return { selectAllFromTableOrView, viewDetail, openNewTab };
}

export type SchemaMenuItem = {
  readonly key: string;
  readonly label: string;
  readonly icon: ReactNode;
  readonly onSelect?: () => void;
  readonly children?: readonly SchemaMenuItem[];
};

export type SchemaMenuDeps = {
  readonly availableActions: readonly AvailableAction[];
  readonly setSchemaViewer: (
    viewer:
      | {
          schema?: string;
          object?: string;
          type?: GetSchemaStringRequest_ObjectType;
        }
      | undefined
  ) => void;
};

const ITEM_ORDER = [
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

/**
 * Mirror Vue's `useDropdown().options` 1:1 — same item ordering, same
 * branching, same keys. Schema-typed nodes use the consumer-supplied
 * `availableActions` list (from `useCurrentTabViewStateContext` on the
 * Vue side) so per-instance support gating lives in one place.
 *
 * Returns `[]` for disabled / unsupported nodes; callers should hide
 * the popup in that case.
 */
export function useSchemaPaneContextMenu(
  node: TreeNode | null,
  deps: SchemaMenuDeps
): SchemaMenuItem[] {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const dbSchema = useDBSchemaV1Store();
  const { selectAllFromTableOrView, viewDetail, openNewTab } =
    useSchemaPaneActions();

  const notify = useCallback(
    (key: string) => {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(key),
      });
    },
    [t]
  );

  return useMemo<SchemaMenuItem[]>(() => {
    if (!node || node.disabled) return [];
    const { type, target } = node.meta;
    const { database, schema } = target as NodeTarget<"schema">;
    if (!database) return [];

    const db = databaseStore.getDatabaseByName(database);
    const items: SchemaMenuItem[] = [];

    if (type === "schema") {
      for (const action of deps.availableActions) {
        items.push({
          key: action.view,
          label: action.title,
          icon: action.icon,
          onSelect: () => {
            openNewTab({
              title: `[${
                extractDatabaseResourceName(db.name).databaseName
              }] ${action.title}`,
              view: action.view,
              schema,
            });
          },
        });
      }
    }

    if (type === "table" || type === "view") {
      const tableOrView = readableTextForNodeTarget(
        type,
        target as NodeTarget<"table" | "view">
      );

      items.push({
        key: "copy-name",
        label: t("sql-editor.copy-name"),
        icon: <Copy className="size-4" />,
        onSelect: () => {
          const name = schema ? `${schema}.${tableOrView}` : tableOrView;
          void copyToClipboard(name, notify);
        },
      });

      if (type === "view") {
        const { view } = target as NodeTarget<"view">;
        if (supportGetStringSchema(getInstanceResource(db).engine)) {
          items.push({
            key: "view-schema-text",
            label: t("sql-editor.view-schema-text"),
            icon: <Code className="size-4" />,
            onSelect: () => {
              deps.setSchemaViewer({
                schema,
                object: view,
                type: GetSchemaStringRequest_ObjectType.VIEW,
              });
            },
          });
        }
      }

      if (type === "table") {
        const { table } = target as NodeTarget<"table">;

        items.push({
          key: "copy-all-column-names",
          label: t("sql-editor.copy-all-column-names"),
          icon: <Copy className="size-4" />,
          onSelect: () => {
            const tableMetadata = dbSchema.getTableMetadata({
              database,
              schema,
              table,
            });
            const name = tableMetadata.columns
              .map((col) => col.name)
              .join(", ");
            void copyToClipboard(name, notify);
          },
        });

        if (supportGetStringSchema(getInstanceResource(db).engine)) {
          items.push({
            key: "view-schema-text",
            label: t("sql-editor.view-schema-text"),
            icon: <Code className="size-4" />,
            onSelect: () => {
              deps.setSchemaViewer({
                schema,
                object: table,
                type: GetSchemaStringRequest_ObjectType.TABLE,
              });
            },
          });
        }

        if (instanceV1HasAlterSchema(getInstanceResource(db))) {
          items.push({
            key: "edit-schema",
            label: t("database.edit-schema"),
            icon: <SquarePen className="size-4" />,
            onSelect: () => {
              void sqlEditorEvents.emit("alter-schema", {
                databaseName: db.name,
                schema: schema,
                table: table,
              });
            },
          });
        }

        items.push({
          key: "copy-url",
          label: t("sql-editor.copy-url"),
          icon: <LinkIcon className="size-4" />,
          onSelect: () => {
            const { instance, databaseName } = extractDatabaseResourceName(
              db.name
            );
            const route = router.resolve({
              name: SQL_EDITOR_DATABASE_MODULE,
              params: {
                project: extractProjectResourceName(db.project),
                instance: extractInstanceResourceName(instance),
                database: databaseName,
              },
              query: { table, schema },
            });
            const url = new URL(route.href, window.location.origin).href;
            void copyToClipboard(url, notify);
          },
        });
      }

      if (targetSupportsGenerateSQL(target)) {
        items.push({
          key: "preview-table-data",
          label: t("sql-editor.preview-table-data"),
          icon: <TableIcon className="size-4" />,
          onSelect: () => {
            void selectAllFromTableOrView(node);
          },
        });
      }

      const generateSQLChildren: SchemaMenuItem[] = [];
      if (targetSupportsGenerateSQL(target)) {
        const engine = engineForDatabase(database);
        generateSQLChildren.push({
          key: "generate-sql--select",
          label: "SELECT",
          icon: <FileSearch2 className="size-4" />,
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
            await applyContentToCurrentTabOrCopyToClipboard(statement, notify);
          },
        });
        if (type === "table") {
          const { table } = target as NodeTarget<"table">;
          const tableMetadata = dbSchema.getTableMetadata({
            database,
            schema,
            table,
          });
          const columns = tableMetadata.columns.map((c) => c.name);

          generateSQLChildren.push({
            key: "generate-sql--insert",
            label: "INSERT",
            icon: <FilePlus className="size-4" />,
            onSelect: async () => {
              const statement = await formatCode(
                generateSimpleInsertStatement(engine, schema, table, columns),
                engine
              );
              await applyContentToCurrentTabOrCopyToClipboard(
                statement,
                notify
              );
            },
          });
          generateSQLChildren.push({
            key: "generate-sql--update",
            label: "UPDATE",
            icon: <FileDiff className="size-4" />,
            onSelect: async () => {
              const statement = await formatCode(
                generateSimpleUpdateStatement(engine, schema, table, columns),
                engine
              );
              await applyContentToCurrentTabOrCopyToClipboard(
                statement,
                notify
              );
            },
          });
          generateSQLChildren.push({
            key: "generate-sql--delete",
            label: "DELETE",
            icon: <FileMinus className="size-4" />,
            onSelect: async () => {
              const statement = await formatCode(
                generateSimpleDeleteStatement(engine, schema, table),
                engine
              );
              await applyContentToCurrentTabOrCopyToClipboard(
                statement,
                notify
              );
            },
          });
        }
      }
      if (generateSQLChildren.length > 0) {
        items.push({
          key: "generate-sql",
          label: t("sql-editor.generate-sql"),
          icon: <FileCode className="size-4" />,
          children: generateSQLChildren,
        });
      }
    }

    if (
      type === "table" ||
      type === "view" ||
      type === "procedure" ||
      type === "function" ||
      type === "trigger"
    ) {
      items.push({
        key: "view-detail",
        label: t("sql-editor.view-detail"),
        icon: <ExternalLink className="size-4" />,
        onSelect: () => {
          void viewDetail(node);
        },
      });
    }

    sortByDictionary(items, ITEM_ORDER, (item) => item.key);
    return items;
  }, [
    node,
    deps,
    databaseStore,
    dbSchema,
    notify,
    openNewTab,
    selectAllFromTableOrView,
    t,
    viewDetail,
  ]);
}
