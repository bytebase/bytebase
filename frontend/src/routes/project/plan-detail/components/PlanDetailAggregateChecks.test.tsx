import { create } from "@bufbuild/protobuf";
import { act, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, expect, test, vi } from "vitest";
import { PlanCheckRun_ResultSchema, type PlanCheckRun_Result } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

const mocks = vi.hoisted(() => ({
  page: {} as Record<string, unknown>,
  runChecks: vi.fn(),
  open: vi.fn(),
  section: {} as { onRun?: () => Promise<void> },
}));

vi.mock("@/components/plan-check/PlanCheckSection", () => ({
  PlanCheckSection: (props: { onRun: () => Promise<void>; renderResultAction?: (result: PlanCheckRun_Result) => ReactNode }) => {
    mocks.section = props;
    return <div>{props.renderResultAction?.(create(PlanCheckRun_ResultSchema, { status: Advice_Level.ERROR, title: "Rule violation", content: "Missing WHERE", sheetSha256: "a".repeat(64) }))}</div>;
  },
}));

vi.mock("@/lib/plan/check", () => ({
  getPlanCheckSummaryWithFallback: () => ({ total: 1 }),
}));

vi.mock("../hooks/usePlanCheckActions", () => ({
  usePlanCheckActions: () => ({
    allowRunChecks: true,
    isRunningChecks: false,
    refreshChecks: vi.fn(),
    runChecks: mocks.runChecks,
  }),
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

vi.mock("./PlanTargetDisplay", () => ({ PlanTargetDisplay: () => null }));
vi.mock("@/modules/agent/logic/instanceConnectionHelp", () => ({ openAgentWithPrompt: (...args: unknown[]) => mocks.open(...args) }));

import { PlanDetailAggregateChecks } from "./PlanDetailAggregateChecks";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.page = { projectId: "foo", plan: { name: "projects/foo/plans/7" }, planCheckRuns: [{ name: "projects/foo/plans/7/planCheckRun" }] };
  mocks.runChecks.mockResolvedValue(undefined);
});

test("click diagnoses the saved owning Plan and original hash without attaching current SQL", () => {
  render(<PlanDetailAggregateChecks />);
  expect(mocks.open).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Ask Agent" }));
  const [prompt, references] = mocks.open.mock.calls[0];
  expect(references).toEqual([expect.objectContaining({ kind: "plan", name: "projects/foo/plans/7", project: "projects/foo" })]);
  expect(prompt).toContain('"sourceId":"projects/foo/plans/7/planCheckRun"');
  expect(prompt).toContain("a".repeat(64));
  expect(prompt).not.toContain('"sql":');
});

test.each([
  { projectId: "other" }, { isCreating: true }, { plan: { name: "projects/foo/plans/create" } },
  { planCheckRuns: [{ name: "projects/other/plans/7/planCheckRun" }] },
])("hides help when saved ownership cannot be proven: %j", (patch) => {
  Object.assign(mocks.page, patch);
  render(<PlanDetailAggregateChecks />);
  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();
});

test("runs the plan checks from the footer", async () => {
  render(<PlanDetailAggregateChecks />);
  await act(async () => {
    await mocks.section.onRun?.();
  });
  expect(mocks.runChecks).toHaveBeenCalledTimes(1);
});
