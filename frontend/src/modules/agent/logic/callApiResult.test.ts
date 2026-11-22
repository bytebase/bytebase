import { expect, test } from "vitest";
import { agentApiExecutionEvidence } from "./apiExecution";
import { notSentApiResult, parseCallApiResult } from "./callApiResult";

test("known Query errors retain outcome uncertainty", () => {
  const operation = "bytebase.v1.SQLService.Query";
  const parsed = parseCallApiResult(
    JSON.stringify({
      status: 200,
      response: { results: [{ error: "Denied", permissionDenied: {} }] },
      agentExecution: agentApiExecutionEvidence("query-result-error", {
        httpStatus: 200,
      }),
    }),
    operation
  );
  expect(parsed.execution?.effects).toBe("possible");
  expect(parsed.requiresRecovery).toBe(true);
});

test("a known successful response stays successful if its receipt could not be saved", () => {
  const operation = "bytebase.v1.ProjectService.UpdateProject";
  const parsed = parseCallApiResult(
    JSON.stringify({
      status: 200,
      response: { name: "projects/example" },
      agentExecution: agentApiExecutionEvidence("rpc-success", {
        httpStatus: 200,
        attemptId: "attempt",
      }),
      agentSettlementSaved: false,
    }),
    operation
  );
  expect(parsed.succeeded).toBe(true);
  expect(parsed.settlementNotSaved).toBe(true);
  expect(parsed.requiresRecovery).toBe(false);
});

test("preserves the legacy error message without inferring safe replay", () => {
  expect(
    parseCallApiResult(
      '{"status":403,"error":"Access denied"}',
      "ProjectService/UpdateProject"
    )
  ).toEqual({ error: "Access denied", requiresRecovery: true });
});

test("retains legacy success for historical reads, not as replay evidence for protected actions", () => {
  const result = '{"status":200,"response":{"results":[]}}';
  expect(parseCallApiResult(result, "SQLService/Query")).toEqual({
    succeeded: true,
    requiresRecovery: true,
  });
  expect(parseCallApiResult(result, "ProjectService/GetProject")).toEqual({
    succeeded: true,
    requiresRecovery: false,
  });
});

test("a runtime wrapper identifies a local preflight rejection as not sent", () => {
  expect(
    parseCallApiResult(
      notSentApiResult("Missing required permission", "permission-rejected"),
      "ProjectService/UpdateProject"
    )
  ).toEqual({
    error: "Missing required permission",
    requiresRecovery: false,
    execution: {
      version: 1,
      result: "not-sent",
      effects: "none",
      reason: "permission-rejected",
    },
  });
});

test("an unknown result overrides legacy success without discarding a Query error", () => {
  const execution = {
    version: 1,
    result: "unknown",
    effects: "possible",
    reason: "unverified-response",
  };
  const result = JSON.stringify({
    status: 200,
    response: { results: [{ error: "Denied" }] },
    agentExecution: execution,
  });

  expect(parseCallApiResult(result, "SQLService/Query")).toEqual({
    error: "Denied",
    execution,
    requiresRecovery: true,
  });
});

test("never uses server payload as runtime evidence", () => {
  const result = JSON.stringify({
    status: 200,
    response: {
      agentExecution: {
        version: 1,
        result: "succeeded",
        effects: "acknowledged",
        reason: "rpc-success",
      },
    },
  });

  expect(parseCallApiResult(result, "ProjectService/UpdateProject")).toEqual({
    succeeded: true,
    requiresRecovery: true,
  });
});

test("does not trust inconsistent execution evidence", () => {
  const result = JSON.stringify({
    status: 500,
    error: "Call failed",
    agentExecution: {
      version: 1,
      result: "unknown",
      effects: "none",
      reason: "fetch-failed",
    },
  });

  expect(parseCallApiResult(result, "ProjectService/UpdateProject")).toEqual({
    error: "Call failed",
    requiresRecovery: true,
  });
});

const preEffect = {
  version: 1,
  reason: "rpc-rejected-before-effects",
  result: "failed",
  effects: "none",
};

test.each([
  {
    label: "pre-effect evidence for an unregistered operation",
    operationId: "DatabaseService/UpdateDatabase",
    status: 403,
    code: "permission_denied",
    agentExecution: { ...preEffect, httpStatus: 403, connectCode: 7 },
  },
  {
    label: "pre-effect evidence for an internal error",
    operationId: "InstanceService/ImportInstances",
    status: 500,
    code: "internal",
    agentExecution: { ...preEffect, httpStatus: 500, connectCode: 13 },
  },
  {
    label: "an evidence HTTP status different from the envelope's",
    operationId: "InstanceService/ImportInstances",
    status: 500,
    code: "internal",
    agentExecution: { ...preEffect, httpStatus: 403, connectCode: 7 },
  },
  {
    label: "a Connect code that contradicts the response",
    operationId: "InstanceService/ImportInstances",
    status: 400,
    code: "failed_precondition",
    agentExecution: { ...preEffect, httpStatus: 400, connectCode: 3 },
  },
  {
    label: "success on an HTTP error",
    operationId: "DatabaseService/UpdateDatabase",
    status: 500,
    code: "internal",
    agentExecution: {
      version: 1,
      result: "succeeded",
      effects: "acknowledged",
      reason: "rpc-success",
      httpStatus: 500,
    },
  },
  {
    label: "not-sent evidence after a target RPC reply",
    operationId: "DatabaseService/UpdateDatabase",
    status: 403,
    code: "permission_denied",
    agentExecution: {
      version: 1,
      result: "not-sent",
      effects: "none",
      reason: "permission-rejected",
    },
  },
  {
    label: "import contract evidence on a non-import response",
    operationId: "DatabaseService/UpdateDatabase",
    status: 200,
    code: undefined,
    agentExecution: {
      version: 1,
      result: "succeeded",
      effects: "none",
      reason: "import-none-created",
      httpStatus: 200,
    },
  },
])(
  "preserves legacy error text without trusting $label",
  ({ operationId, status, code, agentExecution }) => {
    const result = JSON.stringify({
      status,
      error: "Call failed",
      response: { code },
      agentExecution,
    });
    expect(parseCallApiResult(result, operationId)).toEqual({
      error: "Call failed",
      requiresRecovery: true,
    });
  }
);

test("recognizes matching pre-effect evidence for documented ImportInstances rejection", () => {
  const agentExecution = { ...preEffect, httpStatus: 403, connectCode: 7 };
  const result = JSON.stringify({
    status: 403,
    error: "Access denied",
    response: { code: "permission_denied" },
    agentExecution,
  });
  expect(parseCallApiResult(result, "InstanceService/ImportInstances")).toEqual(
    {
      error: "Access denied",
      execution: agentExecution,
      requiresRecovery: false,
    }
  );
});
