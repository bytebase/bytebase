import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { ApprovalStatus, State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  RolloutSchema,
  StageSchema,
  Task_Status,
  TaskSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { PlanCheckSummary } from "../../utils/phaseSummary";
import {
  type PlanLifecycleResolverInput,
  resolvePlanLifecycleHeaderState,
} from "./planLifecycleHeaderState";

const checks = (error = 0, running = 0, success = 8): PlanCheckSummary => ({
  error,
  running,
  success,
  total: error + running + success,
  warning: 0,
});

const base: PlanLifecycleResolverInput = {
  isCreating: false,
  isGitOpsPlan: false,
  readonly: false,
  planState: State.ACTIVE,
  hasIssue: true,
  issueStatus: IssueStatus.OPEN,
  issueDraft: false,
  approvalStatus: ApprovalStatus.PENDING,
  hasCurrentStep: false,
  isCurrentUserCandidate: false,
  checks: checks(),
  hasRollout: false,
  rollout: undefined,
};

const resolve = (over: Partial<PlanLifecycleResolverInput> = {}) =>
  resolvePlanLifecycleHeaderState({ ...base, ...over });

const stage = (name: string, statuses: Task_Status[]) =>
  create(StageSchema, {
    name,
    tasks: statuses.map((status) => create(TaskSchema, { status })),
  });

const rollout = (...stages: ReturnType<typeof stage>[]) =>
  create(RolloutSchema, { stages });

describe("resolvePlanLifecycleHeaderState — draft & terminal", () => {
  test("creating", () => {
    expect(resolve({ isCreating: true }).kind).toBe("create");
  });

  test("closed plan is a terminal stamp", () => {
    expect(resolve({ planState: State.DELETED }).kind).toBe("closed");
  });

  test("a valid Draft Review Issue is ready for review", () => {
    expect(resolve({ issueDraft: true }).kind).toBe("ready-for-review");
  });

  test("draft lifecycle wins over malformed stale rollout data", () => {
    expect(
      resolve({
        issueDraft: true,
        hasRollout: true,
        rollout: rollout(stage("s1", [Task_Status.NOT_STARTED])),
      }).kind
    ).toBe("ready-for-review");
  });

  test("a persisted plan without a loadable linked issue is incomplete", () => {
    expect(resolve({ hasIssue: false }).kind).toBe("incomplete");
  });

  test("canceled review with no rollout -> closed", () => {
    expect(resolve({ issueStatus: IssueStatus.CANCELED }).kind).toBe("closed");
  });

  test("closing and reopening a draft preserves its draft lifecycle", () => {
    expect(
      resolve({
        issueDraft: true,
        planState: State.DELETED,
        issueStatus: IssueStatus.CANCELED,
      }).kind
    ).toBe("closed");
    expect(
      resolve({
        issueDraft: true,
        issueStatus: IssueStatus.OPEN,
      }).kind
    ).toBe("ready-for-review");
  });

  test("canceled review is terminal even with a rollout (not deploy)", () => {
    expect(
      resolve({
        issueStatus: IssueStatus.CANCELED,
        hasRollout: true,
        rollout: rollout(stage("s1", [Task_Status.NOT_STARTED])),
      }).kind
    ).toBe("closed");
  });

  test("gitops plan with no rollout -> none", () => {
    expect(resolve({ isGitOpsPlan: true, hasIssue: false }).kind).toBe("none");
  });
});

describe("resolvePlanLifecycleHeaderState — review", () => {
  test("approval still generating -> review-generating", () => {
    expect(resolve({ approvalStatus: ApprovalStatus.CHECKING }).kind).toBe(
      "review-generating"
    );
  });

  test("pending + current user's turn -> review-your-turn", () => {
    expect(
      resolve({ hasCurrentStep: true, isCurrentUserCandidate: true }).kind
    ).toBe("review-your-turn");
  });

  test("readonly suppresses the your-turn advance -> plan-status", () => {
    const state = resolve({
      hasCurrentStep: true,
      isCurrentUserCandidate: true,
      readonly: true,
    });
    expect(state.kind).toBe("plan-status");
  });

  test("pending + not your turn + clean checks -> in-review", () => {
    const state = resolve({});
    expect(state).toMatchObject({ kind: "plan-status", reason: "in-review" });
  });

  test("pending + checks failing -> still in-review (review is the active gate)", () => {
    const state = resolve({ checks: checks(2) });
    expect(state).toMatchObject({ kind: "plan-status", reason: "in-review" });
  });

  test("rejected -> plan-status rejected", () => {
    const state = resolve({ approvalStatus: ApprovalStatus.REJECTED });
    expect(state).toMatchObject({ kind: "plan-status", reason: "rejected" });
  });
});

describe("resolvePlanLifecycleHeaderState — pre-deploy gates", () => {
  test("approved + checks failing -> checks-failing", () => {
    const state = resolve({
      approvalStatus: ApprovalStatus.APPROVED,
      checks: checks(1),
    });
    expect(state).toMatchObject({
      kind: "plan-status",
      reason: "checks-failing",
    });
  });

  test("approved + checks still running -> checking", () => {
    const state = resolve({
      approvalStatus: ApprovalStatus.APPROVED,
      checks: checks(0, 2),
    });
    expect(state).toMatchObject({ kind: "plan-status", reason: "checking" });
  });

  test("approved + all gates passed, no rollout yet -> preparing-rollout", () => {
    expect(resolve({ approvalStatus: ApprovalStatus.APPROVED }).kind).toBe(
      "preparing-rollout"
    );
    // Skipped approval behaves the same as approved.
    expect(resolve({ approvalStatus: ApprovalStatus.SKIPPED }).kind).toBe(
      "preparing-rollout"
    );
  });

  test("done issue with approved gates and no loaded rollout -> preparing-rollout", () => {
    expect(
      resolve({
        issueStatus: IssueStatus.DONE,
        approvalStatus: ApprovalStatus.APPROVED,
      }).kind
    ).toBe("preparing-rollout");
  });
});

describe("resolvePlanLifecycleHeaderState — deploy", () => {
  const deployBase: Partial<PlanLifecycleResolverInput> = {
    hasRollout: true,
    approvalStatus: ApprovalStatus.APPROVED,
    issueStatus: IssueStatus.DONE,
  };

  test("frontier with runnable tasks -> run-stage at the frontier", () => {
    const state = resolve({
      ...deployBase,
      rollout: rollout(
        stage("stages/1", [Task_Status.DONE]),
        stage("stages/2", [Task_Status.NOT_STARTED])
      ),
    });
    expect(state.kind).toBe("run-stage");
    expect(state.kind === "run-stage" && state.stage.name).toBe("stages/2");
  });

  test("frontier with only in-progress tasks -> running-stage", () => {
    const state = resolve({
      ...deployBase,
      rollout: rollout(stage("stages/1", [Task_Status.RUNNING])),
    });
    expect(state).toMatchObject({ kind: "running-stage" });
  });

  test("every stage complete -> deployed", () => {
    expect(
      resolve({
        ...deployBase,
        rollout: rollout(
          stage("stages/1", [Task_Status.DONE]),
          stage("stages/2", [Task_Status.SKIPPED])
        ),
      }).kind
    ).toBe("deployed");
  });

  test("rollout exists but is empty -> none", () => {
    expect(resolve({ ...deployBase, rollout: rollout() }).kind).toBe("none");
  });

  test("deploy takes priority over a pending approvalStatus once a rollout exists", () => {
    // hasRollout short-circuits to deploy resolution before the review block.
    const state = resolve({
      hasRollout: true,
      approvalStatus: ApprovalStatus.PENDING,
      rollout: rollout(stage("stages/1", [Task_Status.NOT_STARTED])),
    });
    expect(state.kind).toBe("run-stage");
  });
});
