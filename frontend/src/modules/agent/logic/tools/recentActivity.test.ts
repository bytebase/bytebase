// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import { AuditLogSchema } from "@/types/proto-es/v1/audit_log_service_pb";
import { PermissionDeniedDetailSchema } from "@/types/proto-es/v1/common_pb";

const mocks = vi.hoisted(() => ({ searchAuditLogs: vi.fn() }));

vi.mock("@/api", () => ({
  auditLogServiceClientConnect: { searchAuditLogs: mocks.searchAuditLogs },
}));

import { getRecentActivity } from "./recentActivity";

describe("getRecentActivity", () => {
  test.each([
    [Code.Unavailable, "unavailable"],
    [Code.InvalidArgument, "invalid_argument"],
  ] as const)(
    "retains generic code %s without exposing server diagnostics",
    async (code, wireCode) => {
      mocks.searchAuditLogs.mockRejectedValue(
        new ConnectError("private diagnostic", code)
      );
      const raw = await getRecentActivity("projects/example");
      expect(JSON.parse(raw)).toEqual({
        error: "Unable to load recent activity.",
        apiError: { source: "connect", code: wireCode },
        apiRecovery: { kind: "api-error", code, source: "api-client" },
      });
      expect(raw).not.toContain("private diagnostic");
      expect(mocks.searchAuditLogs).toHaveBeenCalledOnce();
    }
  );

  beforeEach(() => {
    mocks.searchAuditLogs.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("preserves explicit paid-feature recovery without retrying the read", async () => {
    const message =
      "feature FEATURE_AUDIT_LOG is a TEAM feature, please upgrade to access it";
    mocks.searchAuditLogs.mockRejectedValue(
      new ConnectError(message, Code.PermissionDenied)
    );

    expect(
      JSON.parse(await getRecentActivity("projects/example"))
    ).toMatchObject({
      error: message,
      apiError: { source: "connect", code: "permission_denied" },
      apiRecovery: {
        kind: "feature-required",
        feature: "FEATURE_AUDIT_LOG",
        requiredPlan: "TEAM",
      },
    });
    expect(mocks.searchAuditLogs).toHaveBeenCalledTimes(1);
  });

  test("keeps instance activation distinct from a plan upgrade", async () => {
    const message =
      'feature "FEATURE_DATA_MASKING" requires an activated instance under the current license: example';
    mocks.searchAuditLogs.mockRejectedValue(
      new ConnectError(message, Code.PermissionDenied)
    );

    expect(
      JSON.parse(await getRecentActivity("projects/example"))
    ).toMatchObject({
      error: message,
      apiError: { source: "connect", code: "permission_denied" },
      apiRecovery: {
        kind: "instance-activation-required",
        feature: "FEATURE_DATA_MASKING",
        instance: "example",
      },
    });
    expect(mocks.searchAuditLogs).toHaveBeenCalledTimes(1);
  });

  test("preserves typed missing permissions and resources", async () => {
    const error = new ConnectError(
      "Permission denied",
      Code.PermissionDenied,
      undefined,
      [
        {
          desc: PermissionDeniedDetailSchema,
          value: {
            method: "/bytebase.v1.AuditLogService/SearchAuditLogs",
            requiredPermissions: ["bb.auditLogs.search"],
            resources: ["projects/example"],
          },
        },
      ]
    );
    mocks.searchAuditLogs.mockRejectedValue(error);

    expect(
      JSON.parse(await getRecentActivity("projects/example"))
    ).toMatchObject({
      error: "Unable to load recent activity.",
      apiError: { source: "connect", code: "permission_denied" },
      apiRecovery: {
        kind: "permission-denied",
        requiredPermissions: ["bb.auditLogs.search"],
        resources: ["projects/example"],
      },
    });
    expect(mocks.searchAuditLogs).toHaveBeenCalledTimes(1);
  });

  test.each([
    new Error("Network unavailable"),
    new Error(
      "feature FEATURE_AUDIT_LOG is a TEAM feature, please upgrade to access it"
    ),
  ])(
    "retains the existing fallback for an unverified error: %s",
    async (error) => {
      mocks.searchAuditLogs.mockRejectedValue(error);

      expect(JSON.parse(await getRecentActivity("projects/example"))).toEqual({
        error: "Unable to load recent activity.",
        apiError: { source: "client" },
        apiRecovery: { kind: "api-error", source: "api-client" },
      });
      expect(mocks.searchAuditLogs).toHaveBeenCalledTimes(1);
    }
  );

  test("uses a silent bounded read and returns only sanitized activity", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-11-20T12:00:00.000Z"));
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: [
        create(AuditLogSchema, {
          createTime: timestampFromMs(Date.UTC(2026, 10, 19, 8)),
          actor: "users/member@example.com",
          method: "/bytebase.v1.PlanService/CreatePlan",
          resource: "projects/example/plans/1",
          request: '{"statement":"private statement"}',
          response: '{"data":"private response"}',
        }),
        create(AuditLogSchema, {
          method: "/bytebase.v1.SQLService/Query",
          resource: "instances/example",
        }),
      ],
      nextPageToken: "next",
    });

    expect(JSON.parse(await getRecentActivity("projects/example"))).toEqual({
      events: [
        {
          occurredAt: "2026-11-19T08:00:00.000Z",
          actor: "users/member@example.com",
          action: "created a plan",
          resource: "projects/example/plans/1",
          outcome: "succeeded",
        },
      ],
      partial: true,
    });
    expect(mocks.searchAuditLogs).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "projects/example",
        filter: "create_time >= '2026-11-17T12:00:00.000Z'",
        orderBy: "create_time desc",
        pageSize: 500,
      }),
      expect.objectContaining({ contextValues: expect.anything() })
    );
    expect(
      mocks.searchAuditLogs.mock.calls[0][1].contextValues.get(silentContextKey)
    ).toBe(true);
    expect(mocks.searchAuditLogs).toHaveBeenCalledTimes(1);
  });
});
