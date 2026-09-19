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
    planQuery,
    translate,
    disallowCopyingData,
    syncSelectionWithHash,
  }: {
    planSource: string;
    textPlanSource?: string;
    planQuery?: string;
    translate: (key: "tab.text") => string;
    disallowCopyingData?: boolean;
    syncSelectionWithHash?: boolean;
  }) => (
    <div
      data-testid="inline-query-plan"
      data-copy-disabled={disallowCopyingData}
      data-hash-sync={syncSelectionWithHash}
    >
      {planQuery}:{planSource}:{textPlanSource}:{translate("tab.text")}
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
        disallowCopyingData
      />
    );

    expect(screen.getByTestId("inline-query-plan")).toHaveTextContent(
      "SELECT 1:structured plan:raw plan:sql-editor.query-plan-viewer.tab.text"
    );
    expect(screen.queryByRole("tab")).toBeNull();
    expect(screen.getByTestId("inline-query-plan")).toHaveAttribute(
      "data-copy-disabled",
      "true"
    );
    expect(screen.getByTestId("inline-query-plan")).toHaveAttribute(
      "data-hash-sync",
      "false"
    );
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
        "structured plan:text plan:sql-editor.query-plan-viewer.tab.text"
      )
    );
    expect(loadPlan).toHaveBeenCalledTimes(1);
  });

  test("does not reload when only the loader identity changes", async () => {
    const plan = { source: "structured plan", statement: "EXPLAIN SELECT 1" };
    const first = vi.fn().mockResolvedValue(plan);
    const second = vi.fn().mockResolvedValue(plan);
    const { rerender } = render(
      <QueryPlanResultView
        engine={Engine.POSTGRES}
        rawPlan="text plan"
        loadPlan={first}
      />
    );
    await screen.findByTestId("inline-query-plan");

    rerender(
      <QueryPlanResultView
        engine={Engine.POSTGRES}
        rawPlan="text plan"
        loadPlan={second}
      />
    );

    expect(screen.getByTestId("inline-query-plan")).toBeInTheDocument();
    expect(first).toHaveBeenCalledTimes(1);
    expect(second).not.toHaveBeenCalled();
  });

  test("uses a single Plan tab when no picture is available", () => {
    render(<QueryPlanResultView rawPlan="text plan" />);

    expect(screen.getByRole("tab", { name: "common.plan" })).toHaveAttribute(
      "aria-selected",
      "true"
    );
    expect(screen.queryAllByRole("tab")).toHaveLength(1);
    expect(screen.getByText("formatted:text plan")).toBeInTheDocument();
  });

  test("hides copying for a protected text plan", () => {
    render(
      <QueryPlanResultView rawPlan="text plan" disallowCopyingData />
    );

    expect(screen.queryByRole("button", { name: "copy" })).toBeNull();
    expect(screen.getByText("formatted:text plan").parentElement).toHaveClass(
      "select-none"
    );
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
