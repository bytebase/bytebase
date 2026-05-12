import type { CSSProperties, ReactNode } from "react";
import { useLayoutEffect, useRef } from "react";
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
  // react-arborist fires `onToggle` synchronously inside `tree.open()` /
  // `tree.close()` calls. If we forward those programmatic notifications to
  // the consumer's onToggle (which usually toggles the id in/out of the
  // expanded set), we get a feedback loop: setting expandedIds to N keys
  // → primitive opens N keys → Arborist fires onToggle for each → consumer
  // toggles keys back out → primitive closes them → onToggle fires again.
  // The flag below suppresses consumer notifications during the
  // programmatic batch so the consumer's state stays stable.
  const programmaticToggleRef = useRef(false);
  // When the user clicks the chevron, react-arborist toggles its INTERNAL
  // open state synchronously AND fires onToggle. The consumer then mirrors
  // that change in `expandedIds`, which fires the sync effect below — but
  // a re-applied `tree.open(id)` / `tree.close(id)` on the same id forces
  // react-arborist to recompute its visible-rows list a second time. For
  // a "Columns" node with hundreds of descendants that doubled the cost
  // of every collapse. Track the id the user just toggled and skip
  // syncing it on the next effect pass — react-arborist's internal state
  // is already in sync for that node.
  const userToggledIdRef = useRef<string | null>(null);

  // Sync expandedIds to the arborist tree programmatically.
  //
  // We use `useLayoutEffect` (not `useEffect`) deliberately: when the
  // consumer flips `expandedIds`, the FIRST commit shows the tree with
  // the new `expandedIds` prop but arborist's internal open-state is
  // still the OLD value, so the rendered rows haven't changed yet. The
  // sync below calls `tree.open(id)` / `tree.close(id)` which triggers
  // a SECOND commit inside arborist. With `useEffect` (passive), the
  // browser can paint the first (stale) commit before the second one
  // runs, and on some interactions only the next input event seemed to
  // give the browser a chance to flush the second paint — users saw
  // "click did nothing until I moved the mouse." `useLayoutEffect`
  // runs after commit but BEFORE paint, so the sync happens between
  // the two commits and the browser paints the final state in one go.
  useLayoutEffect(() => {
    const tree = treeRef.current;
    if (!tree) return;

    const prev = new Set(prevExpandedRef.current ?? []);
    const next = new Set(expandedIds ?? []);
    const skipId = userToggledIdRef.current;
    userToggledIdRef.current = null;

    programmaticToggleRef.current = true;
    try {
      // Close nodes that were previously open but no longer in expandedIds
      for (const id of prev) {
        if (id === skipId) continue;
        if (!next.has(id)) {
          tree.close(id);
        }
      }

      // Open nodes that are in expandedIds but not previously open
      for (const id of next) {
        if (id === skipId) continue;
        if (!prev.has(id)) {
          tree.open(id);
        }
      }
    } finally {
      programmaticToggleRef.current = false;
    }

    prevExpandedRef.current = expandedIds;
  }, [expandedIds]);

  const handleArboristToggle = (id: string) => {
    if (programmaticToggleRef.current) return;
    userToggledIdRef.current = id;
    onToggle?.(id);
  };

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
        onToggle={handleArboristToggle}
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
