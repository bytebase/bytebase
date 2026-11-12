import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type {
  AffectedRowsSummary,
  TargetAffectedRows,
} from "../utils/affectedRows";
import { AffectedRowsStrip } from "./AffectedRowsStrip";

const mocks = vi.hoisted(() => ({
  allowRunChecks: true,
  runChecks: vi.fn(async () => {}),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      options ? `${key}(${Object.values(options).join(",")})` : key,
  }),
}));
vi.mock("@/components/HumanizeTs", () => ({
  HumanizeTs: () => <span>time</span>,
}));
vi.mock("@/components/ui/sheet", () => {
  const Pass = ({ children }: { children: ReactNode }) => <div>{children}</div>;
  return {
    Sheet: ({ children, open }: { children: ReactNode; open: boolean }) =>
      open ? <div data-testid="details">{children}</div> : null,
    SheetBody: Pass,
    SheetContent: Pass,
    SheetDescription: Pass,
    SheetHeader: Pass,
    SheetTitle: Pass,
  };
});
vi.mock("./PlanTargetDisplay", () => ({
  PlanTargetDisplay: ({ target }: { target: string }) => <span>{target}</span>,
}));
vi.mock("../hooks/usePlanCheckActions", () => ({
  usePlanCheckActions: () => ({
    allowRunChecks: mocks.allowRunChecks,
    isRunningChecks: false,
    runChecks: mocks.runChecks,
  }),
}));
vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({ isCreating: false }),
}));

const target = (
  database: string,
  overrides: Partial<TargetAffectedRows> = {}
): TargetAffectedRows => ({
  target: `instances/i/databases/${database}`,
  status: "estimated",
  rows: 600n,
  title: "OK",
  reason: "",
  ...overrides,
});

const renderStrip = (
  targets: TargetAffectedRows[],
  onGoToLine?: (line: number) => void
) => {
  const summary: AffectedRowsSummary = { targets };
  return render(
    <AffectedRowsStrip onGoToLine={onGoToLine} summary={summary} />
  );
};

beforeEach(() => {
  mocks.allowRunChecks = true;
  mocks.runChecks.mockClear();
});
afterEach(cleanup);

describe("AffectedRowsStrip", () => {
  test("lists each database's estimate and offers no rerun when all are complete", () => {
    renderStrip([target("a"), target("b", { rows: 1200n })]);
    const strip = screen.getByTestId("affected-rows");
    expect(strip.textContent).toContain("plan.affected-rows.on(600,a)");
    expect(strip.textContent).toContain(
      `plan.affected-rows.on(${(1200).toLocaleString()},b)`
    );
    expect(strip.textContent).not.toContain("plan.affected-rows.incomplete");
    expect(screen.queryByText("plan.affected-rows.rerun")).toBeNull();
  });

  test("marks an incomplete estimate and opens Details on the incomplete databases", () => {
    renderStrip([
      target("a"),
      target("b", {
        status: "incomplete",
        rows: 200170n,
        reason: "permission denied for table products",
      }),
    ]);
    const strip = screen.getByTestId("affected-rows");
    expect(strip.textContent).toContain(
      `plan.affected-rows.on(${(200170).toLocaleString()},b)`
    );
    // The reason waits in Details.
    expect(strip.textContent).not.toContain("permission denied");

    fireEvent.click(screen.getByText("plan.affected-rows.incomplete"));

    const details = screen.getByTestId("details");
    expect(details.textContent).toContain("instances/i/databases/b");
    expect(details.textContent).not.toContain("instances/i/databases/a");
    expect(details.textContent).toContain("plan.affected-rows.incomplete");
    expect(details.textContent).toContain(
      "permission denied for table products"
    );
  });

  test("reruns the checks when an estimate could not run EXPLAIN", () => {
    renderStrip([target("a", { status: "incomplete" })]);

    fireEvent.click(screen.getByText("plan.affected-rows.rerun"));

    expect(mocks.runChecks).toHaveBeenCalledTimes(1);
  });

  test("hides rerun from people who cannot run checks", () => {
    mocks.allowRunChecks = false;
    renderStrip([target("a", { status: "incomplete" })]);
    expect(screen.queryByText("plan.affected-rows.rerun")).toBeNull();
  });

  test("names only the largest estimate beyond three databases", () => {
    renderStrip([
      target("a"),
      target("b", { rows: 9000n }),
      target("c"),
      target("d", { status: "incomplete" }),
    ]);
    const strip = screen.getByTestId("affected-rows");
    expect(strip.textContent).toContain(
      `plan.affected-rows.up-to-on(${(9000).toLocaleString()},b)`
    );
    expect(strip.textContent).toContain("plan.affected-rows.n-databases(4)");
    expect(strip.textContent).toContain(
      "plan.affected-rows.incomplete-on(1)"
    );
  });

  test("shows a syntax error with a way to its line and its diagnostic in Details", () => {
    const onGoToLine = vi.fn();
    renderStrip(
      [
        target("a", {
          status: "not-estimated",
          rows: 0n,
          title: "Syntax error",
          reason: 'Syntax error at line 2:59 related text: "AND;"',
          line: 2,
        }),
      ],
      onGoToLine
    );
    const strip = screen.getByTestId("affected-rows");
    expect(strip.textContent).toContain("plan.affected-rows.not-estimated");
    expect(strip.textContent).toContain("Syntax error");

    fireEvent.click(screen.getByText("plan.affected-rows.go-to-line(2)"));
    expect(onGoToLine).toHaveBeenCalledWith(2);

    fireEvent.click(screen.getByText("plan.affected-rows.details"));
    expect(screen.getByTestId("details").textContent).toContain(
      'Syntax error at line 2:59 related text: "AND;"'
    );
  });
});
