import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { QueryPlanResultView } from "./QueryPlanResultView";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/apps/explain-visualizer/QueryPlanView", () => ({
  QueryPlanView: ({
    planSource,
    textPlanSource,
    textTabLabel,
    planQuery,
  }: {
    planSource: string;
    textPlanSource?: string;
    textTabLabel?: string;
    planQuery?: string;
  }) => (
    <div data-testid="inline-query-plan">
      {planQuery}:{planSource}:{textPlanSource}:{textTabLabel}
    </div>
  ),
}));

vi.mock("@/apps/explain-visualizer/QueryPlanViewer", () => ({
  formatPlanSource: (source: string) => `formatted:${source}`,
}));

vi.mock("@/components/ui/copy-button", () => ({
  CopyButton: ({ content }: { content: string }) => (
    <button type="button" data-content={content}>
      copy
    </button>
  ),
}));

describe("QueryPlanResultView", () => {
  beforeEach(() => vi.clearAllMocks());

  test("passes the structured and text plans to the inline viewer", () => {
    render(
      <QueryPlanResultView
        engine={Engine.POSTGRES}
        rawPlan="raw plan"
        initialPlan={{ source: "structured plan", statement: "SELECT 1" }}
      />
    );

    expect(screen.getByTestId("inline-query-plan")).toHaveTextContent(
      "SELECT 1:structured plan:raw plan:sql-editor.query-plan-text"
    );
    expect(screen.queryByRole("tab")).toBeNull();
  });

  test("loads a drawable plan immediately and opens on its text tab", async () => {
    const loadPlan = vi.fn().mockResolvedValue({
      source: "structured plan",
      statement: "EXPLAIN SELECT 1",
    });
    render(
      <QueryPlanResultView
        engine={Engine.POSTGRES}
        rawPlan="text plan"
        loadPlan={loadPlan}
      />
    );

    expect(screen.getByText("common.loading")).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByTestId("inline-query-plan")).toHaveTextContent(
        "structured plan:text plan:sql-editor.query-plan-text"
      )
    );
    expect(loadPlan).toHaveBeenCalledTimes(1);
  });

  test("uses a single Plan tab when no picture is available", () => {
    render(<QueryPlanResultView rawPlan="text plan" />);

    expect(screen.getByRole("tab", { name: "common.plan" })).toHaveAttribute(
      "aria-selected",
      "true"
    );
    expect(
      screen.queryByRole("tab", { name: "sql-editor.query-plan-text" })
    ).toBeNull();
    expect(screen.getByText("formatted:text plan")).toBeInTheDocument();
  });

  test("keeps the text plan visible when the structured plan fails", async () => {
    render(
      <QueryPlanResultView
        engine={Engine.POSTGRES}
        rawPlan="text plan"
        loadPlan={vi.fn().mockResolvedValue(undefined)}
      />
    );

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "sql-editor.query-plan-load-failed"
      )
    );
    expect(screen.getByText("formatted:text plan")).toBeInTheDocument();
  });
});
