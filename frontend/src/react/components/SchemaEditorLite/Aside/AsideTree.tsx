import { create } from "@bufbuild/protobuf";
import { debounce } from "lodash-es";
import {
  ChevronDown,
  ChevronRight,
  Database as DatabaseIcon,
  FileCode,
  FunctionSquare,
  Layers,
  Table2,
  View,
} from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";
import type { NodeRendererProps } from "react-arborist";
import { Tree } from "react-arborist";
import { createPortal } from "react-dom";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import { SearchInput } from "@/react/components/ui/search-input";
import { cn } from "@/react/lib/utils";
import type {
  FunctionMetadata,
  ProcedureMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  FunctionMetadataSchema,
  ProcedureMetadataSchema,
  ViewMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";
import { SchemaNameDialog } from "../Modals/SchemaNameDialog";
import { TableNameDialog } from "../Modals/TableNameDialog";
import type { EditStatus } from "../types";
import { NodeCheckbox } from "./NodeCheckbox";
import type { TreeNode, TreeNodeForTable } from "./tree-builder";
import { buildTree } from "./tree-builder";
import { useContextMenu } from "./useContextMenu";

export function AsideTree() {
  const context = useSchemaEditorContext();
  const {
    targets,
    readonly,
    tabs,
    editStatus,
    selection,
    scrollStatus,
    rebuildTree,
    treeBuildVersion,
  } = context;

  const [searchPattern, setSearchPattern] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);

  // Tree data — treeBuildVersion busts the cache after in-place metadata mutations
  const { tree, nodeMap } = useMemo(
    () => buildTree(targets, { byInstance: targets.length > 1 }),
    [targets, treeBuildVersion]
  );

  const arboristData = useMemo(() => convertToArboristData(tree), [tree]);

  const handleSearchChange = useMemo(
    () => debounce((value: string) => setSearchPattern(value), 200),
    []
  );

  // Context menu
  const { menuState, menuOptions, showMenu, hideMenu } =
    useContextMenu(editStatus);

  // Modal state
  const [tableNameModalCtx, setTableNameModalCtx] = useState<{
    db: ReturnType<typeof useSchemaEditorContext>["targets"][0]["database"];
    database: ReturnType<
      typeof useSchemaEditorContext
    >["targets"][0]["metadata"];
    schema: ReturnType<
      typeof useSchemaEditorContext
    >["targets"][0]["metadata"]["schemas"][0];
    table?: TreeNodeForTable["metadata"]["table"];
  } | null>(null);

  const [schemaNameModalCtx, setSchemaNameModalCtx] = useState<{
    db: ReturnType<typeof useSchemaEditorContext>["targets"][0]["database"];
    database: ReturnType<
      typeof useSchemaEditorContext
    >["targets"][0]["metadata"];
  } | null>(null);

  // Handle context menu selection
  const handleMenuSelect = useCallback(
    (key: string) => {
      const node = menuState.node;
      if (!node) return;
      hideMenu();

      if (key === "create-schema" && node.type === "database") {
        setSchemaNameModalCtx({
          db: node.db,
          database: node.metadata.database,
        });
      } else if (
        key === "create-table" &&
        node.type === "group" &&
        node.group === "table"
      ) {
        setTableNameModalCtx({
          db: node.db,
          database: node.metadata.database,
          schema: node.metadata.schema,
        });
      } else if (key === "rename-table" && node.type === "table") {
        setTableNameModalCtx({
          db: node.db,
          database: node.metadata.database,
          schema: node.metadata.schema,
          table: node.metadata.table,
        });
      } else if (key === "drop-table" && node.type === "table") {
        const status = editStatus.getTableStatus(node.db, node.metadata);
        if (status === "created") {
          const idx = node.metadata.schema.tables.indexOf(node.metadata.table);
          if (idx >= 0) node.metadata.schema.tables.splice(idx, 1);
          editStatus.removeEditStatus(
            node.db,
            { schema: node.metadata.schema, table: node.metadata.table },
            true
          );
        } else {
          editStatus.markEditStatus(
            node.db,
            { schema: node.metadata.schema, table: node.metadata.table },
            "dropped"
          );
        }
        rebuildTree(false);
      } else if (key === "restore-table" && node.type === "table") {
        editStatus.removeEditStatus(
          node.db,
          { schema: node.metadata.schema, table: node.metadata.table },
          false
        );
        rebuildTree(false);
      } else if (key === "drop-schema" && node.type === "schema") {
        editStatus.markEditStatus(
          node.db,
          { schema: node.metadata.schema },
          "dropped"
        );
        rebuildTree(false);
      } else if (key === "restore-schema" && node.type === "schema") {
        editStatus.removeEditStatus(
          node.db,
          { schema: node.metadata.schema },
          false
        );
        rebuildTree(false);
      } else if (
        key === "create-view" &&
        node.type === "group" &&
        node.group === "view"
      ) {
        const view = create(ViewMetadataSchema, {
          name: "new_view",
          definition: "",
        }) as ViewMetadata;
        node.metadata.schema.views.push(view);
        editStatus.markEditStatus(
          node.db,
          { schema: node.metadata.schema, view },
          "created"
        );
        tabs.addTab({
          type: "view",
          database: node.db,
          metadata: {
            database: node.metadata.database,
            schema: node.metadata.schema,
            view,
          },
        });
        rebuildTree(false);
      } else if (
        key === "create-procedure" &&
        node.type === "group" &&
        node.group === "procedure"
      ) {
        const procedure = create(ProcedureMetadataSchema, {
          name: "new_procedure",
          definition: "",
        }) as ProcedureMetadata;
        node.metadata.schema.procedures.push(procedure);
        editStatus.markEditStatus(
          node.db,
          { schema: node.metadata.schema, procedure },
          "created"
        );
        tabs.addTab({
          type: "procedure",
          database: node.db,
          metadata: {
            database: node.metadata.database,
            schema: node.metadata.schema,
            procedure,
          },
        });
        rebuildTree(false);
      } else if (
        key === "create-function" &&
        node.type === "group" &&
        node.group === "function"
      ) {
        const func = create(FunctionMetadataSchema, {
          name: "new_function",
          definition: "",
        }) as FunctionMetadata;
        node.metadata.schema.functions.push(func);
        editStatus.markEditStatus(
          node.db,
          { schema: node.metadata.schema, function: func },
          "created"
        );
        tabs.addTab({
          type: "function",
          database: node.db,
          metadata: {
            database: node.metadata.database,
            schema: node.metadata.schema,
            function: func,
          },
        });
        rebuildTree(false);
      } else if (
        key.startsWith("drop-") &&
        (node.type === "view" ||
          node.type === "procedure" ||
          node.type === "function")
      ) {
        editStatus.markEditStatus(
          node.db,
          { schema: node.metadata.schema, ...getResourceKey(node) },
          "dropped"
        );
        rebuildTree(false);
      } else if (
        key.startsWith("restore-") &&
        (node.type === "view" ||
          node.type === "procedure" ||
          node.type === "function")
      ) {
        editStatus.removeEditStatus(
          node.db,
          { schema: node.metadata.schema, ...getResourceKey(node) },
          false
        );
        rebuildTree(false);
      }
    },
    [menuState.node, hideMenu, editStatus, rebuildTree, tabs]
  );

  // Node click handler
  const handleNodeClick = useCallback(
    (node: TreeNode) => {
      if (node.type === "database") {
        tabs.addTab({
          type: "database",
          database: node.db,
          metadata: node.metadata,
        });
      } else if (node.type === "schema") {
        tabs.addTab({
          type: "database",
          database: node.db,
          metadata: { database: node.metadata.database },
          selectedSchema: node.metadata.schema.name,
        });
      } else if (node.type === "table") {
        tabs.addTab({
          type: "table",
          database: node.db,
          metadata: node.metadata,
        });
      } else if (node.type === "view") {
        tabs.addTab({
          type: "view",
          database: node.db,
          metadata: node.metadata,
        });
      } else if (node.type === "procedure") {
        tabs.addTab({
          type: "procedure",
          database: node.db,
          metadata: node.metadata,
        });
      } else if (node.type === "function") {
        tabs.addTab({
          type: "function",
          database: node.db,
          metadata: node.metadata,
        });
      } else if (node.type === "column") {
        tabs.addTab({
          type: "table",
          database: node.db,
          metadata: {
            database: node.metadata.database,
            schema: node.metadata.schema,
            table: node.metadata.table,
          },
        });
        scrollStatus.queuePendingScrollToColumn({
          db: node.db,
          metadata: node.metadata,
        });
      }
    },
    [tabs, scrollStatus]
  );

  const getNodeStatus = useCallback(
    (node: TreeNode): EditStatus => {
      if (node.type === "table")
        return editStatus.getTableStatus(node.db, node.metadata);
      if (node.type === "column")
        return editStatus.getColumnStatus(node.db, node.metadata);
      if (node.type === "schema")
        return editStatus.getSchemaStatus(node.db, node.metadata);
      if (node.type === "view")
        return editStatus.getViewStatus(node.db, node.metadata);
      if (node.type === "procedure")
        return editStatus.getProcedureStatus(node.db, node.metadata);
      if (node.type === "function")
        return editStatus.getFunctionStatus(node.db, node.metadata);
      return "normal";
    },
    [editStatus]
  );

  return (
    <div className="flex size-full flex-col gap-y-2">
      <div className="sticky top-0 px-1 pt-1">
        <SearchInput
          placeholder="Search..."
          onChange={(e) => handleSearchChange(e.target.value)}
          className="h-7"
        />
      </div>
      <div ref={containerRef} className="flex-1 overflow-hidden">
        <Tree
          data={arboristData}
          idAccessor="id"
          searchTerm={searchPattern}
          searchMatch={(node, term) =>
            node.data.name.toLowerCase().includes(term.toLowerCase())
          }
          rowHeight={28}
          indent={16}
          openByDefault={false}
          width="100%"
          height={containerRef.current?.clientHeight ?? 400}
        >
          {(props) => (
            <NodeRenderer
              {...props}
              nodeMap={nodeMap}
              onNodeClick={handleNodeClick}
              onContextMenu={readonly ? undefined : showMenu}
              getNodeStatus={getNodeStatus}
              selection={selection}
            />
          )}
        </Tree>
      </div>

      {/* Context menu portal */}
      {menuState.show &&
        menuOptions.length > 0 &&
        createPortal(
          <div
            className="fixed inset-0"
            onClick={hideMenu}
            onContextMenu={(e) => {
              e.preventDefault();
              hideMenu();
            }}
          >
            <div
              className={`absolute rounded-sm border border-control-border bg-white py-1 shadow-md ${LAYER_SURFACE_CLASS}`}
              style={{ left: menuState.x, top: menuState.y }}
              onClick={(e) => e.stopPropagation()}
            >
              {menuOptions.map((opt) => (
                <button
                  key={opt.key}
                  type="button"
                  className="flex w-full items-center px-3 py-1.5 text-left text-sm hover:bg-control-bg-hover"
                  onClick={() => handleMenuSelect(opt.key)}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>,
          getLayerRoot("overlay")
        )}

      {/* Modals */}
      {tableNameModalCtx && (
        <TableNameDialog
          open
          onClose={() => setTableNameModalCtx(null)}
          db={tableNameModalCtx.db}
          database={tableNameModalCtx.database}
          schema={tableNameModalCtx.schema}
          table={tableNameModalCtx.table}
        />
      )}
      {schemaNameModalCtx && (
        <SchemaNameDialog
          open
          onClose={() => setSchemaNameModalCtx(null)}
          db={schemaNameModalCtx.db}
          database={schemaNameModalCtx.database}
        />
      )}
    </div>
  );
}

// Helper to get resource key for view/procedure/function nodes
function getResourceKey(node: TreeNode) {
  if (node.type === "view") return { view: node.metadata.view };
  if (node.type === "procedure") return { procedure: node.metadata.procedure };
  if (node.type === "function") return { function: node.metadata.function };
  return {};
}

// Convert TreeNode[] to react-arborist compatible format
interface ArboristNode {
  id: string;
  name: string;
  children?: ArboristNode[];
}

function convertToArboristData(nodes: TreeNode[]): ArboristNode[] {
  return nodes.map((node) => ({
    id: node.key,
    name: node.label,
    children:
      node.children && node.children.length > 0
        ? convertToArboristData(node.children as TreeNode[])
        : node.isLeaf
          ? undefined
          : [],
  }));
}

// Node icon by type
function NodeIcon({ node }: { node: TreeNode }) {
  const cls = "size-4 shrink-0";
  switch (node.type) {
    case "instance":
      return <Layers className={cls} />;
    case "database":
      return <DatabaseIcon className={cls} />;
    case "schema":
      return <Layers className={cls} />;
    case "table":
      return <Table2 className={cls} />;
    case "column":
      return (
        <div className={cn(cls, "text-center text-xs font-bold leading-4")}>
          C
        </div>
      );
    case "view":
      return <View className={cls} />;
    case "procedure":
      return <FileCode className={cls} />;
    case "function":
      return <FunctionSquare className={cls} />;
    case "group":
      switch (node.group) {
        case "table":
          return <Table2 className={cls} />;
        case "view":
          return <View className={cls} />;
        case "procedure":
          return <FileCode className={cls} />;
        case "function":
          return <FunctionSquare className={cls} />;
      }
      return null;
    case "placeholder":
      return null;
  }
}

function statusClassName(status: EditStatus): string {
  switch (status) {
    case "created":
      return "text-success";
    case "updated":
      return "text-warning";
    case "dropped":
      return "text-error line-through";
    default:
      return "";
  }
}

// Custom node renderer
function NodeRenderer(
  props: NodeRendererProps<ArboristNode> & {
    nodeMap: Map<string, TreeNode>;
    onNodeClick: (node: TreeNode) => void;
    onContextMenu?: (e: React.MouseEvent, node: TreeNode) => void;
    getNodeStatus: (node: TreeNode) => EditStatus;
    selection: ReturnType<typeof useSchemaEditorContext>["selection"];
  }
) {
  const {
    node,
    style,
    nodeMap,
    onNodeClick,
    onContextMenu,
    getNodeStatus,
    selection,
  } = props;
  const treeNode = nodeMap.get(node.data.id);

  if (!treeNode) return null;
  if (treeNode.type === "placeholder") {
    return (
      <div
        style={style}
        className="flex h-7 items-center px-2 text-xs text-control-light italic"
      >
        No items
      </div>
    );
  }

  const status = getNodeStatus(treeNode);
  const hasChildren = !treeNode.isLeaf;

  return (
    <div
      style={style}
      className={cn(
        "flex h-7 cursor-pointer items-center gap-x-1 px-1 text-sm hover:bg-control-bg-hover",
        node.isSelected && "bg-control-bg-hover"
      )}
      onClick={(e) => {
        e.stopPropagation();
        if (hasChildren) node.toggle();
        onNodeClick(treeNode);
      }}
      onContextMenu={(e) => onContextMenu?.(e, treeNode)}
    >
      {hasChildren ? (
        <span className="flex size-4 shrink-0 items-center justify-center">
          {node.isOpen ? (
            <ChevronDown className="size-3" />
          ) : (
            <ChevronRight className="size-3" />
          )}
        </span>
      ) : (
        <span className="size-4 shrink-0" />
      )}
      <NodeCheckbox node={treeNode} selection={selection} />
      <NodeIcon node={treeNode} />
      <span className={cn("truncate", statusClassName(status))}>
        {treeNode.label || "(empty)"}
      </span>
    </div>
  );
}
