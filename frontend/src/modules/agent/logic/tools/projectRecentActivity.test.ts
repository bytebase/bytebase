// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { AuditLogSchema } from "@/types/proto-es/v1/audit_log_service_pb";

const mocks = vi.hoisted(() => ({ searchAuditLogs: vi.fn() }));

vi.mock("@/api", () => ({
  auditLogServiceClientConnect: { searchAuditLogs: mocks.searchAuditLogs },
}));

import { createProjectRecentActivityTool } from "./projectRecentActivity";

describe("createProjectRecentActivityTool", () => {
  beforeEach(() => {
    mocks.searchAuditLogs.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("loads a bounded, sanitized summary from the chat-bound Project", async () => {
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: [
        create(AuditLogSchema, {
          createTime: timestampFromMs(Date.UTC(2026, 9, 20, 8)),
          actor: "users/alice@example.com",
          method: "/bytebase.v1.IssueService/CreateIssue",
          resource: "projects/orders/issues/42",
          request: '{"title":"Sensitive request payload"}',
          response: '{"private":"response payload"}',
        }),
        create(AuditLogSchema, {
          createTime: timestampFromMs(Date.UTC(2026, 9, 20, 7)),
          actor: "users/bob@example.com",
          method: "/bytebase.v1.UnknownService/UnknownMethod",
          resource: "projects/orders/resources/1",
        }),
      ],
      nextPageToken: "next-page",
    });

    const result = JSON.parse(
      await createProjectRecentActivityTool({
        projectName: "projects/orders",
      })()
    );

    expect(mocks.searchAuditLogs).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "projects/orders",
        orderBy: "create_time desc",
        pageSize: 500,
      })
    );
    expect(result).toEqual({
      project: "projects/orders",
      events: [
        {
          occurredAt: "2026-10-20T08:00:00.000Z",
          actor: "users/alice@example.com",
          action: "created an Issue",
          resource: "projects/orders/issues/42",
          outcome: "succeeded",
        },
      ],
      partial: true,
    });
    expect(JSON.stringify(result)).not.toContain("Sensitive request payload");
    expect(JSON.stringify(result)).not.toContain("response payload");
  });

  test("rejects an unbound Project without calling the audit API", async () => {
    const result = JSON.parse(await createProjectRecentActivityTool({})());

    expect(result).toEqual({
      error: "This starter is not bound to a project.",
    });
    expect(mocks.searchAuditLogs).not.toHaveBeenCalled();
  });

  test("returns at most five events from one bounded audit-log page", async () => {
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: Array.from({ length: 6 }, (_, index) =>
        create(AuditLogSchema, {
          actor: "users/alice@example.com",
          method: "/bytebase.v1.PlanService/CreatePlan",
          resource: `projects/orders/plans/${index}`,
        })
      ),
    });

    const result = JSON.parse(
      await createProjectRecentActivityTool({
        projectName: "projects/orders",
      })()
    );

    expect(result.events).toHaveLength(5);
    expect(result.partial).toBe(true);
  });

  test("includes project settings, policies, and access changes", async () => {
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: [
        create(AuditLogSchema, {
          actor: "users/alice@example.com",
          method: "/bytebase.v1.ProjectService/UpdateProject",
          resource: "projects/orders",
        }),
        create(AuditLogSchema, {
          actor: "users/admin@example.com",
          method: "/bytebase.v1.OrgPolicyService/UpdatePolicy",
          resource: "projects/orders/policies/review",
        }),
        create(AuditLogSchema, {
          actor: "users/admin@example.com",
          method: "/bytebase.v1.ProjectService/SetIamPolicy",
          resource: "projects/orders",
        }),
      ],
    });

    const result = JSON.parse(
      await createProjectRecentActivityTool({
        projectName: "projects/orders",
      })()
    );

    expect(result.events).toEqual([
      expect.objectContaining({
        action: "updated project settings",
      }),
      expect.objectContaining({
        action: "updated a policy",
      }),
      expect.objectContaining({
        action: "updated project access",
      }),
    ]);
  });

  test("searches the last three days and omits routine audit entries", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-10-20T12:00:00.000Z"));
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: [
        create(AuditLogSchema, {
          actor: "users/alice@example.com",
          method: "/bytebase.v1.SQLService/Query",
          resource: "instances/prod",
        }),
        create(AuditLogSchema, {
          actor: "users/alice@example.com",
          method: "/bytebase.v1.IssueService/ApproveIssue",
          resource: "projects/orders/issues/42",
        }),
        create(AuditLogSchema, {
          actor: "users/admin@example.com",
          method: "/bytebase.v1.SettingService/UpdateSetting",
          resource: "settings/workspace-profile",
        }),
      ],
    });

    const result = JSON.parse(
      await createProjectRecentActivityTool({
        projectName: "projects/orders",
      })()
    );

    expect(mocks.searchAuditLogs).toHaveBeenCalledWith(
      expect.objectContaining({
        filter: "create_time >= '2026-10-17T12:00:00.000Z'",
        pageSize: 500,
      })
    );
    expect(result.events).toEqual([
      expect.objectContaining({
        action: "approved an Issue",
      }),
      expect.objectContaining({
        action: "updated a workspace setting",
      }),
    ]);
  });

  test("returns a safe failure when the audit API is unavailable", async () => {
    mocks.searchAuditLogs.mockRejectedValue(new Error("permission denied"));

    const result = JSON.parse(
      await createProjectRecentActivityTool({
        projectName: "projects/orders",
      })()
    );

    expect(result).toEqual({
      error: "Unable to load recent activity.",
    });
  });
});
