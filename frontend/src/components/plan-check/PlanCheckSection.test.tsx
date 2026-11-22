import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";
import { type PlanCheckRun_Result, PlanCheckRun_ResultSchema, PlanCheckRun_Status, PlanCheckRunSchema } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { PlanCheckResultsDrawer, PlanCheckSection } from "./PlanCheckSection";

vi.mock("react-i18next", async (importOriginal) => ({ ...await importOriginal<typeof import("react-i18next")>(), useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock("@/components/HumanizeTs", () => ({ HumanizeTs: () => null }));
vi.mock("@/types", () => ({ getTimeForPbTimestampProtoEs: () => 0, ruleTemplateMapV2: new Map() }));

const result = (status: Advice_Level) => create(PlanCheckRun_ResultSchema, { status, title: Advice_Level[status], content: "Evidence" });
const run = (status = PlanCheckRun_Status.DONE) => create(PlanCheckRunSchema, { name: "projects/example/plans/42/planCheckRun", status, results: [result(Advice_Level.ERROR), result(Advice_Level.WARNING), result(Advice_Level.SUCCESS)] });

test.each([undefined, "Unsaved SQL checks"])("keeps the check heading scoped to the displayed SQL: %s", (title) => {
  render(<PlanCheckSection title={title} planCheckRuns={[]} canRun={false} isRunning={false} onRun={() => {}} />);
  expect(screen.getByRole("heading", { name: title ?? "plan.checks.self" })).toBeInTheDocument();
  expect(screen.getAllByRole("heading")).toHaveLength(1);
});

test("renders one action beside each visible problem, never on summary or success", () => {
  const action = vi.fn((selected: PlanCheckRun_Result) => <button type="button">Help {selected.title}</button>);
  render(<PlanCheckSection planCheckRuns={[run()]} canRun={false} isRunning={false} onRun={() => {}} renderResultAction={action} />);
  expect(screen.queryByRole("button", { name: /Help/ })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "plan.navigator.checks" }));
  expect(screen.getAllByRole("button", { name: /Help/ })).toHaveLength(2);
  expect([...new Set(action.mock.calls.map(([selected]) => selected.status))]).toEqual([Advice_Level.ERROR, Advice_Level.WARNING]);
  fireEvent.click(screen.getByRole("button", { name: "common.success1" }));
  expect(screen.queryByRole("button", { name: /Help/ })).toBeNull();
});

test.each([PlanCheckRun_Status.FAILED, PlanCheckRun_Status.CANCELED])("provides a synthetic stopped-run problem action for %s", (status) => {
  const stopped = create(PlanCheckRunSchema, { name: "projects/example/plans/42/planCheckRun", status, error: "Check execution stopped" });
  const action = vi.fn(() => <button type="button">Help</button>);
  render(<PlanCheckResultsDrawer open onOpenChange={() => {}} includeRunFailure planCheckRuns={[stopped]} renderResultAction={action} />);
  expect(screen.getAllByRole("button", { name: "Help" })).toHaveLength(1);
  expect(action).toHaveBeenCalledWith(expect.objectContaining({ status: Advice_Level.ERROR, content: stopped.error }));
});

test.each([{ planCheckRuns: [] }, { planCheckRuns: [create(PlanCheckRunSchema, { status: PlanCheckRun_Status.RUNNING })] }])("does not request actions without result evidence", ({ planCheckRuns }) => {
  const action = vi.fn();
  render(<PlanCheckResultsDrawer open onOpenChange={() => {}} planCheckRuns={planCheckRuns} renderResultAction={action} />);
  expect(action).not.toHaveBeenCalled();
});

test("preserves refresh on open and supports missing diagnostic context", () => {
  const refresh = vi.fn(async () => {});
  render(<PlanCheckResultsDrawer open onOpenChange={() => {}} onRefreshOnOpen={refresh} planCheckRuns={[run()]} renderResultAction={() => null} />);
  expect(refresh).toHaveBeenCalledTimes(1);
  expect(screen.getByText("ERROR")).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: /Help/ })).toBeNull();
});
