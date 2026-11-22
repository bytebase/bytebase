import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, expect, test, vi } from "vitest";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import { type PlanCheckRun_Result, PlanCheckRun_ResultSchema } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

const mocks = vi.hoisted(() => ({ page: {} as Record<string, unknown>, open: vi.fn() }));
vi.mock("@/api", () => ({ issueServiceClientConnect: {} }));
vi.mock("@/hooks/useAppState", () => ({ useCurrentUser: () => ({ email: "member@example.com" }) }));
vi.mock("@/hooks/usePrincipal", () => ({ usePrincipal: () => null, principalLabel: () => "Member" }));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/stores/app", () => {
  const getState = () => ({ roleList: [], workspaceResourceName: () => "workspaces/example", currentUser: { name: "users/member@example.com" } });
  return { useAppStore: Object.assign((select: (state: unknown) => unknown) => select(getState()), { getState }) };
});
vi.mock("../../shell/PlanDetailContext", () => ({ usePlanDetailContext: () => mocks.page }));
vi.mock("../../hooks/usePlanReviewState", () => ({ usePlanReviewState: () => null }));
vi.mock("../review/ReviewApprovalFlow", () => ({ deriveSteps: () => [], ReviewApprovalFlow: () => null }));
vi.mock("../review/bypassDeploy", () => ({ BypassAndDeploySheet: () => null, useBypassDeploy: () => ({ weight: "none", planCheckSummary: { total: 1, error: 1, warning: 0, running: 0, success: 0 } }) }));
vi.mock("@/components/plan-check/PlanCheckSection", () => ({ PlanCheckResultsDrawer: ({ open, renderResultAction }: { open: boolean; renderResultAction?: (result: PlanCheckRun_Result) => ReactNode }) => open ? <div data-testid="checks-drawer">{renderResultAction?.(create(PlanCheckRun_ResultSchema, { status: Advice_Level.ERROR, title: "Check failed", content: "Check execution failed" }))}</div> : null }));
vi.mock("@/modules/agent/logic/instanceConnectionHelp", () => ({ openAgentWithPrompt: (...args: unknown[]) => mocks.open(...args) }));
import { PlanStatusAction } from "./PlanStatusAction";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.page = { projectId: "example", plan: { name: "projects/example/plans/42" }, planCheckRuns: [{ name: "projects/example/plans/42/planCheckRun" }] };
});

const openChecks = () => {
  render(<PlanStatusAction issue={create(IssueSchema)} reason="checks-failing" />);
  fireEvent.click(screen.getByRole("button", { name: "Checks failing" }));
  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "View Details" }));
};

test("leaves status popover as details navigation and diagnoses only drawer evidence", () => {
  openChecks();
  expect(screen.getAllByRole("button", { name: "Ask Agent" })).toHaveLength(1);
  expect(mocks.open).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Ask Agent" }));
  const [prompt, references] = mocks.open.mock.calls[0];
  expect(references).toEqual([expect.objectContaining({ kind: "plan", name: "projects/example/plans/42", project: "projects/example" })]);
  expect(prompt).toContain('"sourceId":"projects/example/plans/42/planCheckRun"');
  expect(prompt).toContain("Check execution failed");
});

test("suppresses help when the Plan belongs to another project", () => {
  mocks.page.projectId = "other";
  openChecks();
  expect(screen.getByTestId("checks-drawer")).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();
});
