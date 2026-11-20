import { Code } from "@connectrpc/connect";
import { describe, expect, test } from "vitest";
import {
  agentApiNeedsRecovery,
  classifyAgentApiResponse,
} from "./apiExecution";
import type { PreparedApiCall } from "./tools/callApi";

const projectDelete = {
  operationId: "bytebase.v1.ProjectService.DeleteProject",
  path: "/bytebase.v1.ProjectService/DeleteProject",
  body: '{"name":"projects/example"}',
};

const batchUpdate = {
  operationId: "bytebase.v1.DatabaseService.BatchUpdateDatabases",
  path: "/bytebase.v1.DatabaseService/BatchUpdateDatabases",
  body: "{}",
};

const databaseUpdate = {
  operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
  path: "/bytebase.v1.DatabaseService/UpdateDatabase",
  body: '{"database":{"name":"instances/example/databases/app"},"updateMask":"labels"}',
};

const importCall = {
  operationId: "bytebase.v1.InstanceService.ImportInstances",
  path: "/bytebase.v1.InstanceService/ImportInstances",
  body: '{"imports":[{"instanceId":"first"},{"instanceId":"second"}]}',
};

const query = {
  operationId: "bytebase.v1.SQLService.Query",
  path: "/bytebase.v1.SQLService/Query",
  body: '{"statement":"UPDATE records SET state = 1; SELECT 1"}',
};

function response(
  call: PreparedApiCall,
  body: unknown,
  status = 200,
  contentType: string | null = "application/json"
) {
  return classifyAgentApiResponse(call, {
    kind: "response",
    status,
    body: JSON.stringify(body),
    contentType,
  });
}

describe("Agent API execution evidence", () => {
  test("valid_empty_reply_is_success", () => {
    expect(response(projectDelete, {})).toEqual({
      version: 1,
      result: "succeeded",
      effects: "acknowledged",
      reason: "rpc-success",
      httpStatus: 200,
    });
  });

  test("acknowledges a successful email update that renames its resource", () => {
    expect(
      response(
        {
          operationId: "bytebase.v1.UserService.UpdateEmail",
          path: "/bytebase.v1.UserService/UpdateEmail",
          body: JSON.stringify({
            name: "users/old@example.com",
            email: "new@example.com",
          }),
        },
        { name: "users/new@example.com", email: "new@example.com" }
      )
    ).toMatchObject({
      result: "succeeded",
      effects: "acknowledged",
      reason: "rpc-success",
    });
  });

  test("acknowledges valid protobuf defaults without inferring business effects", () => {
    expect(response(databaseUpdate, { labels: {} })).toMatchObject({
      result: "succeeded",
      effects: "acknowledged",
    });
    expect(
      response(databaseUpdate, {
        name: "instances/example/databases/app",
        labels: {},
      })
    ).toMatchObject({ result: "succeeded", effects: "acknowledged" });
  });

  test("supports a resource with a server-assigned name", () => {
    expect(
      response(
        {
          operationId: "bytebase.v1.IssueService.CreateIssue",
          path: "/bytebase.v1.IssueService/CreateIssue",
          body: '{"parent":"projects/example","issue":{"title":"Draft change"}}',
        },
        { name: "projects/example/issues/123" }
      )
    ).toMatchObject({ result: "succeeded", effects: "acknowledged" });
  });

  test("does not acknowledge a resource response that fails protobuf decoding", () => {
    expect(response(databaseUpdate, { labels: [] })).toMatchObject({
      result: "unknown",
      effects: "possible",
    });
  });

  test.each([null, "text/plain", "application/connect+json"])(
    "does not acknowledge Empty or pre-creation errors without unary Connect JSON (%s)",
    (contentType) => {
      expect(response(projectDelete, {}, 200, contentType)).toMatchObject({
        result: "unknown",
        effects: "possible",
      });
      expect(
        response(importCall, { code: "permission_denied" }, 403, contentType)
      ).toMatchObject({ result: "unknown", effects: "possible" });
    }
  );

  test("accepts a UTF-8 charset on unary Connect JSON", () => {
    expect(
      response(projectDelete, {}, 200, "application/json; charset=UTF-8")
    ).toMatchObject({ result: "succeeded", effects: "acknowledged" });
    expect(
      response(
        importCall,
        { code: "permission_denied" },
        403,
        "application/json; charset=UTF-8"
      )
    ).toMatchObject({ result: "failed", effects: "none" });
  });

  test.each([
    null,
    "",
    [],
    { fieldFromFutureServer: true },
    { error: "not an Empty" },
  ])("does not accept a mismatched Empty payload %j", (body) => {
    expect(response(projectDelete, body)).toMatchObject({
      result: "unknown",
      effects: "possible",
    });
  });

  test("does not use the descriptor for an operation that disagrees with the dispatched path", () => {
    expect(
      response(
        { ...projectDelete, path: "/bytebase.v1.ProjectService/UpdateProject" },
        {}
      )
    ).toMatchObject({ result: "unknown", effects: "possible" });
  });

  test("an unsupported operation does not accept a successful payload", () => {
    expect(
      response({ ...projectDelete, operationId: "UnknownService/Write" }, {})
    ).toMatchObject({ result: "unknown", effects: "possible" });
  });

  test.each([
    "bytebase.v1.SQLService.Query",
    "SettingService/GetSetting",
    "UnclassifiedService/Operation",
  ])("requires recovery for uncertain %s", (operationId) => {
    expect(
      agentApiNeedsRecovery(operationId, {
        version: 1,
        result: "unknown",
        effects: "possible",
        reason: "fetch-failed",
      })
    ).toBe(true);
  });

  test("does not require write recovery after losing an ordinary metadata read", () => {
    expect(
      agentApiNeedsRecovery("ProjectService/GetProject", {
        version: 1,
        result: "unknown",
        effects: "possible",
        reason: "fetch-failed",
      })
    ).toBe(false);
  });

  test("batch_error_may_have_partial_effects", () => {
    expect(
      response(
        batchUpdate,
        {
          code: "permission_denied",
          message: "denied on a later resource",
        },
        403
      )
    ).toEqual({
      version: 1,
      result: "failed",
      effects: "possible",
      reason: "rpc-error",
      httpStatus: 403,
      connectCode: Code.PermissionDenied,
    });
  });

  test.each([
    ["invalid_argument", 400],
    ["unauthenticated", 401],
    ["permission_denied", 403],
    ["not_found", 404],
  ])(
    "recognizes only the documented ImportInstances %s pre-creation error",
    (code, status) => {
      const body = { code, message: "request rejected" };
      expect(response(importCall, body, status)).toMatchObject({
        result: "failed",
        effects: "none",
      });
      expect(response(batchUpdate, body, status)).toMatchObject({
        result: "failed",
        effects: "possible",
      });
    }
  );

  test("an unexpected ImportInstances RPC error remains potentially effectful", () => {
    expect(response(importCall, { code: "internal" }, 500)).toMatchObject({
      result: "failed",
      effects: "possible",
    });
  });

  test.each([
    [{ message: "access denied" }, 403],
    [{ code: "permission_denied", details: [{ type: "invalid" }] }, 403],
    [{ code: "permission_denied" }, 500],
    [{ code: "toString" }, 403],
  ])(
    "cannot prove no creation from an unverified RPC error %j",
    (body, status) => {
      expect(response(importCall, body, status)).toMatchObject({
        result: "unknown",
        effects: "possible",
      });
    }
  );

  test("an explicit UNKNOWN requires recovery even without a confirmed creation", () => {
    const evidence = response(importCall, {
      results: [
        { failure: { status: { code: 13 }, creationOutcome: "UNKNOWN" } },
        { failure: { status: { code: 6 }, creationOutcome: "NOT_CREATED" } },
      ],
    });

    expect(evidence).toMatchObject({
      result: "succeeded",
      effects: "possible",
    });
    expect(agentApiNeedsRecovery(importCall.operationId, evidence)).toBe(true);
  });

  test.each(["NOT_CREATED", "UNKNOWN"])(
    "a confirmed import plus %s has partial effects",
    (creationOutcome) => {
      const evidence = response(importCall, {
        results: [
          { imported: { name: "instances/first" } },
          { failure: { status: { code: 13 }, creationOutcome } },
        ],
      });

      expect(evidence).toMatchObject({
        result: "succeeded",
        effects: "partial",
      });
      expect(agentApiNeedsRecovery(importCall.operationId, evidence)).toBe(
        true
      );
    }
  );

  test("all explicit NOT_CREATED import results prove none", () => {
    const evidence = response(importCall, {
      results: [
        { failure: { status: { code: 13 }, creationOutcome: "NOT_CREATED" } },
        { failure: { status: { code: 6 }, creationOutcome: "NOT_CREATED" } },
      ],
    });

    expect(evidence).toMatchObject({ result: "succeeded", effects: "none" });
    expect(agentApiNeedsRecovery(importCall.operationId, evidence)).toBe(false);
  });

  test("every explicitly confirmed registration is acknowledged", () => {
    expect(
      response(importCall, {
        results: [
          { imported: { name: "instances/first" } },
          { imported: { name: "instances/second" } },
        ],
      })
    ).toMatchObject({ result: "succeeded", effects: "acknowledged" });
  });

  test.each(
    [
      [],
      [{ imported: { name: "instances/first" } }],
      [{ imported: { name: "instances/first" } }, {}],
      [{ imported: { name: "instances/first" } }, { imported: {} }],
      [
        { imported: { name: "instances/first" } },
        { imported: { name: "instances/another" } },
      ],
      [
        { imported: { name: "instances/first" } },
        { failure: { creationOutcome: "NOT_CREATED" } },
      ],
      [
        { imported: { name: "instances/first" } },
        { failure: { status: { code: 13 }, creationOutcome: 99 } },
      ],
    ].map((results) => ({ results }))
  )(
    "does not trust an incomplete or unrecognized import reply: $results",
    ({ results }) => {
      expect(response(importCall, { results })).toMatchObject({
        result: "unknown",
        effects: "possible",
      });
    }
  );

  test("sql_denial_does_not_clear_prior_statement_uncertainty", () => {
    const evidence = response(query, {
      results: [
        { statement: "UPDATE records SET state = 1", rowsCount: "1" },
        {
          statement: "SELECT 1",
          permissionDenied: { requiredPermissions: ["bb.sql.select"] },
        },
      ],
    });

    expect(evidence).toMatchObject({
      result: "succeeded",
      effects: "possible",
      reason: "query-result-error",
    });
    expect(agentApiNeedsRecovery(query.operationId, evidence)).toBe(true);
  });

  test("an empty Query response may encode zero statements", () => {
    expect(response(query, {})).toMatchObject({
      result: "succeeded",
      effects: "acknowledged",
    });
  });
});
