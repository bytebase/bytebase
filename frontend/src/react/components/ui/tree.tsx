import type { CSSProperties, ReactNode } from "react";
import { useEffect, useRef } from "react";
import type { MoveHandler, NodeApi, TreeApi } from "react-arborist";
import { Tree as ArboristTree } from "react-arborist";
import { cn } from "@/react/lib/utils";

export interface TreeDataNode<T> {
  readonly id: string;
  readonly children?: readonly TreeDataNode<T>[];
  readonly data: T;
}

export interface TreeProps<T> {
  readonly data: readonly TreeDataNode<T>[];
  readonly renderNode: (args: {
    node: NodeApi<TreeDataNode<T>>;
    style: CSSProperties;
    dragHandle?: (el: HTMLDivElement | null) => void;
  }) => ReactNode;

  readonly selectedIds?: readonly string[];
  readonly expandedIds?: readonly string[];
  readonly onSelect?: (ids: readonly string[]) => void;
  readonly onToggle?: (id: string) => void;

  readonly searchTerm?: string;
  readonly searchMatch?: (node: TreeDataNode<T>, term: string) => boolean;

  readonly height?: number;
  readonly rowHeight?: number;
  readonly indent?: number;

  // Drag-and-drop. Defaults are disabled — opt in by passing `onMove` AND
  // setting `disableDrag` / `disableDrop` to false (or a per-node predicate).
  readonly onMove?: MoveHandler<TreeDataNode<T>>;
  readonly disableDrag?: boolean | ((data: TreeDataNode<T>) => boolean);
  readonly disableDrop?:
    | boolean
    | ((args: {
        parentNode: NodeApi<TreeDataNode<T>>;
        dragNodes: NodeApi<TreeDataNode<T>>[];
        index: number;
      }) => boolean);

  readonly className?: string;
}

export function Tree<T>({
  data,
  renderNode,
  selectedIds,
  expandedIds,
  onSelect,
  onToggle,
  searchTerm,
  searchMatch,
  height = 300,
  rowHeight = 28,
  indent = 16,
  onMove,
  disableDrag = true,
  disableDrop = true,
  className,
}: TreeProps<T>) {
  const treeRef = useRef<TreeApi<TreeDataNode<T>> | null>(null);
  const prevExpandedRef = useRef<readonly string[] | undefined>(undefined);

  // Sync expandedIds to the arborist tree programmatically
  useEffect(() => {
    const tree = treeRef.current;
    if (!tree) return;

    const prev = new Set(prevExpandedRef.current ?? []);
    const next = new Set(expandedIds ?? []);

    // Close nodes that were previously open but no longer in expandedIds
    for (const id of prev) {
      if (!next.has(id)) {
        tree.close(id);
      }
    }

    // Open nodes that are in expandedIds but not previously open
    for (const id of next) {
      if (!prev.has(id)) {
        tree.open(id);
      }
    }

    prevExpandedRef.current = expandedIds;
  }, [expandedIds]);

  const handleSelect = (nodes: NodeApi<TreeDataNode<T>>[]) => {
    onSelect?.(nodes.map((n) => n.id));
  };

  const defaultSearchMatch = (
    node: NodeApi<TreeDataNode<T>>,
    term: string
  ): boolean => {
    const d = node.data.data as Record<string, unknown>;
    const label = typeof d?.label === "string" ? d.label : node.data.id;
    return label.toLowerCase().includes(term.toLowerCase());
  };

  const resolvedSearchMatch = searchMatch
    ? (node: NodeApi<TreeDataNode<T>>, term: string) =>
        searchMatch(node.data, term)
    : defaultSearchMatch;

  return (
    <div
      className={cn("outline-none focus:outline-none rounded-sm", className)}
    >
      <ArboristTree<TreeDataNode<T>>
        ref={treeRef}
        data={data as TreeDataNode<T>[]}
        idAccessor="id"
        childrenAccessor="children"
        selection={selectedIds?.[0]}
        onSelect={handleSelect}
        onToggle={onToggle}
        onMove={onMove}
        searchTerm={searchTerm}
        searchMatch={resolvedSearchMatch}
        openByDefault={false}
        disableDrag={disableDrag}
        disableDrop={disableDrop}
        disableEdit
        height={height}
        rowHeight={rowHeight}
        indent={indent}
        width="100%"
      >
        {({ node, style, dragHandle }) =>
          renderNode({ node, style, dragHandle })
        }
      </ArboristTree>
    </div>
  );
}
