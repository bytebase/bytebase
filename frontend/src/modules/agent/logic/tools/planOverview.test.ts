// @vitest-environment node

import { beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import {
  ApprovalStatus,
  IssueStatus,
  State,
} from "@/types/proto-es/v1/common_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

const mocks = vi.hoisted(() => ({ getPlan: vi.fn() }));

vi.mock("@/api", () => ({
  planServiceClientConnect: { getPlan: mocks.getPlan },
}));

import {
  collectPlanOverview,
  createPlanOverviewTool,
  extractPlanOverviewEvidence,
} from "./planOverview";

const projectName = "projects/orders";
const planName = `${projectName}/plans/123`;

describe("planOverview", () => {
  beforeEach(() => mocks.getPlan.mockReset());

  test("returns a bounded summary of an accessible active Plan", async () => {
    mocks.getPlan.mockResolvedValue({
      name: planName,
      state: State.ACTIVE,
      title: "Update orders",
      description: "Review before deployment",
      creator: "users/member@example.com",
      issue: `${projectName}/issues/456`,
      approvalStatus: ApprovalStatus.PENDING,
      issueStatus: IssueStatus.OPEN,
      hasRollout: true,
      planCheckRunStatusCount: { DONE: 1, FAILED: 1, WARNING: 2 },
      specs: [
        {
          id: "change-1",
          config: {
            case: "changeDatabaseConfig",
            value: {
              targets: Array.from(
                { length: 12 },
                (_, index) => `instances/main/databases/db_${index}`
              ),
              sheet: `${projectName}/sheets/1`,
              release: "",
              enablePriorBackup: true,
            },
          },
        },
        ...Array.from({ length: 10 }, (_, index) => ({
          id: `create-${index}`,
          config: {
            case: "createDatabaseConfig",
            value: {
              target: "instances/main",
              database: `new_${index}`,
              environment: "environments/test",
            },
          },
        })),
      ],
      rolloutStageSummaries: [
        {
          stage: `${planName}/rollout/stages/test`,
          taskStatusCounts: [{ status: Task_Status.FAILED, count: 2 }],
        },
      ],
    });

    const overview = await collectPlanOverview(projectName, planName);
    expect(overview).toMatchObject({
      project: projectName,
      evidence: {
        project: projectName,
        references: [{ kind: "plan", name: planName, title: "Update orders" }],
      },
      plan: {
        status: "available",
        name: planName,
        title: "Update orders",
        state: "ACTIVE",
        issue: `${projectName}/issues/456`,
        approvalStatus: "PENDING",
        issueStatus: "OPEN",
        hasRollout: true,
        planCheckRunStatusCount: { DONE: 1, FAILED: 1, WARNING: 2 },
        specCount: 11,
        specsPartial: true,
        rolloutStageSummaries: [
          {
            stage: `${planName}/rollout/stages/test`,
            taskStatusCounts: [{ status: "FAILED", count: 2 }],
          },
        ],
      },
    });
    expect(
      overview?.plan.status === "available" && overview.plan.specs
    ).toHaveLength(10);
    if (overview?.plan.status === "available") {
      expect(overview.plan.specs[0]).toMatchObject({
        type: "change_database",
        targetsPartial: true,
        sheet: `${projectName}/sheets/1`,
        targets: Array.from(
          { length: 10 },
          (_, index) => `instances/main/databases/db_${index}`
        ),
      });
    }
    expect(mocks.getPlan).toHaveBeenCalledWith(
      expect.objectContaining({ name: planName }),
      expect.objectContaining({
        contextValues: expect.objectContaining({}),
      })
    );
    expect(
      mocks.getPlan.mock.calls[0][1].contextValues.get(silentContextKey)
    ).toBe(true);
  });

  test("extracts only canonical Plan evidence from its owning project", () => {
    expect(
      extractPlanOverviewEvidence(
        JSON.stringify({
          plan: {
            status: "available",
            name: planName,
            title: "Update orders",
          },
          evidence: {
            project: projectName,
            references: [
              { kind: "plan", name: planName, title: "Update orders" },
              {
                kind: "plan",
                name: "projects/other/plans/456",
                title: "Other project",
              },
              {
                kind: "issue",
                name: `${projectName}/issues/2`,
                title: "Issue",
              },
            ],
          },
        })
      )
    ).toEqual({
      project: projectName,
      references: [{ kind: "plan", name: planName, title: "Update orders" }],
    });
    expect(
      extractPlanOverviewEvidence(
        JSON.stringify({
          plan: { status: "unavailable" },
          evidence: {
            project: projectName,
            references: [
              { kind: "plan", name: planName, title: "Update orders" },
            ],
          },
        })
      )
    ).toBeUndefined();
  });

  test("does not expose inaccessible, deleted, or mismatched Plans", async () => {
    mocks.getPlan.mockRejectedValueOnce(new Error("permission denied"));
    await expect(collectPlanOverview(projectName, planName)).resolves.toEqual({
      project: projectName,
      plan: { status: "unavailable" },
    });
    mocks.getPlan.mockResolvedValueOnce({
      name: planName,
      state: State.DELETED,
    });
    await expect(collectPlanOverview(projectName, planName)).resolves.toEqual({
      project: projectName,
      plan: { status: "unavailable" },
    });
    mocks.getPlan.mockResolvedValueOnce({
      name: `${projectName}/plans/other`,
      state: State.ACTIVE,
    });
    await expect(collectPlanOverview(projectName, planName)).resolves.toEqual({
      project: projectName,
      plan: { status: "unavailable" },
    });
  });

  test("rejects malformed names and cross-project Plan requests before API access", async () => {
    for (const invalidName of [
      "projects/other/plans/123",
      `${projectName}/plans/`,
      `${projectName}/plans/-`,
      `${projectName}/issues/123`,
    ]) {
      await expect(
        collectPlanOverview(projectName, invalidName)
      ).resolves.toBeNull();
    }
    const tool = createPlanOverviewTool({ projectName });
    expect(
      JSON.parse(await tool({ planName: "projects/other/plans/123" }))
    ).toEqual({
      error: "The Plan must belong to the project bound to this chat.",
    });
    expect(mocks.getPlan).not.toHaveBeenCalled();
  });
});
