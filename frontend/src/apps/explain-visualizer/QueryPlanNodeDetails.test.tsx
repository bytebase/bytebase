import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { PLAN_FULL_TABLE_SCAN, type PlanNode } from "./plan-model";
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
  warnings: [],
  children: [],
  ...overrides,
});

describe("QueryPlanNodeDetails", () => {
  test("prompts for a selection when no node is chosen", () => {
    render(<QueryPlanNodeDetails node={undefined} />);
    expect(
      screen.getByText("Select a node to see its details.")
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
    expect(screen.getByText("Added cost").nextSibling).toHaveTextContent(
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

  test("explains every warning without needing a hover to do it", () => {
    render(
      <QueryPlanNodeDetails
        node={node({
          warnings: [
            PLAN_FULL_TABLE_SCAN,
            { title: "No join predicate", detail: "Every row pairs up." },
          ],
        })}
      />
    );

    const warnings = screen.getAllByTestId("plan-details-warning");
    expect(warnings).toHaveLength(2);
    expect(warnings[0]).toHaveTextContent("Full table scan");
    expect(warnings[0]).toHaveTextContent(/an index on the filtered columns/);
    expect(warnings[1]).toHaveTextContent("No join predicate");
    expect(screen.getAllByRole("alert")).toEqual(warnings);
  });

  test("shows a warning the engine repeats as often as it reports it", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    render(
      <QueryPlanNodeDetails
        node={node({ warnings: [PLAN_FULL_TABLE_SCAN, PLAN_FULL_TABLE_SCAN] })}
      />
    );

    expect(screen.getAllByTestId("plan-details-warning")).toHaveLength(2);
    // React reports colliding keys through console.error.
    expect(consoleError).not.toHaveBeenCalled();
    consoleError.mockRestore();
  });

  test("says nothing when the node has no warnings", () => {
    render(<QueryPlanNodeDetails node={node()} />);
    expect(screen.queryByTestId("plan-details-warning")).toBeNull();
  });

  test("lists only the estimates the node has", () => {
    render(
      <QueryPlanNodeDetails
        node={node({
          startupCost: undefined,
          width: undefined,
          totalCost: 0.2138,
          selfCost: 0.2138,
          rows: 50,
        })}
      />
    );

    expect(screen.getByText("Total cost").nextSibling).toHaveTextContent(
      "0.21"
    );
    expect(screen.getByText("Estimated rows")).toBeInTheDocument();
    expect(screen.queryByText("Startup cost")).toBeNull();
    expect(screen.queryByText("Row width")).toBeNull();
  });

  test("leaves out the estimates section of a node without any", () => {
    const { container } = render(
      <QueryPlanNodeDetails
        node={{
          id: "0",
          nodeType: "Distributed Union",
          properties: [{ label: "Split Range", value: "true" }],
          warnings: [],
          children: [],
        }}
      />
    );

    expect(screen.queryByText("Total cost")).toBeNull();
    // The property list is the only list left.
    expect(container.querySelectorAll("dl")).toHaveLength(1);
    expect(screen.getByText("Split Range").nextSibling).toHaveTextContent(
      "true"
    );
  });
});
