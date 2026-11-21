import { Code, ConnectError } from "@connectrpc/connect";
import { errorToJson } from "@connectrpc/connect/protocol-connect";
import { describe, expect, it } from "vitest";
import { PermissionDeniedDetailSchema } from "@/types/proto-es/v1/common_pb";
import {
  agentApiErrorResult,
  extractDeniedQueryAccessRequest,
  getAgentApiRecovery,
  handleAgentApiResponse,
} from "./apiResponse";

const auditArgs = {
  operationId: "bytebase.v1.AuditLogService.SearchAuditLogs",
};
const paidMessage =
  "feature FEATURE_AUDIT_LOG is a TEAM feature, please upgrade to access it";
const database = "projects/sample/instances/primary/databases/main";
const queryArgs = {
  operationId: "bytebase.v1.SQLService.Query",
  body: { name: database, statement: " SELECT id FROM records;\n" },
};
const permissionFields = {
  requiredPermissions: ["bb.sql.select"],
  resources: [`${database}/schemas/public/tables/records`],
};
const queryDenial = {
  error: "Permission denied",
  permissionDenied: permissionFields,
};

const responseResult = (status: number, response: unknown) =>
  JSON.stringify({ status, response });

const permissionError = (message: string, status = 403) =>
  responseResult(status, { code: "permission_denied", message });

function typedPermissionError() {
  const error = new ConnectError("Access denied", Code.PermissionDenied);
  error.details.push({
    desc: PermissionDeniedDetailSchema,
    value: { method: "/bytebase.v1.SQLService/Query", ...permissionFields },
  });
  return error;
}

describe("Agent API recovery", () => {
  it.each([
    [Code.InvalidArgument, 400, "invalid_argument"],
    [Code.FailedPrecondition, 400, "failed_precondition"],
    [Code.Unauthenticated, 401, "unauthenticated"],
    [Code.PermissionDenied, 403, "permission_denied"],
    [Code.NotFound, 404, "not_found"],
    [Code.AlreadyExists, 409, "already_exists"],
    [Code.Aborted, 409, "aborted"],
    [Code.ResourceExhausted, 429, "resource_exhausted"],
    [Code.Unavailable, 503, "unavailable"],
    [Code.DeadlineExceeded, 504, "deadline_exceeded"],
    [Code.Canceled, 499, "canceled"],
    [Code.Internal, 500, "internal"],
  ] as const)(
    "retains a validated generic RPC failure code %s",
    (code, status, wireCode) => {
      expect(
        getAgentApiRecovery(
          auditArgs,
          responseResult(status, {
            code: wireCode,
            message: "Reported API failure",
          })
        )
      ).toEqual({ kind: "api-error", code });
    }
  );

  it("retains typed code without inventing an HTTP response or leaking diagnostics", () => {
    const error = new ConnectError(
      "credential=private-token",
      Code.Unavailable
    );
    error.details.push({
      type: "example.PrivateDetail",
      value: new TextEncoder().encode("private-detail"),
    });
    const raw = agentApiErrorResult(
      auditArgs.operationId,
      error,
      "Unable to load activity."
    );
    expect(JSON.parse(raw)).toEqual({
      error: "Unable to load activity.",
      apiError: { source: "connect", code: "unavailable" },
      apiRecovery: {
        kind: "api-error",
        code: Code.Unavailable,
        source: "api-client",
      },
    });
    expect(raw).not.toContain("private-");
  });

  it.each([
    [
      "local refresh failure",
      new ConnectError(
        "Session refresh temporarily unavailable",
        Code.Unavailable
      ),
      Code.Unavailable,
    ],
    [
      "offline fetch",
      ConnectError.from(new TypeError("Failed to fetch")),
      Code.Unknown,
    ],
    [
      "local cancellation",
      new ConnectError("Canceled locally", Code.Canceled),
      Code.Canceled,
    ],
  ])("keeps %s as client-reported evidence", (_description, error, code) => {
    const raw = agentApiErrorResult(
      undefined,
      error,
      "Unable to verify state."
    );
    const result = JSON.parse(raw);
    expect(result).not.toHaveProperty("status");
    expect(result).not.toHaveProperty("response");
    expect(getAgentApiRecovery({}, raw)).toEqual({
      kind: "api-error",
      code,
      source: "api-client",
    });
    expect(raw).not.toContain(error.rawMessage);
  });

  it.each([
    { version: 1, result: "not-sent", effects: "none", reason: "rpc-error" },
    {
      version: 1,
      result: "failed",
      effects: "possible",
      reason: "rpc-error",
      connectCode: Code.NotFound,
    },
    {
      version: 1,
      result: "failed",
      effects: "possible",
      reason: "rpc-error",
      connectCode: Code.PermissionDenied,
      httpStatus: 500,
    },
    { reason: "rpc-error" },
    {
      version: 1,
      result: "failed",
      effects: "none",
      reason: "rpc-rejected-before-effects",
      connectCode: Code.PermissionDenied,
      httpStatus: 403,
    },
  ])(
    "rejects contradictory execution evidence before specialized advice %#",
    (agentExecution) => {
      const raw = JSON.stringify({
        status: 403,
        response: { code: "permission_denied", message: paidMessage },
        agentExecution,
      });
      expect(getAgentApiRecovery(auditArgs, raw)).toBeUndefined();
      expect(handleAgentApiResponse(auditArgs, raw)).toBe(raw);
    }
  );

  it("adds paid-feature evidence without changing the execution receipt", () => {
    const envelope = {
      status: 403,
      error: paidMessage,
      response: { code: "permission_denied", message: paidMessage },
      agentExecution: {
        version: 1,
        result: "failed",
        effects: "possible",
        reason: "rpc-error",
        attemptId: "attempt-1",
        httpStatus: 403,
        connectCode: Code.PermissionDenied,
      },
    };
    expect(
      JSON.parse(handleAgentApiResponse(auditArgs, JSON.stringify(envelope)))
    ).toEqual({
      ...envelope,
      apiRecovery: {
        kind: "feature-required",
        feature: "FEATURE_AUDIT_LOG",
        requiredPlan: "TEAM",
      },
    });
  });

  it("keeps instance activation separate from plan entitlement", () => {
    const activation =
      'feature "FEATURE_INSTANCE_READ_ONLY_CONNECTION" requires an activated instance under the current license: ';
    for (const instance of ["primary", "projects/sample/instances/primary"]) {
      expect(
        getAgentApiRecovery(queryArgs, permissionError(activation + instance))
      ).toEqual({
        kind: "instance-activation-required",
        feature: "FEATURE_INSTANCE_READ_ONLY_CONNECTION",
        instance,
      });
    }
    expect(
      getAgentApiRecovery(
        queryArgs,
        permissionError(activation + "instances/-")
      )
    ).toEqual({ kind: "api-error", code: Code.PermissionDenied });
  });

  it("reports the explicit SharePoint feature without guessing a minimum plan", () => {
    expect(
      getAgentApiRecovery(
        auditArgs,
        permissionError(
          "SharePoint export requires the FEATURE_SHAREPOINT_EXPORT feature"
        )
      )
    ).toEqual({
      kind: "feature-required",
      feature: "FEATURE_SHAREPOINT_EXPORT",
    });
  });

  it("recognizes the explicit role-request workflow error", () => {
    expect(
      getAgentApiRecovery(
        { operationId: "IssueService/CreateIssue" },
        permissionError(
          "role request requires approval workflow feature (available in Enterprise plan)"
        )
      )
    ).toEqual({
      kind: "feature-required",
      feature: "FEATURE_REQUEST_ROLE_WORKFLOW",
      requiredPlan: "ENTERPRISE",
    });
  });

  it.each([
    ["ordinary permission error", permissionError("Proxy denied access")],
    [
      "unknown feature",
      permissionError(
        paidMessage.replace("FEATURE_AUDIT_LOG", "FEATURE_UNRECOGNIZED")
      ),
    ],
    [
      "unset feature",
      permissionError(
        paidMessage.replace("FEATURE_AUDIT_LOG", "FEATURE_UNSPECIFIED")
      ),
    ],
    ["unrecognized plan", permissionError(paidMessage.replace("TEAM", "FREE"))],
    [
      "similar noncanonical message",
      permissionError(`Update failed: ${paidMessage}`),
    ],
    ["mismatched status", permissionError(paidMessage, 400)],
    [
      "unverified wire response",
      JSON.stringify({
        status: 403,
        response: { code: "permission_denied", message: paidMessage },
        agentExecution: { reason: "unverified-response", result: "unknown" },
      }),
    ],
    ["malformed envelope", "not JSON"],
    [
      "untrusted annotation",
      JSON.stringify({
        status: 403,
        response: { code: "permission_denied" },
        apiRecovery: { kind: "feature-required", feature: "FEATURE_AUDIT_LOG" },
      }),
    ],
  ])("does not infer an entitlement from %s", (description, raw) => {
    const unverified = [
      "mismatched status",
      "unverified wire response",
      "malformed envelope",
    ].includes(description);
    if (unverified) {
      expect(getAgentApiRecovery(auditArgs, raw)).toBeUndefined();
      expect(handleAgentApiResponse(auditArgs, raw)).toBe(raw);
    } else {
      expect(getAgentApiRecovery(auditArgs, raw)).toEqual({
        kind: "api-error",
        code: Code.PermissionDenied,
      });
      expect(
        JSON.parse(handleAgentApiResponse(auditArgs, raw)).apiRecovery
      ).toEqual({ kind: "api-error", code: Code.PermissionDenied });
    }
  });

  it("does not treat successful row data as a license error", () => {
    const result = responseResult(200, {
      results: [{ rows: [{ values: [{ stringValue: paidMessage }] }] }],
    });
    expect(getAgentApiRecovery(queryArgs, result)).toBeUndefined();
  });

  it("reads binary permission details rather than their debug rendering", () => {
    const wire = errorToJson(typedPermissionError(), undefined);
    const result = responseResult(403, {
      ...wire,
      details: [
        {
          type: PermissionDeniedDetailSchema.typeName,
          value: "Eg1iYi5zcWwuc2VsZWN0",
          debug: { requiredPermissions: ["bb.sql.export"] },
        },
      ],
    });
    expect(getAgentApiRecovery(auditArgs, result)).toEqual({
      kind: "permission-denied",
      requiredPermissions: ["bb.sql.select"],
      resources: [],
    });
    expect(
      getAgentApiRecovery(queryArgs, responseResult(403, wire))
    ).toMatchObject({
      kind: "query-access-denied",
      request: { ...permissionFields, targets: [database] },
    });
  });

  it("does not propose a query request when one denied scope is missing", () => {
    const error = typedPermissionError();
    error.details.push({
      desc: PermissionDeniedDetailSchema,
      value: { requiredPermissions: ["bb.sql.select"], resources: [] },
    });
    const result = responseResult(403, errorToJson(error, undefined));
    expect(extractDeniedQueryAccessRequest(queryArgs, result)).toBeUndefined();
    expect(getAgentApiRecovery(queryArgs, result)).toMatchObject({
      kind: "permission-denied",
    });
  });

  it("does not propose a query request when a matching permission detail cannot be decoded", () => {
    const response = errorToJson(typedPermissionError(), undefined);
    const result = responseResult(403, {
      ...response,
      details: [
        ...(response.details as unknown[]),
        { type: PermissionDeniedDetailSchema.typeName, value: "/w" },
      ],
    });
    expect(extractDeniedQueryAccessRequest(queryArgs, result)).toBeUndefined();
    expect(getAgentApiRecovery(queryArgs, result)).toMatchObject({
      kind: "permission-denied",
    });
  });

  it("adapts typed Connect errors through the same protocol shape", () => {
    const error = typedPermissionError();
    const result = agentApiErrorResult(queryArgs.operationId, error, "Failed");
    expect(JSON.parse(result)).toEqual({
      error: "Failed",
      apiError: {
        source: "connect",
        code: "permission_denied",
        ...permissionFields,
      },
      apiRecovery: {
        kind: "permission-denied",
        ...permissionFields,
        source: "api-client",
      },
    });
    expect(
      JSON.parse(
        agentApiErrorResult(
          auditArgs.operationId,
          new Error(paidMessage),
          "Failed"
        )
      )
    ).toEqual({
      error: "Failed",
      apiError: { source: "client" },
      apiRecovery: { kind: "api-error", source: "api-client" },
    });
  });

  it.each([
    Code.Internal,
    Code.Unavailable,
    Code.PermissionDenied,
    Code.ResourceExhausted,
    Code.Unauthenticated,
  ])(
    "retains the typed-reader fallback for an unclassified Connect error (%s)",
    (code) => {
      const result = agentApiErrorResult(
        auditArgs.operationId,
        new ConnectError("internal diagnostic: private-token", code),
        "Failed"
      );
      expect(JSON.parse(result)).toMatchObject({
        error: "Failed",
        apiRecovery: { kind: "api-error", code },
      });
      expect(result).not.toContain("private-token");
    }
  );

  it("emits only validated recovery evidence from a typed permission error", () => {
    const error = new ConnectError(
      "internal diagnostic: private-token",
      Code.PermissionDenied,
      undefined,
      [
        {
          desc: PermissionDeniedDetailSchema,
          value: { method: "private-method", ...permissionFields },
        },
      ]
    );
    error.details.push({
      type: "example.PrivateDetail",
      value: new TextEncoder().encode("private-detail"),
    });
    const raw = agentApiErrorResult(auditArgs.operationId, error, "Failed");
    expect(JSON.parse(raw)).toEqual({
      error: "Failed",
      apiError: {
        source: "connect",
        code: "permission_denied",
        ...permissionFields,
      },
      apiRecovery: {
        kind: "permission-denied",
        ...permissionFields,
        source: "api-client",
      },
    });
    expect(raw).not.toContain("private-");
  });

  it("preserves not-sent execution metadata on a local permission rejection", () => {
    const envelope = {
      error: "Not called: Permission denied",
      preflightStatus: 403,
      preflight: { code: "permission_denied", ...permissionFields },
      agentExecution: {
        version: 1,
        reason: "preflight-rejected",
        result: "not-sent",
        effects: "none",
      },
    };
    const result = JSON.stringify(envelope);
    expect(JSON.parse(handleAgentApiResponse(auditArgs, result))).toEqual({
      ...envelope,
      apiRecovery: {
        kind: "permission-denied",
        ...permissionFields,
        source: "preflight",
      },
    });
    expect(getAgentApiRecovery(queryArgs, result)).toMatchObject({
      kind: "query-access-denied",
    });
  });

  it("retains the exact query privately when every statement is denied", () => {
    const result = responseResult(200, { results: [queryDenial, queryDenial] });
    expect(extractDeniedQueryAccessRequest(queryArgs, result)).toEqual({
      ...permissionFields,
      sourceDatabase: database,
      targets: [database],
      statement: queryArgs.body.statement,
    });
    expect(
      JSON.parse(handleAgentApiResponse(queryArgs, result)).apiRecovery
    ).toEqual({
      kind: "query-access-denied",
      ...permissionFields,
    });
  });

  it.each(["bb.sql.explain", "bb.sql.export", "bb.sql.execute"])(
    "explains an in-band %s denial without suggesting an Access Grant",
    (permission) => {
      const result = responseResult(200, {
        results: [
          {
            ...queryDenial,
            permissionDenied: {
              requiredPermissions: [permission],
              resources: [database],
            },
          },
        ],
      });
      expect(getAgentApiRecovery(queryArgs, result)).toEqual({
        kind: "permission-denied",
        requiredPermissions: [permission],
        resources: [database],
      });
      expect(
        extractDeniedQueryAccessRequest(queryArgs, result)
      ).toBeUndefined();
      expect(getAgentApiRecovery(auditArgs, result)).toBeUndefined();
      expect(
        getAgentApiRecovery(
          queryArgs,
          responseResult(200, {
            results: [
              {
                ...queryDenial,
                permissionDenied: {
                  requiredPermissions: [permission],
                  resources: ["not-a-resource"],
                },
              },
            ],
          })
        )
      ).toBeUndefined();
    }
  );

  it("does not recover mixed statement results", () => {
    const result = responseResult(200, {
      results: [{ rows: [] }, queryDenial],
    });
    expect(extractDeniedQueryAccessRequest(queryArgs, result)).toBeUndefined();
    expect(getAgentApiRecovery(queryArgs, result)).toBeUndefined();
  });

  it.each([
    [
      "table as source",
      {
        ...queryArgs,
        body: { ...queryArgs.body, name: permissionFields.resources[0] },
      },
    ],
    [
      "oversized statement",
      {
        ...queryArgs,
        body: { ...queryArgs.body, statement: "x".repeat(100_001) },
      },
    ],
    ["other operation", { ...queryArgs, operationId: "SQLService/Export" }],
  ])("does not recover a query with %s", (_description, args) => {
    const result = responseResult(200, { results: [queryDenial] });
    expect(extractDeniedQueryAccessRequest(args, result)).toBeUndefined();
  });

  it("does not offer query recovery for export permission", () => {
    const result = responseResult(200, {
      results: [
        {
          ...queryDenial,
          permissionDenied: {
            ...permissionFields,
            requiredPermissions: ["bb.sql.export"],
          },
        },
      ],
    });
    expect(extractDeniedQueryAccessRequest(queryArgs, result)).toBeUndefined();
  });
});
