import { useMemo } from "react";
import {
  Tree as ArboristTree,
  type TreeDataNode,
} from "@/react/components/ui/tree";
import { cn } from "@/react/lib/utils";
import { getInstanceResource, hasSchemaProperty } from "@/utils";
import { DEFAULT_PADDINGS } from "../common/const";
import { useSchemaDiagramContext } from "../common/context";
import { Label, Prefix, Suffix } from "./TreeNode";
import type { NavigatorTreeNode } from "./types";
import { isTypedNode } from "./utils";

interface NavigatorTreeProps {
  keyword?: string;
  /** Outer height of the virtualized tree viewport. */
  height?: number;
}

/**
 * React port of `Navigator/Tree.vue`. Reuses the shared `Tree` primitive
 * (react-arborist) and renders schema → table levels with prefix /
 * suffix slots. Clicking a table emits `set-center` so the canvas
 * recenters on it.
 */
export function NavigatorTree({
  keyword = "",
  height = 480,
}: NavigatorTreeProps) {
  const ctx = useSchemaDiagramContext();
  const { selectedSchemas, events, database } = ctx;

  const isFlatTree = useMemo(
    () => hasSchemaProperty(getInstanceResource(database).engine),
    [database]
  );

  const data = useMemo<readonly TreeDataNode<NavigatorTreeNode>[]>(() => {
    const schemaNodes = selectedSchemas.map((schema) => {
      const tableChildren: TreeDataNode<NavigatorTreeNode>[] =
        schema.tables.map((table) => ({
          id: `table-${schema.name}-${table.name}`,
          data: {
            id: `table-${schema.name}-${table.name}`,
            type: "table" as const,
            data: table,
          },
        }));
      const node: TreeDataNode<NavigatorTreeNode> = {
        id: `schema-${schema.name}`,
        children: tableChildren,
        data: {
          id: `schema-${schema.name}`,
          type: "schema" as const,
          data: schema,
          children: tableChildren.map((c) => c.data),
        },
      };
      return node;
    });
    // Mirror Vue: collapse the tree when there's a single, unnamed schema
    // (e.g. MySQL's default schema) — show only the table children.
    if (schemaNodes.length === 1 && selectedSchemas[0].name === "") {
      return schemaNodes[0].children ?? [];
    }
    return schemaNodes;
  }, [selectedSchemas]);

  // Match Vue's `default-expand-all="true"` — every schema row opens
  // expanded so the table children are visible without manual clicking.
  const expandedIds = useMemo(() => data.map((node) => node.id), [data]);

  return (
    <ArboristTree<NavigatorTreeNode>
      className={cn(
        "bb-schema-diagram-nav-tree select-none",
        isFlatTree && "flat"
      )}
      data={data}
      height={height}
      expandedIds={expandedIds}
      searchTerm={keyword}
      searchMatch={(node, term) =>
        (node.data.data.name ?? "").toLowerCase().includes(term.toLowerCase())
      }
      renderNode={({ node, style, dragHandle }) => {
        const treeNode = node.data.data;
        return (
          <div
            ref={dragHandle}
            style={style}
            className={cn(
              "group flex items-center gap-x-1 px-1 cursor-pointer hover:bg-control-bg"
            )}
            onClick={() => {
              if (isTypedNode(treeNode, "schema")) {
                node.toggle();
                return;
              }
              if (isTypedNode(treeNode, "table")) {
                void events.emit("set-center", {
                  type: "table",
                  target: treeNode.data,
                  padding: DEFAULT_PADDINGS,
                });
              }
            }}
            data-node-type={treeNode.type}
          >
            <Prefix node={treeNode} />
            <div className="flex-1 min-w-0">
              <Label node={treeNode} keyword={keyword} />
            </div>
            <Suffix node={treeNode} />
          </div>
        );
      }}
    />
  );
}
