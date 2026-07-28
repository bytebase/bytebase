import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import { SpannerPlanNode } from "./SpannerPlanNode";
import type { SpannerPlanNodeData } from "./spanner-types";

const node = (
  index: number,
  kind: string,
  displayName: string,
  extra: Partial<SpannerPlanNodeData> = {}
): SpannerPlanNodeData => ({
  index,
  kind,
  displayName,
  ...extra,
});

describe("SpannerPlanNode", () => {
  test("renders a leaf node with its kind and displayName", () => {
    const root = node(0, "RELATIONAL", "Distributed Union");
    render(<SpannerPlanNode node={root} allNodes={[root]} depth={0} />);
    expect(screen.getByText("RELATIONAL")).toBeInTheDocument();
    expect(screen.getByText("Distributed Union")).toBeInTheDocument();
  });

  test("renders only RELATIONAL child links, skipping scalar links", () => {
    const root = node(0, "RELATIONAL", "Distributed Union", {
      childLinks: [{ childIndex: 1 }, { childIndex: 2, type: "Reference" }],
    });
    const relChild = node(1, "RELATIONAL", "Hash Join");
    const scalarChild = node(2, "SCALAR", "Param");
    render(
      <SpannerPlanNode
        node={root}
        allNodes={[root, relChild, scalarChild]}
        depth={0}
      />
    );
    expect(screen.getByText("Hash Join")).toBeInTheDocument();
    expect(screen.queryByText("Param")).not.toBeInTheDocument();
  });

  test("collapses children when the node row is clicked", () => {
    const root = node(0, "RELATIONAL", "Distributed Union", {
      childLinks: [{ childIndex: 1 }],
    });
    const child = node(1, "RELATIONAL", "Hash Join");
    render(<SpannerPlanNode node={root} allNodes={[root, child]} depth={0} />);
    expect(screen.getByText("Hash Join")).toBeInTheDocument();
    // The root's row is the first element with class `ev-spanner-row`.
    const row = document.querySelector(".ev-spanner-row");
    expect(row).not.toBeNull();
    fireEvent.click(row as Element);
    expect(screen.queryByText("Hash Join")).not.toBeInTheDocument();
  });

  test("filters out metadata keys that start with `_`", () => {
    const root = node(0, "RELATIONAL", "Scan", {
      metadata: {
        table: "users",
        _internal_id: "skip-me",
      },
    });
    render(<SpannerPlanNode node={root} allNodes={[root]} depth={0} />);
    expect(screen.getByText("table:")).toBeInTheDocument();
    expect(screen.getByText("users")).toBeInTheDocument();
    expect(screen.queryByText("_internal_id:")).not.toBeInTheDocument();
    expect(screen.queryByText("skip-me")).not.toBeInTheDocument();
  });

  test("renders shortRepresentation.description when present", () => {
    const root = node(0, "RELATIONAL", "Filter", {
      shortRepresentation: { description: "age > 18" },
    });
    render(<SpannerPlanNode node={root} allNodes={[root]} depth={0} />);
    expect(screen.getByText("age > 18")).toBeInTheDocument();
  });

  test("renders unknown placeholder for missing child indexes", () => {
    const root = node(0, "RELATIONAL", "Union", {
      childLinks: [{ childIndex: 42 }],
    });
    // Child 42 is missing from allNodes — but the parent only renders
    // children whose RELATIONAL kind matches. With a missing child the
    // filter drops the link, so nothing renders for it.
    render(<SpannerPlanNode node={root} allNodes={[root]} depth={0} />);
    expect(screen.queryByText(/Unknown Node/)).not.toBeInTheDocument();
  });
});
