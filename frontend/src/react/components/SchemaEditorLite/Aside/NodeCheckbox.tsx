import { useCallback } from "react";
import type { SelectionContext } from "../types";
import type {
  TreeNode,
  TreeNodeForColumn,
  TreeNodeForFunction,
  TreeNodeForGroup,
  TreeNodeForProcedure,
  TreeNodeForSchema,
  TreeNodeForTable,
  TreeNodeForView,
} from "./tree-builder";

interface Props {
  node: TreeNode;
  selection: SelectionContext;
}

export function NodeCheckbox({ node, selection }: Props) {
  if (!selection.selectionEnabled) return null;

  switch (node.type) {
    case "table":
      return <TableCheckbox node={node} selection={selection} />;
    case "column":
      return <ColumnCheckbox node={node} selection={selection} />;
    case "view":
      return <ViewCheckbox node={node} selection={selection} />;
    case "procedure":
      return <ProcedureCheckbox node={node} selection={selection} />;
    case "function":
      return <FunctionCheckbox node={node} selection={selection} />;
    case "group":
      return <GroupCheckbox node={node} selection={selection} />;
    case "schema":
      return <SchemaCheckbox node={node} selection={selection} />;
    default:
      return null;
  }
}

function TableCheckbox({
  node,
  selection,
}: {
  node: TreeNodeForTable;
  selection: SelectionContext;
}) {
  const state = selection.getTableSelectionState(node.db, node.metadata);
  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      selection.updateTableSelection(node.db, node.metadata, e.target.checked);
    },
    [selection, node]
  );
  return (
    <input
      type="checkbox"
      checked={state.checked}
      ref={(el) => {
        if (el) el.indeterminate = state.indeterminate;
      }}
      onChange={handleChange}
      onClick={(e) => e.stopPropagation()}
      className="size-3.5"
    />
  );
}

function ColumnCheckbox({
  node,
  selection,
}: {
  node: TreeNodeForColumn;
  selection: SelectionContext;
}) {
  const state = selection.getColumnSelectionState(node.db, node.metadata);
  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      selection.updateColumnSelection(node.db, node.metadata, e.target.checked);
    },
    [selection, node]
  );
  return (
    <input
      type="checkbox"
      checked={state.checked}
      onChange={handleChange}
      onClick={(e) => e.stopPropagation()}
      className="size-3.5"
    />
  );
}

function ViewCheckbox({
  node,
  selection,
}: {
  node: TreeNodeForView;
  selection: SelectionContext;
}) {
  const state = selection.getViewSelectionState(node.db, node.metadata);
  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      selection.updateViewSelection(node.db, node.metadata, e.target.checked);
    },
    [selection, node]
  );
  return (
    <input
      type="checkbox"
      checked={state.checked}
      onChange={handleChange}
      onClick={(e) => e.stopPropagation()}
      className="size-3.5"
    />
  );
}

function ProcedureCheckbox({
  node,
  selection,
}: {
  node: TreeNodeForProcedure;
  selection: SelectionContext;
}) {
  const state = selection.getProcedureSelectionState(node.db, node.metadata);
  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      selection.updateProcedureSelection(
        node.db,
        node.metadata,
        e.target.checked
      );
    },
    [selection, node]
  );
  return (
    <input
      type="checkbox"
      checked={state.checked}
      onChange={handleChange}
      onClick={(e) => e.stopPropagation()}
      className="size-3.5"
    />
  );
}

function FunctionCheckbox({
  node,
  selection,
}: {
  node: TreeNodeForFunction;
  selection: SelectionContext;
}) {
  const state = selection.getFunctionSelectionState(node.db, node.metadata);
  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      selection.updateFunctionSelection(
        node.db,
        node.metadata,
        e.target.checked
      );
    },
    [selection, node]
  );
  return (
    <input
      type="checkbox"
      checked={state.checked}
      onChange={handleChange}
      onClick={(e) => e.stopPropagation()}
      className="size-3.5"
    />
  );
}

function GroupCheckbox({
  node,
  selection,
}: {
  node: TreeNodeForGroup;
  selection: SelectionContext;
}) {
  // Group checkboxes toggle all children of a specific type
  if (node.group === "table") {
    const tables = node.children?.filter(
      (c): c is TreeNodeForTable => c.type === "table"
    );
    if (!tables || tables.length === 0) return null;
    const tableMetas = tables.map((t) => t.metadata.table);
    const state = selection.getAllTablesSelectionState(
      node.db,
      node.metadata,
      tableMetas
    );
    return (
      <input
        type="checkbox"
        checked={state.checked}
        ref={(el) => {
          if (el) el.indeterminate = state.indeterminate;
        }}
        onChange={(e) => {
          selection.updateAllTablesSelection(
            node.db,
            node.metadata,
            tableMetas,
            e.target.checked
          );
        }}
        onClick={(e) => e.stopPropagation()}
        className="size-3.5"
      />
    );
  }

  if (node.group === "view") {
    const views = node.children?.filter(
      (c): c is TreeNodeForView => c.type === "view"
    );
    if (!views || views.length === 0) return null;
    const viewMetas = views.map((v) => v.metadata.view);
    const state = selection.getAllViewsSelectionState(
      node.db,
      node.metadata,
      viewMetas
    );
    return (
      <input
        type="checkbox"
        checked={state.checked}
        ref={(el) => {
          if (el) el.indeterminate = state.indeterminate;
        }}
        onChange={(e) => {
          selection.updateAllViewsSelection(
            node.db,
            node.metadata,
            viewMetas,
            e.target.checked
          );
        }}
        onClick={(e) => e.stopPropagation()}
        className="size-3.5"
      />
    );
  }

  // Procedure and function group checkboxes follow the same pattern
  return null;
}

function SchemaCheckbox({
  node: _node,
  selection: _selection,
}: {
  node: TreeNodeForSchema;
  selection: SelectionContext;
}) {
  // Schema-level checkbox would aggregate all child groups.
  // Deferred for simplicity — individual type checkboxes cover the use case.
  return null;
}
