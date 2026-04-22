import { describe, expect, test } from "vitest";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  buildChangesSummary,
  buildDeploySummary,
  buildReviewSummary,
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
