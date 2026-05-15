import { uniq, without } from "lodash-es";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import {
  type CSSProperties,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { TableSchemaViewer } from "@/react/components/TableSchemaViewer";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { Tree, type TreeDataNode } from "@/react/components/ui/tree";
import { countVisibleRows } from "@/react/components/ui/tree-utils";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/react/stores/sqlEditor/tab-vue-state";
import { useDBSchemaV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import type {
  DatabaseMetadata,
  GetSchemaStringRequest_ObjectType,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  getInstanceResource,
  instanceV1Name,
} from "@/utils";
import { findAncestor } from "@/utils/dom";
import { useSchemaPaneActions } from "./actions";
import { useAvailableActions } from "./availableActions";
import { useClickEvents } from "./click";
import { FlatTableList } from "./FlatTableList";
import { HoverPanel } from "./HoverPanel/HoverPanel";
import { HoverStateProvider, useHoverState } from "./hover-state";
import {
  SchemaContextMenu,
  type SchemaContextMenuHandle,
} from "./SchemaContextMenu";
import { SyncSchemaButton } from "./SyncSchemaButton";
import {
  buildDatabaseSchemaTree,
  ExpandableNodeTypes,
  type NodeTarget,
  type TreeNode as SchemaTreeNode,
} from "./schemaTree";
import { Label } from "./TreeNode/Label";

const ROW_HEIGHT = 21;
const TREE_FALLBACK_HEIGHT = 360;
const FLAT_TABLE_THRESHOLD = 1000;
const SEARCH_DEBOUNCE_MS = 200;

/**
 * React port of `frontend/src/views/sql-editor/AsidePanel/SchemaPane/SchemaPane.vue`.
 *
 * Wires together every Phase 1–4 artifact:
 *   - schemaTree.ts builder              (Phase 1)
 *   - useDelayedValue / useHoverState    (Phase 1)
 *   - leaf TreeNode/*.tsx                (Phase 1)
 *   - heavy nodes + Label dispatcher     (Phase 2)
 *   - HoverPanel + *Info cards           (Phase 3)
 *   - useSchemaPaneActions / ContextMenu (Phase 4a/4b)
 *   - SyncSchemaButton                   (Phase 4c)
 *   - FlatTableList                      (Phase 4d)
 *
 * The schema-viewer modal lives in the Vue parent (`AsidePanel.vue`)
 * since it embeds Vue-only `BBModal` + `TableSchemaViewer`. The menu's
 * "View schema text" action emits a `show-schema-viewer` event on
 * `sqlEditorEvents` which the Vue context provider listens for and
 * mirrors onto its inject ref.
 */
export function SchemaPane() {
  return (
    <HoverStateProvider>
      <SchemaPaneInner />
      <HoverPanel offsetX={8} offsetY={0} margin={4} />
    </HoverStateProvider>
  );
}

function SchemaPaneInner() {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const dbSchemaStore = useDBSchemaV1Store();
  const connection = useConnectionOfCurrentSQLEditorTab();
  const hoverState = useHoverState();

  const database = useVueState(() => connection.database.value);
  const currentTabId = useVueState(() => tabStore.currentTab?.id);
  // `connection` is mutated in place via `Object.assign`-style updates,
  // so depth is required to catch nested-field changes (schema, database,
  // instance) without a fresh reference.
  const tabConnection = useVueState(() => tabStore.currentTab?.connection, {
    deep: true,
  });
  const tabConnectionSchema = tabConnection?.schema;
  // tab.viewState shapes the selectedKeys highlight when the editor
  // panel is on a non-CODE view (e.g. TABLES drill-down).
  const panelViewState = useVueState(() => tabStore.currentTab?.viewState, {
    deep: true,
  });

  const [searchPattern, setSearchPattern] = useState("");
  const debouncedSearch = useDebouncedValue(searchPattern, SEARCH_DEBOUNCE_MS);

  // Reset the search box on every tab switch — matches Vue's
  // `watch(() => currentTab.value?.id, ...)`.
  useEffect(() => {
    setSearchPattern("");
  }, [currentTabId]);

  const [metadata, setMetadata] = useState<DatabaseMetadata | null>(null);
  const [isFetching, setIsFetching] = useState(false);

  // Fetch metadata on database change. The store de-duplicates so this
  // is safe to fire on every render where the name flips.
  useEffect(() => {
    if (!isValidDatabaseName(database.name)) {
      setMetadata(null);
      setIsFetching(false);
      return;
    }
    setIsFetching(true);
    let cancelled = false;
    void dbSchemaStore
      .getOrFetchDatabaseMetadata({ database: database.name })
      .then((md) => {
        if (cancelled) return;
        setMetadata(md);
      })
      .finally(() => {
        if (cancelled) return;
        setIsFetching(false);
      });
    return () => {
      cancelled = true;
    };
  }, [database.name, dbSchemaStore]);

  const totalTableCount = useMemo(() => {
    if (!metadata) return 0;
    return metadata.schemas.reduce(
      (sum, schema) => sum + (schema.tables?.length ?? 0),
      0
    );
  }, [metadata]);

  // Build the tree via requestAnimationFrame so the heavy walk doesn't
  // block the metadata-fetch teardown's transition. Mirrors Vue's
  // `requestAnimationFrame(() => { tree.value = buildDatabaseSchemaTree(...) })`.
  const [tree, setTree] = useState<SchemaTreeNode[] | undefined>(undefined);
  useEffect(() => {
    if (isFetching || !metadata) {
      setTree(undefined);
      return;
    }
    if (totalTableCount > FLAT_TABLE_THRESHOLD) {
      setTree(undefined);
      return;
    }
    let raf = 0;
    raf = requestAnimationFrame(() => {
      const built = buildDatabaseSchemaTree(database, metadata);
      setTree(built);
      // First-mount default expand: seed treeState.keys so the user
      // sees database/schema/Tables/Views opened by default. The Vue
      // version seeds when `treeStateDb !== connectionDb && connectionDb`.
      const tab = tabStore.currentTab;
      const connectionDb = tab?.connection.database;
      if (tab && connectionDb && tab.treeState.database !== connectionDb) {
        tab.treeState.database = connectionDb;
        tab.treeState.keys = defaultExpandedKeys(built);
      }
    });
    return () => cancelAnimationFrame(raf);
  }, [isFetching, metadata, totalTableCount, database, tabStore]);

  // Reactive proxy for `tab.treeState.keys`. Writes via `setExpandedKeys`
  // always REPLACE the whole array (`tab.treeState.keys = keys`), so a
  // shallow watch on the getter is enough — the getter's read of
  // `tab.treeState.keys` registers as a dep, and array replacement
  // triggers the watch. Deep traversal over an array of plain strings
  // would just add per-element overhead with no extra reactivity.
  const expandedKeys = useVueState<string[]>(() => {
    const tab = tabStore.currentTab;
    if (!tab) return [];
    if (tab.treeState.database !== database.name) return [];
    return tab.treeState.keys ?? [];
  });

  const setExpandedKeys = useCallback(
    (keys: string[]) => {
      const tab = tabStore.currentTab;
      if (!tab) return;
      if (tab.treeState.database !== database.name) return;
      tab.treeState.keys = keys;
    },
    [tabStore, database.name]
  );

  const upsertExpandedKeys = useCallback(
    (keys: string[]) => {
      setExpandedKeys(uniq([...expandedKeys, ...keys]));
    },
    [expandedKeys, setExpandedKeys]
  );
  const removeExpandedKeys = useCallback(
    (keys: string[]) => {
      setExpandedKeys(without(expandedKeys, ...keys));
    },
    [expandedKeys, setExpandedKeys]
  );

  // ---- Selected-key highlight (current connection / detail view) ---------
  const selectedKeys = useMemo(() => {
    if (!isValidDatabaseName(database.name)) return [];
    const keys: string[] = [];
    if (tabConnectionSchema !== undefined) {
      keys.push(`${database.name}/schemas/${tabConnectionSchema}`);
    }
    if (!panelViewState || panelViewState.view === "CODE") return keys;

    const detail = panelViewState.detail ?? {};
    const parts = [database.name, `schemas/${panelViewState.schema ?? ""}`];
    if (detail.table) {
      parts.push(`tables/${detail.table}`);
      if (detail.column) parts.push(`columns/${detail.column}`);
      else if (detail.index) parts.push(`indexes/${detail.index}`);
      else if (detail.partition)
        parts.push(`partitionTables/${detail.partition}`);
      else if (detail.trigger) parts.push(`triggers/${detail.trigger}`);
      else if (detail.foreignKey)
        parts.push(`foreignKeys/${detail.foreignKey}`);
    } else if (detail.view) {
      parts.push(`views/${detail.view}`);
      if (detail.column) parts.push(`columns/${detail.column}`);
    } else if (detail.procedure) {
      parts.push(`procedures/${detail.procedure}`);
    } else if (detail.func) {
      parts.push(`functions/${detail.func}`);
    } else if (detail.sequence) {
      parts.push(`sequences/${detail.sequence}`);
    } else if (detail.externalTable) {
      parts.push(`externalTables/${detail.externalTable}`);
    } else if (detail.package) {
      parts.push(`packages/${detail.package}`);
    }
    return [parts.join("/")];
  }, [database.name, tabConnectionSchema, panelViewState]);

  // ---- Adaptive flat / tree -----------------------------------------------
  const treeData = useMemo<TreeDataNode<SchemaTreeNode>[]>(() => {
    if (!tree) return [];
    return tree.map(toTreeData);
  }, [tree]);

  const expandedKeySet = useMemo(() => new Set(expandedKeys), [expandedKeys]);
  const searchKeyword = debouncedSearch.trim();
  const visibleRowCount = useMemo(() => {
    if (!tree) return 0;
    return tree.reduce(
      (sum, root) =>
        sum +
        countVisibleRows(root, expandedKeySet, searchKeyword, schemaNodeMatch),
      0
    );
  }, [tree, expandedKeySet, searchKeyword]);

  // ---- Click discriminator (single vs double) -----------------------------
  const { selectAllFromTableOrView } = useSchemaActionsForClick();

  const toggleNode = useCallback(
    (node: SchemaTreeNode) => {
      if (expandedKeys.includes(node.key)) {
        removeExpandedKeys([node.key]);
      } else if (ExpandableNodeTypes.includes(node.meta.type)) {
        upsertExpandedKeys([node.key]);
      }
    },
    [expandedKeys, removeExpandedKeys, upsertExpandedKeys]
  );

  const { handleClick } = useClickEvents({
    onSingleClick: (node) => {
      if (node.meta.type === "schema") {
        const tab = tabStore.currentTab;
        if (tab) {
          tab.connection.schema = (
            node.meta.target as NodeTarget<"schema">
          ).schema;
        }
      }
      toggleNode(node);
    },
    onDoubleClick: (node) => {
      const type = node.meta.type;
      if (type === "table" || type === "view") {
        void selectAllFromTableOrView(node);
      } else {
        // schema / expandable-text / leaf nodes: mirror single-click
        // toggle behavior so the tree state matches what the user
        // expects after a fast double-tap.
        toggleNode(node);
      }
    },
  });

  // ---- Context menu (right-click) -----------------------------------------
  const contextMenuRef = useRef<SchemaContextMenuHandle>(null);
  // Schema-viewer modal lives here now (was bridged through Vue parent
  // until the React TableSchemaViewer landed). Local state because the
  // SchemaPane right-click menu is the only trigger.
  const [schemaViewer, setSchemaViewerState] = useState<
    | {
        schema?: string;
        object?: string;
        type?: GetSchemaStringRequest_ObjectType;
      }
    | undefined
  >(undefined);
  const setSchemaViewer = useCallback(
    (
      viewer:
        | {
            schema?: string;
            object?: string;
            type?: GetSchemaStringRequest_ObjectType;
          }
        | undefined
    ) => {
      setSchemaViewerState(viewer);
    },
    []
  );
  const availableActions = useAvailableActions();

  // ---- Flat list handlers (when totalTableCount > 1000) -------------------
  const onFlatSelect = useCallback(
    (item: { schema?: string }) => {
      const tab = tabStore.currentTab;
      if (tab && item.schema) {
        tab.connection.schema = item.schema;
      }
    },
    [tabStore]
  );
  const onFlatSelectAll = useCallback(
    (item: { key: string; schema: string; metadata: TableMetadata }) => {
      const synthetic: SchemaTreeNode = {
        key: item.key,
        meta: {
          type: "table",
          target: {
            database: database.name,
            schema: item.schema,
            table: item.metadata.name,
          },
        },
      };
      void selectAllFromTableOrView(synthetic);
    },
    [database.name, selectAllFromTableOrView]
  );
  const onFlatContextMenu = useCallback(
    (
      e: React.MouseEvent,
      item: { key: string; schema: string; metadata: TableMetadata }
    ) => {
      const synthetic: SchemaTreeNode = {
        key: item.key,
        meta: {
          type: "table",
          target: {
            database: database.name,
            schema: item.schema,
            table: item.metadata.name,
          },
        },
      };
      contextMenuRef.current?.show(synthetic, e);
    },
    [database.name]
  );

  return (
    <div
      className="gap-y-1 h-full flex flex-col items-stretch relative overflow-hidden"
      // Fail-safe: hide the hover panel when the cursor leaves the
      // SchemaPane container entirely (e.g. moves into the main editor
      // area). Per-row `onMouseLeave` already schedules a hide, but if
      // the cursor exits through whitespace where no row handler fires
      // — or crosses through the floating panel itself which keeps
      // itself alive on hover — the panel can otherwise linger.
      onMouseLeave={() => hoverState.update(undefined, "after")}
    >
      <div className="px-1 flex flex-row gap-1">
        <div className="flex-1 overflow-hidden">
          <Input
            value={searchPattern}
            placeholder={t("common.search")}
            disabled={!tabStore.currentTab}
            onChange={(e) => setSearchPattern(e.target.value)}
            className="h-8 text-sm w-full"
          />
        </div>
        <div className="shrink-0 flex items-center">
          <SyncSchemaButton />
        </div>
      </div>

      <div className="schema-tree flex-1 px-1 pb-1 text-sm overflow-auto select-none">
        {isFetching ? (
          <div className="flex items-center justify-center mt-16 text-control-light">
            <Loader2 className="size-5 animate-spin" />
          </div>
        ) : metadata ? (
          totalTableCount > FLAT_TABLE_THRESHOLD ? (
            <FlatTableList
              metadata={metadata}
              search={debouncedSearch}
              database={database.name}
              onSelect={onFlatSelect}
              onSelectAll={onFlatSelectAll}
              onContextMenu={onFlatContextMenu}
            />
          ) : tree && tree.length > 0 ? (
            <Tree<SchemaTreeNode>
              data={treeData}
              selectedIds={selectedKeys}
              expandedIds={expandedKeys}
              searchTerm={searchKeyword || undefined}
              searchMatch={searchMatch}
              onToggle={(id) => {
                const next = new Set(expandedKeys);
                if (next.has(id)) next.delete(id);
                else next.add(id);
                setExpandedKeys([...next]);
              }}
              height={
                Math.max(visibleRowCount, 1) * ROW_HEIGHT ||
                TREE_FALLBACK_HEIGHT
              }
              rowHeight={ROW_HEIGHT}
              indent={10}
              renderNode={({ node, style }) => (
                <SchemaTreeRow
                  style={style}
                  node={node.data.data}
                  isOpen={!!node.isOpen}
                  hasChildren={
                    !!node.data.data.children &&
                    node.data.data.children.length > 0
                  }
                  keyword={searchKeyword}
                  selected={selectedKeys.includes(node.data.data.key)}
                  hoverUpdate={hoverState.update}
                  hoverHasState={hoverState.state !== undefined}
                  hoverSetPosition={hoverState.setPosition}
                  databaseName={database.name}
                  onClickNode={handleClick}
                  onContextMenu={(n, e) => contextMenuRef.current?.show(n, e)}
                />
              )}
            />
          ) : (
            <EmptyHint label={t("common.empty")} />
          )
        ) : (
          <EmptyHint label={t("common.empty")} />
        )}
      </div>

      <SchemaContextMenu
        ref={contextMenuRef}
        availableActions={availableActions}
        setSchemaViewer={setSchemaViewer}
      />

      {/* Schema-viewer modal — Vue's BBModal+TableSchemaViewer used to
          live in the parent (`SQLEditorHomePage.vue`) bridged via the
          `show-schema-viewer` event because TableSchemaViewer was Vue.
          Now the React TableSchemaViewer ships, so the trigger and its
          modal home in the same place. */}
      <Dialog
        open={!!schemaViewer}
        onOpenChange={(open) => {
          if (!open) setSchemaViewerState(undefined);
        }}
      >
        <DialogContent
          className="p-4"
          style={{
            width: "calc(100vw - 12rem)",
            height: "calc(100vh - 12rem)",
            maxWidth: "calc(100vw - 12rem)",
          }}
        >
          <DialogTitle className="text-base font-medium flex items-center gap-x-1">
            <DatabaseTitle database={database} />
          </DialogTitle>
          {schemaViewer ? (
            <TableSchemaViewer
              database={database}
              schema={schemaViewer.schema}
              object={schemaViewer.object}
              type={schemaViewer.type}
              className="flex-1"
            />
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ---- Helpers / sub-components ----------------------------------------------

function toTreeData(node: SchemaTreeNode): TreeDataNode<SchemaTreeNode> {
  return {
    id: node.key,
    data: node,
    children: node.children?.map(toTreeData),
  };
}

const searchMatch = (
  node: TreeDataNode<SchemaTreeNode>,
  term: string
): boolean => {
  // Mirror Vue's NTree default: case-insensitive substring match against
  // the node's display label.
  const label = node.data.label ?? "";
  return label.toLowerCase().includes(term.toLowerCase());
};

const schemaNodeMatch = (node: SchemaTreeNode, term: string): boolean => {
  const label = node.label ?? "";
  return label.toLowerCase().includes(term.toLowerCase());
};

function defaultExpandedKeys(tree: SchemaTreeNode[]): string[] {
  if (tree.length === 0) return [];
  const keys: string[] = [];
  const collect = (n: SchemaTreeNode) => {
    keys.push(n.key);
    n.children?.forEach(walk);
  };
  const walk = (n: SchemaTreeNode) => {
    const t = n.meta.type;
    if (t === "database" || t === "schema") {
      collect(n);
    } else if (t === "expandable-text") {
      const mockType = (n.meta.target as NodeTarget<"expandable-text">)
        .mockType;
      if (mockType === "table" || mockType === "view") collect(n);
    }
  };
  walk(tree[0]);
  return keys;
}

function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(id);
  }, [value, delayMs]);
  return debounced;
}

/**
 * Direct re-export of `useSchemaPaneActions` so SchemaPaneInner can
 * grab `selectAllFromTableOrView` for the double-click handler. The
 * full menu-action surface lives inside `useSchemaPaneContextMenu`
 * (consumed by SchemaContextMenu).
 */
function useSchemaActionsForClick() {
  return useSchemaPaneActions();
}

/**
 * Minimal database breadcrumb for the schema-viewer modal title:
 * engine icon + instance name + chevron + database name. Replaces
 * Vue's `<RichDatabaseName>` for this single use site — full port
 * of `RichDatabaseName.vue` is deferred until another React surface
 * needs it.
 */
function DatabaseTitle({
  database,
}: {
  database: { name: string; project: string } & Record<string, unknown>;
}) {
  const instance = getInstanceResource(database as never);
  const databaseName = extractDatabaseResourceName(database.name).databaseName;
  return (
    <span className="inline-flex items-center gap-x-1">
      <EngineIcon engine={instance.engine} className="size-4" />
      <span className="truncate">{instanceV1Name(instance)}</span>
      <ChevronRight className="size-3 shrink-0" />
      <span className="truncate">{databaseName}</span>
    </span>
  );
}

function EmptyHint({ label }: { label: string }) {
  return (
    <div className="flex items-center justify-center mt-16 text-control-light text-sm">
      {label}
    </div>
  );
}

function SchemaTreeRow({
  style,
  node,
  isOpen,
  hasChildren,
  keyword,
  selected,
  hoverUpdate,
  hoverHasState,
  hoverSetPosition,
  databaseName,
  onClickNode,
  onContextMenu,
}: {
  style: CSSProperties;
  node: SchemaTreeNode;
  isOpen: boolean;
  hasChildren: boolean;
  keyword: string;
  selected: boolean;
  hoverUpdate: ReturnType<typeof useHoverState>["update"];
  hoverHasState: boolean;
  hoverSetPosition: ReturnType<typeof useHoverState>["setPosition"];
  databaseName: string;
  onClickNode: (node: SchemaTreeNode) => void;
  onContextMenu: (node: SchemaTreeNode, e: React.MouseEvent) => void;
}) {
  const rowRef = useRef<HTMLDivElement | null>(null);

  const handleMouseEnter = (e: React.MouseEvent<HTMLDivElement>) => {
    const type = node.meta.type;
    if (
      type !== "table" &&
      type !== "external-table" &&
      type !== "column" &&
      type !== "view" &&
      type !== "partition-table"
    ) {
      return;
    }
    const target = node.meta.target as
      | NodeTarget<"table">
      | NodeTarget<"external-table">
      | NodeTarget<"column">
      | NodeTarget<"view">
      | NodeTarget<"partition-table">;
    // Build the sparse hover state shape from the current target.
    const state: {
      database: string;
      schema?: string;
      table?: string;
      externalTable?: string;
      view?: string;
      column?: string;
      partition?: string;
    } = { database: databaseName };
    if ("schema" in target) state.schema = target.schema as string;
    if ("table" in target) state.table = target.table as string;
    if ("externalTable" in target)
      state.externalTable = target.externalTable as string;
    if ("view" in target) state.view = target.view as string;
    if ("column" in target) state.column = target.column as string;
    if ("partition" in target) state.partition = target.partition as string;

    const delay = hoverHasState ? 150 : undefined;
    hoverUpdate(state, "before", delay);

    // Compute position from the row's bounding rect, like Vue's
    // `findAncestor(.n-tree-node)` → getBoundingClientRect.
    const wrapper = findAncestor(
      e.target as HTMLElement,
      ".bb-schema-tree-row"
    );
    if (!wrapper) {
      hoverUpdate(undefined, "after", 150);
      return;
    }
    const rect = wrapper.getBoundingClientRect();
    hoverSetPosition({ x: e.clientX, y: rect.bottom });
  };

  const handleMouseLeave = () => {
    hoverUpdate(undefined, "after");
  };

  return (
    <div
      ref={rowRef}
      style={style}
      className={cn(
        "bb-schema-tree-row flex items-center w-full pr-2 cursor-pointer rounded-sm h-full",
        "hover:bg-control-bg/70",
        selected && "bg-accent/10 font-medium",
        node.disabled && "cursor-default opacity-70"
      )}
      data-node-key={node.key}
      data-node-meta-type={node.meta.type}
      // Drive the click discriminator off `mouseup` (gated to the primary
      // button) instead of `onClick`. Safari drops the synthesized `click`
      // when `mousedown` / `mouseup` resolve to different inner elements
      // inside the row (chevron, icon, label). `mouseup` always fires.
      onMouseUp={(e) => {
        if (e.button !== 0) return;
        if (node.disabled) return;
        onClickNode(node);
      }}
      onContextMenu={(e) => {
        e.preventDefault();
        onContextMenu(node, e);
      }}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      <span className="shrink-0 inline-flex size-5 items-center justify-center text-control-light">
        {hasChildren ? (
          isOpen ? (
            <ChevronDown className="size-3.5" />
          ) : (
            <ChevronRight className="size-3.5" />
          )
        ) : null}
      </span>
      <Label node={node} keyword={keyword} />
    </div>
  );
}
