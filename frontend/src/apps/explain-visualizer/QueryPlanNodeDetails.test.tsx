import { render, screen } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import type { PlanNode } from "./plan-model";
import { QueryPlanNodeDetails } from "./QueryPlanNodeDetails";

const node = (overrides: Partial<PlanNode> = {}): PlanNode => ({
  id: "0",
  nodeType: "Hash Join",
  subject: "orders",
  relationship: "Inner",
  startupCost: 140.5,
  totalCost: 1090.86,
  selfCost: 193.86,
  rows: 50000,
  width: 7,
  properties: [
    { label: "Hash Cond", value: "(o.customer_id = c.id)" },
    { label: "Join Type", value: "Inner" },
  ],
  children: [],
  ...overrides,
});

describe("QueryPlanNodeDetails", () => {
  test("prompts for a selection when no node is chosen", () => {
    render(<QueryPlanNodeDetails node={undefined} />);
    expect(
      screen.getByText("Select a node to see its estimates.")
    ).toBeInTheDocument();
  });

  test("shows the node identity, its estimates, and its properties", () => {
    render(<QueryPlanNodeDetails node={node()} />);

    expect(
      screen.getByRole("heading", { name: "Hash Join" })
    ).toBeInTheDocument();
    expect(screen.getByText("orders")).toBeInTheDocument();

    expect(screen.getByText("Startup cost").nextSibling).toHaveTextContent(
      "140.5"
    );
    expect(screen.getByText("Total cost").nextSibling).toHaveTextContent(
      "1,090.86"
    );
    expect(screen.getByText("Self cost").nextSibling).toHaveTextContent(
      "193.86"
    );
    expect(screen.getByText("Estimated rows").nextSibling).toHaveTextContent(
      "50,000"
    );
    expect(screen.getByText("Row width").nextSibling).toHaveTextContent(
      "7 bytes"
    );

    expect(screen.getByText("Hash Cond").nextSibling).toHaveTextContent(
      "(o.customer_id = c.id)"
    );
  });

  test("renders a filter literal as text, never as markup", () => {
    const injected = "(name = '<img src=x onerror=alert(1)>'::text)";
    const { container } = render(
      <QueryPlanNodeDetails
        node={node({ properties: [{ label: "Filter", value: injected }] })}
      />
    );

    expect(screen.getByText(injected)).toBeInTheDocument();
    expect(container.querySelector("img")).toBeNull();
  });

  test("omits the property list when the node has no extra attributes", () => {
    render(<QueryPlanNodeDetails node={node({ properties: [] })} />);
    expect(screen.queryByText("Hash Cond")).not.toBeInTheDocument();
    expect(screen.getByText("Total cost")).toBeInTheDocument();
  });

  test("explains a full table scan without needing a hover to do it", () => {
    render(
      <QueryPlanNodeDetails
        node={node({ nodeType: "Seq Scan", subject: "orders", totalCost: 819 })}
      />
    );

    const warning = screen.getByTestId("plan-details-full-scan");
    expect(warning).toHaveTextContent("Full table scan");
    expect(warning).toHaveTextContent(/an index on the filtered columns/);
    expect(screen.getByRole("alert")).toBe(warning);
  });

  test("says nothing about scans that are not worth flagging", () => {
    render(
      <QueryPlanNodeDetails
        node={node({ nodeType: "Seq Scan", subject: "regions", totalCost: 12 })}
      />
    );
    expect(screen.queryByTestId("plan-details-full-scan")).toBeNull();

    render(<QueryPlanNodeDetails node={node({ totalCost: 5000 })} />);
    expect(screen.queryByTestId("plan-details-full-scan")).toBeNull();
  });
});
