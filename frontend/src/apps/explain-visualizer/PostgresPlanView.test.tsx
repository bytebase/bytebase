import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import cteNestedLoopInitplan from "./fixtures/cte-nested-loop-initplan.json";
import { PostgresPlanView } from "./PostgresPlanView";
import {
  POSTGRES_PLAN_EMPTY_MESSAGE,
  POSTGRES_PLAN_INVALID_JSON_MESSAGE,
  POSTGRES_PLAN_NO_PLAN_MESSAGE,
} from "./postgres-plan";

vi.mock("react-resizable-panels", () => ({
  Group: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  Panel: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  Separator: () => <div />,
}));

describe("PostgresPlanView", () => {
  beforeEach(() => {
    // jsdom's window exposes no matchMedia under vitest; the viewer queries it
    // to decide whether to stack the detail pane.
    window.matchMedia = ((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => true,
    })) as unknown as typeof window.matchMedia;
  });

  test("renders every node of a real EXPLAIN (FORMAT JSON) plan", () => {
    render(
      <PostgresPlanView
        planSource={JSON.stringify(cteNestedLoopInitplan)}
        planQuery="SELECT c.region FROM orders JOIN customers c ON c.id = orders.customer_id LIMIT 10;"
      />
    );

    for (const nodeType of [
      "Limit",
      "Aggregate",
      "Nested Loop",
      "Seq Scan",
      "Index Scan",
    ]) {
      expect(
        screen.getAllByRole("button", { name: new RegExp(nodeType) }).length
      ).toBeGreaterThan(0);
    }
    expect(screen.getByText("InitPlan 1")).toBeInTheDocument();
  });

  test("explains malformed JSON and still shows what the server returned", () => {
    render(<PostgresPlanView planSource="Seq Scan on customers" />);

    expect(screen.getByRole("alert")).toHaveTextContent(
      POSTGRES_PLAN_INVALID_JSON_MESSAGE
    );
    expect(screen.getByText("Seq Scan on customers")).toBeInTheDocument();
    expect(screen.queryByTestId("plan-diagram-viewport")).toBeNull();
  });

  test("explains JSON that carries no plan", () => {
    render(<PostgresPlanView planSource='[{"Planning Time": 0.1}]' />);

    expect(screen.getByRole("alert")).toHaveTextContent(
      POSTGRES_PLAN_NO_PLAN_MESSAGE
    );
  });

  test("explains an empty plan without echoing an empty block", () => {
    const { container } = render(<PostgresPlanView planSource="" />);

    expect(screen.getByRole("alert")).toHaveTextContent(
      POSTGRES_PLAN_EMPTY_MESSAGE
    );
    expect(container.querySelector("pre")).toBeNull();
  });
});
