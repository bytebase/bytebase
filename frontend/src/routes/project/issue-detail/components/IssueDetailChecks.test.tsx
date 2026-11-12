import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, expect, test, vi } from "vitest";
import { type PlanCheckRun_Result, PlanCheckRun_ResultSchema } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

const mocks = vi.hoisted(() => ({ page: {} as Record<string, unknown>, open: vi.fn(), getPlan: vi.fn(), getPlanCheckRun: vi.fn(), runPlanChecks: vi.fn() }));
vi.mock("@/api", () => ({ planServiceClientConnect: { getPlan: mocks.getPlan, getPlanCheckRun: mocks.getPlanCheckRun, runPlanChecks: mocks.runPlanChecks } }));
vi.mock("@/components/plan-check/PlanCheckSection", () => ({ PlanCheckSection: ({ renderResultAction, onRefreshOnOpen }: { renderResultAction?: (result: PlanCheckRun_Result) => ReactNode; onRefreshOnOpen: () => Promise<unknown> }) => <div><button type="button" onClick={() => void onRefreshOnOpen()}>Refresh</button>{renderResultAction?.(create(PlanCheckRun_ResultSchema, { status: Advice_Level.WARNING, title: "Rule warning", content: "Add a filter" }))}</div> }));
vi.mock("@/hooks/useAppState", () => ({ useCurrentUser: () => ({ email: "member@example.com" }) }));
vi.mock("@/hooks/useProjectByName", () => ({ useProjectByName: () => ({}) }));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/stores/app", () => {
  const getState = () => ({ projectsByName: {}, workspaceResourceName: () => "workspaces/example", currentUser: { name: "users/member@example.com" } });
  return { useAppStore: Object.assign((select: (state: unknown) => unknown) => select(getState()), { getState }) };
});
vi.mock("@/utils", () => ({ hasProjectPermissionV2: () => false }));
vi.mock("../context/IssueDetailContext", () => ({ useIssueDetailContext: () => mocks.page }));
vi.mock("@/modules/agent/logic/instanceConnectionHelp", () => ({ openAgentWithPrompt: (...args: unknown[]) => mocks.open(...args) }));
import { IssueDetailChecks } from "./IssueDetailChecks";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.page = { projectId: "example", issue: { name: "projects/example/issues/5", plan: "projects/example/plans/42" }, plan: { name: "projects/example/plans/42", creator: "users/owner@example.com" }, planCheckRuns: [{ name: "projects/example/plans/42/planCheckRun", results: [] }], patchState: vi.fn() };
});

test("binds exactly the owning Issue and linked check run on member click without fetching", () => {
  render(<IssueDetailChecks />);
  expect(mocks.getPlan).not.toHaveBeenCalled();
  expect(mocks.open).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Ask Agent" }));
  const [prompt, references] = mocks.open.mock.calls[0];
  expect(references).toEqual([expect.objectContaining({ kind: "issue", name: "projects/example/issues/5", project: "projects/example" })]);
  expect(prompt).toContain('"sourceId":"projects/example/plans/42/planCheckRun"');
  expect(prompt).toContain("Add a filter");
  expect(mocks.getPlan).not.toHaveBeenCalled();
});

test("retains the existing drawer refresh and updates loaded checks", async () => {
  const plan = { name: "projects/example/plans/42" };
  const run = { name: `${plan.name}/planCheckRun`, results: [] };
  mocks.getPlan.mockResolvedValue(plan);
  mocks.getPlanCheckRun.mockResolvedValue(run);
  render(<IssueDetailChecks />);
  fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
  await waitFor(() => expect(mocks.page.patchState).toHaveBeenCalledWith({ plan, planCheckRuns: [run] }));
  expect(mocks.getPlan).toHaveBeenCalledWith(expect.objectContaining({ name: plan.name }));
  expect(mocks.getPlanCheckRun).toHaveBeenCalledWith(expect.objectContaining({ name: run.name }));
  expect(mocks.open).not.toHaveBeenCalled();
});

test.each([
  { projectId: "other" }, { issue: undefined }, { isCreating: true },
  { issue: { name: "projects/example/issues/create", plan: "projects/example/plans/42" } },
  { issue: { name: "projects/example/issues/5", plan: "projects/other/plans/42" } },
  { issue: { name: "projects/example/issues/5", plan: "projects/example/plans/99" } },
  { planCheckRuns: [{ name: "projects/example/plans/99/planCheckRun", results: [] }] },
])("hides help for incomplete or mismatched ownership: %j", (patch) => {
  Object.assign(mocks.page, patch);
  render(<IssueDetailChecks />);
  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();
});
