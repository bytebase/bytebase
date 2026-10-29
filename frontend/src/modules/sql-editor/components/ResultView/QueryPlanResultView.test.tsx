import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
    note,
    translate,
    disallowCopyingData,
  }: {
    planSource: string;
    textPlanSource?: string;
    note?: React.ReactNode;
    translate: (key: "tab.text") => string;
    disallowCopyingData?: boolean;
  }) => (
    <div
      data-testid="inline-query-plan"
      data-copy-disabled={disallowCopyingData}
    >
      {planSource}:{textPlanSource}:{translate("tab.text")}
      {note ? <span>{note}</span> : null}
    </div>
  ),
}));

vi.mock("@/apps/explain-visualizer/QueryPlanViewer", () => ({
  formatPlanSource: (source: string) => `formatted:${source}`,
}));

const COPY_PLAN = "sql-editor.query-plan-viewer.copy.plan";

describe("QueryPlanResultView", () => {
  beforeEach(() => vi.clearAllMocks());

  test("passes the structured and text plans to the inline viewer", () => {
    render(
      <QueryPlanResultView
        engine={Engine.POSTGRES}
        rawPlan="raw plan"
        initialPlan={{ source: "structured plan" }}
        disallowCopyingData
      />
    );

    expect(screen.getByTestId("inline-query-plan")).toHaveTextContent(
      "structured plan:raw plan:sql-editor.query-plan-viewer.tab.text"
    );
    expect(screen.queryByRole("tab")).toBeNull();
    expect(screen.getByTestId("inline-query-plan")).toHaveAttribute(
      "data-copy-disabled",
      "true"
    );
  });

  test("loads a drawable plan immediately and hands the viewer its text plan", async () => {
    const loadPlan = vi.fn().mockResolvedValue({
      source: "structured plan",
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
    const plan = { source: "structured plan" };
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
    // The same labelled copy control as the visualizer's Text tab.
    expect(screen.getByTestId("plan-copy-button")).toHaveAccessibleName(
      COPY_PLAN
    );
  });

  test("hides copying for a protected text plan", () => {
    render(
      <QueryPlanResultView rawPlan="text plan" disallowCopyingData />
    );

    expect(screen.queryByTestId("plan-copy-button")).toBeNull();
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
    expect(screen.getByTestId("plan-copy-button")).toHaveAccessibleName(
      COPY_PLAN
    );
  });

  test("loads a text plan on demand when no picture is available", async () => {
    const loadPlan = vi
      .fn()
      .mockResolvedValue({ source: "Seq Scan on t" });
    render(<QueryPlanResultView rawPlan="" loadPlan={loadPlan} estimated />);

    expect(screen.getByText("common.loading")).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByText("formatted:Seq Scan on t")).toBeInTheDocument()
    );
    expect(screen.getByText("sql-editor.plan-estimated")).toBeInTheDocument();
    expect(screen.queryAllByRole("tab")).toHaveLength(1);
    expect(loadPlan).toHaveBeenCalledWith(expect.any(AbortSignal));
  });

  test("cancel aborts the load and hands control back", () => {
    const onCancel = vi.fn();
    let signal: AbortSignal | undefined;
    const loadPlan = vi.fn((requested: AbortSignal) => {
      signal = requested;
      return new Promise<undefined>(() => {});
    });
    render(
      <QueryPlanResultView rawPlan="" loadPlan={loadPlan} onCancel={onCancel} />
    );

    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));

    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(signal?.aborted).toBe(true);
  });

  test("shows the engine's reason when the load fails", async () => {
    render(
      <QueryPlanResultView
        rawPlan=""
        loadPlan={vi
          .fn()
          .mockRejectedValue(new Error("permission denied to access resources"))}
      />
    );

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "permission denied to access resources"
      )
    );
  });

  test("puts the estimate note in the drawable viewer's tab row", async () => {
    const loadPlan = vi
      .fn()
      .mockResolvedValue({ source: "structured plan" });
    render(
      <QueryPlanResultView
        engine={Engine.POSTGRES}
        rawPlan=""
        loadPlan={loadPlan}
        estimated
      />
    );

    await waitFor(() =>
      expect(screen.getByTestId("inline-query-plan")).toHaveTextContent(
        "sql-editor.plan-estimated"
      )
    );
  });
});
