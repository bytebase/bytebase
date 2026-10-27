import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  IssueComment_ThreadState,
  IssueCommentSchema,
  IssueSchema,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import type { PlanReviewState } from "../hooks/usePlanReviewState";
import {
  type AutomatedReviewRow,
  emptyFindings,
  type PlanGate,
} from "../utils/planGates";

const mocks = vi.hoisted(() => ({
  comments: [] as unknown[],
  focusInStatement: vi.fn(),
  focusPlanPhase: vi.fn(),
  requestReviewRow: vi.fn(),
  triggerDeploy: vi.fn(),
  weight: "link" as "none" | "link" | "button",
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      options ? `${key}:${JSON.stringify(options)}` : key,
  }),
}));

vi.mock("@/lib/role", () => ({
  displayRoleTitleFromList: (role: string) => `title(${role})`,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({ roleList: [], getIssueComments: () => mocks.comments }),
}));

vi.mock("../hooks/useFocusThread", () => ({
  useFocusThread: () => ({
    focusInActivity: vi.fn(),
    focusInStatement: mocks.focusInStatement,
  }),
}));

vi.mock("../shared/stores/usePlanDetailStore", () => ({
  usePlanDetailStoreApi: () => ({
    getState: () => ({ requestReviewRow: mocks.requestReviewRow }),
  }),
}));

vi.mock("../shell/focusPhase", () => ({ focusPlanPhase: mocks.focusPlanPhase }));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({
    expandPhase: vi.fn(),
    issue: create(IssueSchema, { name: "projects/p/issues/1" }),
  }),
}));

vi.mock("./review/bypassDeploy", () => ({
  BypassAndDeploySheet: () => null,
  useBypassDeploy: () => ({
    creating: false,
    triggerDeploy: mocks.triggerDeploy,
    weight: mocks.weight,
  }),
}));

import { PlanDetailDeployWaiting } from "./PlanDetailDeployWaiting";

const openRule: AutomatedReviewRow = {
  reviewer: "rule",
  conclusion: "open",
  findings: { ...emptyFindings(), open: 2, p1: 2 },
  holdsRollout: true,
};

const renderWaiting = (
  gates: PlanGate[],
  draft = false,
  rows: AutomatedReviewRow[] = []
) =>
  render(
    <PlanDetailDeployWaiting
      issue={create(IssueSchema, { name: "projects/p/issues/1", draft })}
      plan={create(PlanSchema, { name: "projects/p/plans/1" })}
      state={{ rows, gates, openDiscussions: 0 } as unknown as PlanReviewState}
    />
  );

beforeEach(() => {
  vi.clearAllMocks();
  mocks.comments = [];
  mocks.weight = "link";
});

afterEach(cleanup);

describe("PlanDetailDeployWaiting", () => {
  test("lists what the rollout waits on, each leading to where it is settled", () => {
    mocks.comments = [
      create(IssueCommentSchema, {
        name: "discussion",
        threadState: IssueComment_ThreadState.OPEN,
      }),
    ];
    renderWaiting([
      { kind: "approval", rejected: false, role: "roles/workspaceDBA" },
      { kind: "review", row: openRule },
      { kind: "discussion", open: 1 },
    ]);
    expect(
      screen.getByText(
        'plan.deploy-waiting.approval-from:{"role":"title(roles/workspaceDBA)"}'
      )
    ).toBeDefined();
    const shows = screen.getAllByText("plan.deploy-waiting.show");
    expect(shows).toHaveLength(3);

    fireEvent.click(shows[1]);
    expect(mocks.focusPlanPhase).toHaveBeenCalledWith(
      "review",
      expect.any(Function)
    );
    expect(mocks.requestReviewRow).toHaveBeenCalledWith("rule");

    fireEvent.click(shows[2]);
    expect(mocks.focusInStatement).toHaveBeenCalledWith(
      expect.objectContaining({ name: "discussion" })
    );
  });

  test("says the rollout is being created once nothing holds it", () => {
    renderWaiting([]);
    expect(
      screen.getByText("plan.deploy-waiting.creating-rollout")
    ).toBeDefined();
  });

  test("a draft waits on submission and offers no override", () => {
    renderWaiting([{ kind: "submit" }, { kind: "review", row: openRule }], true);
    expect(screen.getByText("plan.deploy-waiting.submit")).toBeDefined();
    expect(
      screen.queryByText("plan.review.footer.bypass-and-deploy")
    ).toBeNull();
  });

  test("a submitted plan keeps the override beside the list", () => {
    renderWaiting([{ kind: "review", row: openRule }]);
    fireEvent.click(screen.getByText("plan.review.footer.bypass-and-deploy"));
    expect(mocks.triggerDeploy).toHaveBeenCalledOnce();
  });

  test("flags a P0 resolved without a change without holding the rollout", () => {
    renderWaiting([], false, [
      {
        reviewer: "rule",
        conclusion: "resolved",
        findings: { ...emptyFindings(), overridden: 1 },
        holdsRollout: false,
      },
    ]);
    expect(
      screen.getByText("plan.deploy-waiting.creating-rollout")
    ).toBeDefined();
    const line = screen.getByTestId("deploy-overridden");
    expect(line.textContent).toContain("plan.deploy-waiting.overridden");
    fireEvent.click(screen.getAllByText("plan.deploy-waiting.show")[0]);
    expect(mocks.requestReviewRow).toHaveBeenCalledWith("rule");
  });
});
