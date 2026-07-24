import { describe, expect, test } from "vitest";
import { ApprovalStatus, State } from "@/types/proto-es/v1/common_pb";
import { type Issue, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  buildChangesSummary,
  buildDeploySummary,
  buildReviewSummary,
  isRolloutExpected,
} from "./phaseSummary";

const templates: Record<string, string> = {
  "common.and": "and",
  "plan.summary.last-approved-by": "Last approved by {{name}}",
  "plan.summary.n-changes": "{{n}} changes",
  "plan.summary.n-database-groups": "{{n}} database groups",
  "plan.summary.n-databases": "{{n}} databases",
  "plan.summary.n-error": "{{n}} error",
  "plan.summary.n-of-m-approved": "{{n}} of {{m}} approved",
  "plan.summary.n-of-m-tasks": "{{n}}/{{m}} tasks",
  "plan.summary.n-passed": "{{n}} passed",
  "plan.summary.n-warning": "{{n}} warning",
};

const t = (key: string, options?: Record<string, unknown>) => {
  const template = templates[key] ?? key;
  return Object.entries(options ?? {}).reduce(
    (text, [name, value]) => text.replaceAll(`{{${name}}}`, String(value)),
    template
  );
};

describe("plan detail phase summaries", () => {
  test("summarizes changes with targets and check counts", () => {
    const plan = {
      planCheckRunStatusCount: {
        ERROR: 1,
        FAILED: 1,
        SUCCESS: 2,
        WARNING: 1,
      },
      specs: [
        {
          config: {
            case: "changeDatabaseConfig",
            value: {
              targets: [
                "projects/p1/instances/i1/databases/db1",
                "projects/p1/databaseGroups/group1",
              ],
            },
          },
        },
        {
          config: {
            case: "changeDatabaseConfig",
            value: {
              targets: ["projects/p1/instances/i2/databases/db2"],
            },
          },
        },
      ],
    } as unknown as Plan;

    expect(buildChangesSummary(plan, t)).toBe(
      "2 changes · 2 databases and 1 database groups · 2 passed, 1 warning, 2 error"
    );
  });

  test("summarizes review approval progress and last approver", () => {
    const issue = {
      approvalTemplate: {
        flow: {
          roles: ["roles/PROJECT_OWNER", "roles/PROJECT_DBA"],
        },
      },
      approvers: [{ principal: "users/alice@example.com" }],
    } as unknown as Issue;

    expect(buildReviewSummary(issue, t)).toBe(
      "1 of 2 approved · Last approved by alice"
    );
  });

  test("summarizes deploy task progress", () => {
    const rollout = {
      stages: [
        {
          tasks: [
            { status: Task_Status.DONE },
            { status: Task_Status.SKIPPED },
            { status: Task_Status.NOT_STARTED },
          ],
        },
      ],
    } as unknown as Rollout;

    expect(buildDeploySummary(rollout, t)).toBe("2/3 tasks");
  });
});

describe("isRolloutExpected", () => {
  const approvedIssue = {
    approvalStatus: ApprovalStatus.APPROVED,
    draft: false,
    status: IssueStatus.OPEN,
  } as unknown as Issue;
  const planWithSpec = (configCase: string) =>
    ({
      planCheckRunStatusCount: { SUCCESS: 1 },
      specs: [{ config: { case: configCase, value: {} } }],
      state: State.ACTIVE,
    }) as unknown as Plan;

  test("expects automatic rollout for an approved change-database plan", () => {
    expect(
      isRolloutExpected({
        issue: approvedIssue,
        plan: planWithSpec("changeDatabaseConfig"),
      })
    ).toBe(true);
  });

  test.each([
    {
      expected: true,
      issue: {
        ...approvedIssue,
        approvalStatus: ApprovalStatus.SKIPPED,
      },
      name: "skipped approval",
    },
    {
      expected: true,
      issue: {
        ...approvedIssue,
        approvalStatus: ApprovalStatus.PENDING,
        approvalTemplate: { flow: { roles: [] } },
      },
      name: "zero approval roles",
    },
    {
      expected: true,
      issue: {
        ...approvedIssue,
        approvalStatus: ApprovalStatus.PENDING,
        status: IssueStatus.DONE,
      },
      name: "completed issue commitment",
    },
    {
      expected: false,
      issue: {
        ...approvedIssue,
        approvalStatus: ApprovalStatus.CHECKING,
      },
      name: "approval finding",
    },
    {
      expected: false,
      issue: {
        ...approvedIssue,
        approvalStatus: ApprovalStatus.PENDING,
      },
      name: "pending approval",
    },
    {
      expected: false,
      issue: {
        ...approvedIssue,
        draft: true,
      },
      name: "draft issue",
    },
    {
      expected: false,
      issue: {
        ...approvedIssue,
        status: IssueStatus.CANCELED,
      },
      name: "canceled issue",
    },
  ])("returns $expected for $name", ({ expected, issue }) => {
    expect(
      isRolloutExpected({
        issue: issue as unknown as Issue,
        plan: planWithSpec("changeDatabaseConfig"),
      })
    ).toBe(expected);
  });

  test.each([
    { expected: true, name: "no checks", statusCount: {} },
    {
      expected: true,
      name: "successful checks",
      statusCount: { SUCCESS: 2 },
    },
    {
      expected: true,
      name: "warning-only checks",
      statusCount: { WARNING: 1 },
    },
    {
      expected: false,
      name: "queued checks",
      statusCount: { AVAILABLE: 1 },
    },
    {
      expected: false,
      name: "running checks",
      statusCount: { RUNNING: 1 },
    },
    {
      expected: false,
      name: "canceled checks",
      statusCount: { CANCELED: 1 },
    },
    {
      expected: false,
      name: "advice errors",
      statusCount: { ERROR: 1 },
    },
    {
      expected: false,
      name: "failed check runs",
      statusCount: { FAILED: 1 },
    },
  ])("returns $expected with $name", ({ expected, statusCount }) => {
    const plan = {
      ...planWithSpec("changeDatabaseConfig"),
      planCheckRunStatusCount: statusCount,
    } as unknown as Plan;

    expect(isRolloutExpected({ issue: approvedIssue, plan })).toBe(expected);
  });

  test.each(["createDatabaseConfig", "exportDataConfig"])(
    "does not expect automatic rollout for %s",
    (configCase) => {
      expect(
        isRolloutExpected({
          issue: approvedIssue,
          plan: planWithSpec(configCase),
        })
      ).toBe(false);
    }
  );

  test("does not expect rollout when only some specs support it", () => {
    const plan = {
      ...planWithSpec("changeDatabaseConfig"),
      specs: [
        { config: { case: "changeDatabaseConfig", value: {} } },
        { config: { case: "exportDataConfig", value: {} } },
      ],
    } as unknown as Plan;

    expect(isRolloutExpected({ issue: approvedIssue, plan })).toBe(false);
  });

  test("does not expect rollout before an issue exists", () => {
    expect(
      isRolloutExpected({
        plan: planWithSpec("changeDatabaseConfig"),
      })
    ).toBe(false);
  });

  test("does not expect automatic rollout for empty or deleted plans", () => {
    const emptyPlan = {
      specs: [],
      state: State.ACTIVE,
    } as unknown as Plan;
    const deletedPlan = {
      ...planWithSpec("changeDatabaseConfig"),
      state: State.DELETED,
    } as unknown as Plan;

    expect(isRolloutExpected({ issue: approvedIssue, plan: emptyPlan })).toBe(
      false
    );
    expect(isRolloutExpected({ issue: approvedIssue, plan: deletedPlan })).toBe(
      false
    );
  });

  test("does not infer a rollout from a done unsupported issue", () => {
    const doneIssue = {
      ...approvedIssue,
      status: IssueStatus.DONE,
    } as unknown as Issue;

    expect(
      isRolloutExpected({
        issue: doneIssue,
        plan: planWithSpec("exportDataConfig"),
      })
    ).toBe(false);
  });

  test("keeps polling after rollout commitment regardless of eligibility", () => {
    const plan = {
      ...planWithSpec("exportDataConfig"),
      hasRollout: true,
      state: State.DELETED,
    } as unknown as Plan;

    expect(isRolloutExpected({ issue: approvedIssue, plan })).toBe(true);
  });
});
