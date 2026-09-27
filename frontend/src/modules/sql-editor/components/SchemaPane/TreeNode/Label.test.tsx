import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { NodeType, TreeNode } from "../schemaTree";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// Mock every leaf so Label's dispatch is the only thing exercised here.
// Each mock renders a deterministic data-testid so the assertion can match
// type → component without standing up the real render path (the leaves
// already have their own tests).
vi.mock("./DatabaseNode", () => ({
  DatabaseNode: () => <div data-testid="DatabaseNode" />,
}));
vi.mock("./SchemaNode", () => ({
  SchemaNode: () => <div data-testid="SchemaNode" />,
}));
vi.mock("./TableNode", () => ({
  TableNode: () => <div data-testid="TableNode" />,
}));
vi.mock("./ExternalTableNode", () => ({
  ExternalTableNode: () => <div data-testid="ExternalTableNode" />,
}));
vi.mock("./ViewNode", () => ({
  ViewNode: () => <div data-testid="ViewNode" />,
}));
vi.mock("./ColumnNode", () => ({
  ColumnNode: () => <div data-testid="ColumnNode" />,
}));
vi.mock("./IndexNode", () => ({
  IndexNode: () => <div data-testid="IndexNode" />,
}));
vi.mock("./ForeignKeyNode", () => ({
  ForeignKeyNode: () => <div data-testid="ForeignKeyNode" />,
}));
vi.mock("./CheckNode", () => ({
  CheckNode: () => <div data-testid="CheckNode" />,
}));
vi.mock("./PartitionTableNode", () => ({
  PartitionTableNode: () => <div data-testid="PartitionTableNode" />,
}));
vi.mock("./DependencyColumnNode", () => ({
  DependencyColumnNode: () => <div data-testid="DependencyColumnNode" />,
}));
vi.mock("./ProcedureNode", () => ({
  ProcedureNode: () => <div data-testid="ProcedureNode" />,
}));
vi.mock("./PackageNode", () => ({
  PackageNode: () => <div data-testid="PackageNode" />,
}));
vi.mock("./FunctionNode", () => ({
  FunctionNode: () => <div data-testid="FunctionNode" />,
}));
vi.mock("./SequenceNode", () => ({
  SequenceNode: () => <div data-testid="SequenceNode" />,
}));
vi.mock("./TriggerNode", () => ({
  TriggerNode: () => <div data-testid="TriggerNode" />,
}));
vi.mock("./TextNode", () => ({
  TextNode: () => <div data-testid="TextNode" />,
}));
vi.mock("./DummyNode", () => ({
  DummyNode: () => <div data-testid="DummyNode" />,
}));

let Label: typeof import("./Label").Label;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

const makeNode = (type: NodeType): TreeNode =>
  ({
    key: `${type}/k`,
    meta: { type, target: {} },
  }) as unknown as TreeNode;

beforeEach(async () => {
  ({ Label } = await import("./Label"));
});

const cases: ReadonlyArray<readonly [NodeType, string]> = [
  ["database", "DatabaseNode"],
  ["schema", "SchemaNode"],
  ["table", "TableNode"],
  ["external-table", "ExternalTableNode"],
  ["view", "ViewNode"],
  ["column", "ColumnNode"],
  ["index", "IndexNode"],
  ["foreign-key", "ForeignKeyNode"],
  ["check", "CheckNode"],
  ["partition-table", "PartitionTableNode"],
  ["dependency-column", "DependencyColumnNode"],
  ["procedure", "ProcedureNode"],
  ["package", "PackageNode"],
  ["function", "FunctionNode"],
  ["sequence", "SequenceNode"],
  ["trigger", "TriggerNode"],
  ["expandable-text", "TextNode"],
  ["error", "DummyNode"],
];

describe("SchemaPane Label", () => {
  for (const [type, expectedTestId] of cases) {
    test(`dispatches to ${expectedTestId} for type=${type}`, () => {
      const { container, render, unmount } = renderIntoContainer(
        <Label node={makeNode(type)} keyword="" />
      );
      render();
      expect(
        container.querySelector(`[data-testid="${expectedTestId}"]`)
      ).not.toBeNull();
      unmount();
    });
  }
});
