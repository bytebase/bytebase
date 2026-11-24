import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  PolicySchema,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { type SchemaInfo, schemas } from "./gen/openapi-index";

const mocks = vi.hoisted(() => ({
  fetch: vi.fn(),
  getEndpointPath: vi.fn(),
  getRequestSchema: vi.fn(),
  getSchema: vi.fn(),
  refreshTokens: vi.fn(),
  getDatabase: vi.fn(),
  loadSubscription: vi.fn(),
  hasFeature: vi.fn(),
  fetchPolicyByParentAndType: vi.fn(),
  actor: "users/member@example.com",
  workspace: "workspaces/example",
}));

vi.mock("@/api", () => ({
  databaseServiceClientConnect: { getDatabase: mocks.getDatabase },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      currentUser: { name: mocks.actor },
      workspaceResourceName: () => mocks.workspace,
      loadSubscription: mocks.loadSubscription,
      hasFeature: mocks.hasFeature,
      fetchPolicyByParentAndType: mocks.fetchPolicyByParentAndType,
    }),
  },
}));

vi.mock("@/api/refreshToken", () => ({
  refreshTokens: mocks.refreshTokens,
}));

vi.mock("./apiPermissionPreflight", () => ({
  checkPreparedApiPermission: async () => ({ kind: "allowed" }),
}));

vi.mock("./searchApi", () => ({
  getEndpointPath: mocks.getEndpointPath,
  getRequestSchema: mocks.getRequestSchema,
  getSchema: mocks.getSchema,
}));

import type { AgentApiDispatchContext } from "../apiAttempt";
import { parseCallApiResult } from "../callApiResult";
import { withAgentRunLease } from "../runOwnership";
import {
  __testOnly,
  callApi as directCall,
  dispatchApprovedApiCall as directDispatch,
  prepareApiCall,
} from "./callApi";

const context: AgentApiDispatchContext = {
  scopeKey: "scope",
  userName: "users/test",
  chatId: "chat",
  toolCallId: "tool",
  logicalActionId: "approval",
  rootUserMessageId: "root",
  runId: "run",
  isCurrent: () => true,
  beginAgentApiAttempt: async () => ({
    kind: "admitted",
    attemptId: "test-attempt",
  }),
  settleAgentApiAttempt: async () => true,
};
const callApi = async (...[args, options]: Parameters<typeof directCall>) => {
  const prepared = await prepareApiCall(args);
  return prepared.kind === "error"
    ? prepared.result
    : dispatchPreparedApiCall(prepared.call, options);
};
const dispatchPreparedApiCall = async (
  ...[call, options]: Parameters<typeof directDispatch>
) =>
  (
    await withAgentRunLease("scope", "chat", () =>
      directDispatch(call, { ...options, context })
    )
  ).value!;

const createResponse = ({
  status,
  body,
  rejectJSON = false,
}: {
  status: number;
  body?: unknown;
  rejectJSON?: boolean;
}) =>
  new Response(
    rejectJSON ? "<html>invalid JSON</html>" : JSON.stringify(body),
    {
      status,
      headers: {
        "Content-Type": rejectJSON ? "text/html" : "application/json",
      },
    }
  );

const getRequestBody = (): unknown => {
  const request = mocks.fetch.mock.calls.at(-1)?.[1];
  const body = request && "body" in request ? request.body : undefined;
  return JSON.parse(typeof body === "string" ? body : "{}");
};

describe("callApi", () => {
  test("classifies a paid-feature response on direct READ dispatch without retrying", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.DatabaseGroupService/ListDatabaseGroups"
    );
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 403,
        body: {
          code: "permission_denied",
          message:
            "feature FEATURE_DATABASE_GROUPS is a ENTERPRISE feature, please upgrade to access it",
        },
      })
    );
    const result = JSON.parse(
      await directCall({
        operationId: "DatabaseGroupService/ListDatabaseGroups",
        body: { parent: "projects/example" },
      })
    );
    expect(result.apiRecovery).toEqual({
      kind: "feature-required",
      feature: "FEATURE_DATABASE_GROUPS",
      requiredPlan: "ENTERPRISE",
    });
    expect(result.agentExecution).toMatchObject({
      result: "failed",
      effects: "possible",
      reason: "rpc-error",
    });
    expect(mocks.fetch).toHaveBeenCalledOnce();
    expect(mocks.refreshTokens).not.toHaveBeenCalled();
  });

  test("uses the same recovery handler for an approved protected call", async () => {
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 403,
        body: {
          code: "permission_denied",
          message:
            "feature FEATURE_DATA_CLASSIFICATION is a ENTERPRISE feature, please upgrade to access it",
        },
      })
    );
    const result = JSON.parse(
      await dispatchPreparedApiCall({
        operationId: "bytebase.v1.SettingService.UpdateSetting",
        path: "/bytebase.v1.SettingService/UpdateSetting",
        body: '{"setting":{"name":"settings/DATA_CLASSIFICATION"}}',
      })
    );
    expect(result.apiRecovery).toEqual({
      kind: "feature-required",
      feature: "FEATURE_DATA_CLASSIFICATION",
      requiredPlan: "ENTERPRISE",
    });
    expect(result.agentExecution).toMatchObject({
      result: "failed",
      effects: "possible",
      reason: "rpc-error",
      attemptId: "test-attempt",
    });
    expect(mocks.fetch).toHaveBeenCalledOnce();
    expect(mocks.refreshTokens).not.toHaveBeenCalled();
  });

  test("retains an observed failure when the request context cannot be decoded", async () => {
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 400,
        body: { code: "invalid_argument", message: "Invalid request body" },
      })
    );
    const result = JSON.parse(
      await directDispatch({
        operationId: "bytebase.v1.SettingService.UpdateSetting",
        path: "/bytebase.v1.SettingService/UpdateSetting",
        body: "not JSON",
      })
    );
    expect(result).toMatchObject({
      status: 400,
      error: "Invalid request body",
      agentExecution: { reason: "rpc-error", result: "failed" },
    });
    expect(mocks.fetch).toHaveBeenCalledOnce();
  });

  beforeEach(() => {
    vi.stubGlobal("navigator", {
      language: "en-US",
      locks: {
        request: (
          name: string,
          _options: LockOptions,
          action: (lock: Lock) => unknown
        ) => action({ name, mode: "exclusive" }),
      },
    });
    vi.stubGlobal("fetch", mocks.fetch);
    mocks.fetch.mockReset();
    mocks.getEndpointPath.mockReset();
    mocks.getRequestSchema.mockReset();
    mocks.getSchema.mockReset();
    mocks.refreshTokens.mockReset();
    mocks.getDatabase.mockReset();
    mocks.getDatabase.mockImplementation(async ({ name }) => ({
      name,
      project: "projects/target",
    }));
    mocks.loadSubscription.mockReset().mockResolvedValue({});
    mocks.hasFeature.mockReset().mockReturnValue(true);
    mocks.fetchPolicyByParentAndType.mockReset().mockResolvedValue(null);
    mocks.getEndpointPath.mockReturnValue("/bytebase.v1.SQLService/Query");
    mocks.getRequestSchema.mockReturnValue(undefined);
    mocks.getSchema.mockReturnValue(undefined);
  });

  test.each(["scope changed", "still current"])(
    "a DOM send checks its guard and never replays a READ on 401: %s",
    async (state) => {
      const current = state === "still current";
      mocks.fetch.mockResolvedValue(createResponse({ status: 401, body: {} }));
      const result = JSON.parse(
        await dispatchPreparedApiCall(
          {
            operationId: "bytebase.v1.ProjectService.GetProject",
            path: "/bytebase.v1.ProjectService/GetProject",
            body: '{"name":"projects/test"}',
          },
          {
            dom: {
              expectedContext: {
                actor: mocks.actor,
                workspace: mocks.workspace,
              },
              beforeSend: () => current,
            },
          }
        )
      );
      expect(mocks.fetch).toHaveBeenCalledTimes(current ? 1 : 0);
      expect(result).toMatchObject(
        current ? { status: 401 } : { called: false }
      );
      expect(mocks.refreshTokens).not.toHaveBeenCalled();
    }
  );

  test.each(["denied", "unavailable"])(
    "rejects an approved request when final async validation is %s",
    async (state) => {
      mocks.fetch.mockResolvedValue(createResponse({ status: 200, body: {} }));
      const settle = vi.fn(async () => true);
      const result = JSON.parse(
        await directDispatch(
          {
            operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
            path: "/bytebase.v1.DatabaseService/UpdateDatabase",
            body: JSON.stringify({
              database: { name: "instances/example/databases/app" },
              updateMask: "labels",
            }),
          },
          {
            context: { ...context, settleAgentApiAttempt: settle },
            validateBeforeSend: async () => {
              if (state === "unavailable") throw new Error("Unavailable");
              return false;
            },
          }
        )
      );
      expect(result.agentExecution).toMatchObject({
        result: "not-sent",
        reason: "preflight-rejected",
        attemptId: "test-attempt",
      });
      expect(mocks.fetch).not.toHaveBeenCalled();
      expect(settle).toHaveBeenCalledExactlyOnceWith(
        "test-attempt",
        result.agentExecution,
        expect.any(String)
      );
    }
  );

  test.each(["canceled", "scope", "context", "DOM"])(
    "rechecks %s after awaiting final request validation",
    async (change) => {
      mocks.fetch.mockResolvedValue(createResponse({ status: 200, body: {} }));
      const controller = new AbortController();
      let current = true;
      let finishValidation!: (valid: boolean) => void;
      const validation = vi.fn(
        () =>
          new Promise<boolean>((resolve) => {
            finishValidation = resolve;
          })
      );
      const pending = directDispatch(
        {
          operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
          path: "/bytebase.v1.DatabaseService/UpdateDatabase",
          body: JSON.stringify({
            database: { name: "instances/example/databases/app" },
            updateMask: "labels",
          }),
        },
        {
          signal: controller.signal,
          context: {
            ...context,
            isCurrent: () => change !== "context" || current,
          },
          isScopeCurrent: () => change !== "scope" || current,
          validateBeforeSend: validation,
          ...(change === "DOM" && {
            dom: {
              expectedContext: {
                actor: mocks.actor,
                workspace: mocks.workspace,
              },
              beforeSend: () => current,
            },
          }),
        }
      );
      await vi.waitFor(() => expect(validation).toHaveBeenCalledOnce());
      if (change === "canceled") controller.abort();
      else current = false;
      finishValidation(true);
      expect(JSON.parse(await pending).agentExecution).toMatchObject({
        result: "not-sent",
        reason:
          change === "canceled"
            ? "canceled-before-dispatch"
            : "admission-rejected",
        attemptId: "test-attempt",
      });
      expect(mocks.fetch).not.toHaveBeenCalled();
    }
  );

  test.each(["allowed", "denied", "unavailable", "identity"])(
    "revalidates a refreshed metadata read when final scope is %s",
    async (state) => {
      mocks.getEndpointPath.mockReturnValue(
        "/bytebase.v1.ProjectService/GetProject"
      );
      mocks.fetch
        .mockResolvedValueOnce(
          createResponse({ status: 401, body: { code: "unauthenticated" } })
        )
        .mockResolvedValueOnce(
          createResponse({ status: 200, body: { name: "projects/example" } })
        );
      mocks.refreshTokens.mockResolvedValue(undefined);
      let current = true;
      const validate = vi.fn(async () => {
        if (mocks.fetch.mock.calls.length === 0) return true;
        if (state === "unavailable") throw new Error("Unavailable");
        if (state === "identity") current = false;
        return state !== "denied";
      });
      const result = JSON.parse(
        await directCall(
          { operationId: "ProjectService/GetProject" },
          {
            context: { ...context, isCurrent: () => current },
            validateBeforeSend: validate,
          }
        )
      );
      expect(validate).toHaveBeenCalledTimes(2);
      expect(mocks.fetch).toHaveBeenCalledTimes(state === "allowed" ? 2 : 1);
      expect(result).toMatchObject(
        state === "allowed"
          ? { status: 200, response: { name: "projects/example" } }
          : { status: 401, agentExecution: { result: "failed" } }
      );
    }
  );

  test("prepares an exact canonical request without sending it", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.ProjectService/UpdateProject"
    );
    const body = { project: { name: "projects/demo", title: "Before" } };

    const result = await prepareApiCall({
      operationId: "ProjectService/UpdateProject",
      body,
    });
    body.project.title = "After";

    expect(result).toEqual({
      kind: "prepared",
      call: {
        operationId: "bytebase.v1.ProjectService.UpdateProject",
        path: "/bytebase.v1.ProjectService/UpdateProject",
        body: JSON.stringify({
          project: { name: "projects/demo", title: "Before" },
        }),
      },
    });
    if (result.kind === "prepared") {
      expect(Object.isFrozen(result.call)).toBe(true);
    }
    expect(mocks.fetch).not.toHaveBeenCalled();
  });

  test("direct exported callApi cannot dispatch a protected operation without runtime context", async () => {
    const result = JSON.parse(
      await directCall({ operationId: "SQLService/Query" })
    );
    expect(result.agentExecution).toMatchObject({
      result: "not-sent",
      reason: "admission-rejected",
    });
    expect(mocks.fetch).not.toHaveBeenCalled();
  });

  test.each([
    "success",
    "receipt-start-failed",
    "receipt-settlement-failed",
    "response-lost",
    "canceled-before-dispatch",
    "canceled-after-dispatch",
  ])(
    "an approved label update needs no reconciliation read (%s)",
    async (mode) => {
      const call = {
        operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
        path: "/bytebase.v1.DatabaseService/UpdateDatabase",
        body: JSON.stringify({
          database: {
            name: "instances/example/databases/app",
            labels: { environment: "test" },
          },
          updateMask: "labels",
        }),
      };
      const controller = new AbortController();
      const begin = vi.fn(async () => {
        if (mode === "receipt-start-failed")
          throw new Error("Local receipt unavailable");
        return { kind: "admitted" as const, attemptId: "label-update" };
      });
      const settle = vi.fn(async () => mode !== "receipt-settlement-failed");
      mocks.fetch.mockImplementation(async () => {
        if (mode === "response-lost") throw new Error("Response lost");
        if (mode === "canceled-after-dispatch") {
          controller.abort();
          throw new DOMException("Request aborted", "AbortError");
        }
        return createResponse({
          status: 200,
          body: { name: "instances/example/databases/app" },
        });
      });
      if (mode === "canceled-before-dispatch") controller.abort();
      const raw = await directDispatch(call, {
        signal: controller.signal,
        context: {
          ...context,
          beginAgentApiAttempt: begin,
          settleAgentApiAttempt: settle,
        },
      });
      const result = JSON.parse(raw);
      expect(result.agentExecution.result).toBe(
        mode === "canceled-before-dispatch"
          ? "not-sent"
          : mode === "response-lost" || mode === "canceled-after-dispatch"
            ? "unknown"
            : "succeeded"
      );
      expect(mocks.getDatabase).not.toHaveBeenCalled();
      expect(mocks.fetch).toHaveBeenCalledTimes(
        mode === "canceled-before-dispatch" ? 0 : 1
      );
      if (mode !== "canceled-before-dispatch") {
        expect(mocks.fetch).toHaveBeenCalledWith(
          expect.stringContaining(call.path),
          expect.objectContaining({ method: "POST", body: call.body })
        );
        if (mode !== "receipt-start-failed")
          expect(settle).toHaveBeenCalledOnce();
      }
    }
  );

  test("an approved legacy Query sends once without Web Locks or a journal context", async () => {
    vi.stubGlobal("navigator", { language: navigator.language });
    const { createPendingApiCall, resolvePendingApiCall } = await import(
      "./pendingApiCall"
    );
    const identity = {
      chatId: "legacy",
      toolCallId: "query",
      userName: "users/member",
    };
    const call = {
      operationId: "bytebase.v1.SQLService.Query",
      path: "/bytebase.v1.SQLService/Query",
      body: JSON.stringify({
        name: "instances/example/databases/app",
        statement: "SELECT 1",
      }),
    };
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 200,
        body: {
          results: [
            {
              columnNames: ["number"],
              rows: [{ values: [{ int32Value: 1 }] }],
            },
          ],
        },
      })
    );
    const permission = createPendingApiCall({ ...identity, call });
    expect(mocks.fetch).not.toHaveBeenCalled();
    const approve = () =>
      resolvePendingApiCall({ ...identity, ...permission, approved: true });
    expect(JSON.parse(await approve())).toMatchObject({
      agentExecution: { result: "succeeded" },
      response: { results: [{ rows: [{ values: [{ int32Value: 1 }] }] }] },
    });
    expect(JSON.parse(await approve()).agentExecution.result).toBe("not-sent");
    expect(mocks.fetch).toHaveBeenCalledTimes(1);
  });

  test("bounds direct and approved canonical Query results before either caller receives them", async () => {
    const response = {
      results: [
        {
          columnNames: ["payload"],
          rows: [{ values: [{ stringValue: "x".repeat(70_000) }] }],
        },
      ],
      appliedAccessGrant: "projects/example/accessGrants/123",
    };
    mocks.fetch.mockImplementation(async () =>
      createResponse({ status: 200, body: response })
    );
    const args = {
      operationId: "SQLService/Query",
      body: { statement: "SELECT payload FROM records" },
    };

    const direct = await callApi(args);
    const prepared = await prepareApiCall(args);
    expect(prepared.kind).toBe("prepared");
    if (prepared.kind !== "prepared") return;
    const approved = await dispatchPreparedApiCall(prepared.call);

    expect(approved).toBe(direct);
    expect(JSON.parse(direct)).toEqual({
      status: 200,
      agentExecution: {
        attemptId: "test-attempt",
        version: 1,
        result: "succeeded",
        effects: "acknowledged",
        reason: "rpc-success",
        httpStatus: 200,
      },
      response: {
        ...response,
        results: [{ columnNames: ["payload"], rows: [] }],
      },
      agentQueryPreview: {
        version: 1,
        level: "sampled",
        results: [{ receivedRows: 1, retainedRows: 0 }],
      },
    });
  });

  test.each([
    ["bytebase.v1.SQLService.Export", "/bytebase.v1.SQLService/Export"],
    [
      "bytebase.v1.SQLService.AdminExecute",
      "/bytebase.v1.SQLService/AdminExecute",
    ],
    ["bytebase.v1.OtherService.Query", "/bytebase.v1.OtherService/Query"],
  ])("does not reduce non-Query result %s", async (operationId, path) => {
    const response = {
      results: [
        {
          columnNames: ["payload"],
          rows: [{ values: [{ stringValue: "x".repeat(70_000) }] }],
        },
      ],
    };
    mocks.fetch.mockResolvedValue(
      createResponse({ status: 200, body: response })
    );
    const result = await dispatchPreparedApiCall({
      operationId,
      path,
      body: "{}",
    });
    expect(JSON.parse(result).response).toEqual(response);
  });

  test.each(["SQLService/Query", "bytebase.v1.SQLService.Query"])(
    "defaults %s to 100 rows without rewriting SQL or mutating arguments",
    async (operationId) => {
      const body = {
        name: "instances/example/databases/app",
        statement: "SELECT * FROM accounts",
      };

      const result = await prepareApiCall({ operationId, body });

      expect(result).toEqual({
        kind: "prepared",
        call: {
          operationId: "bytebase.v1.SQLService.Query",
          path: "/bytebase.v1.SQLService/Query",
          body: JSON.stringify({ ...body, limit: 100 }),
        },
      });
      expect(body).not.toHaveProperty("limit");
      expect(mocks.fetch).not.toHaveBeenCalled();
    }
  );

  test.each([
    { limit: undefined, expected: 100 },
    { limit: null, expected: 100 },
    { limit: 1, expected: 1 },
    { limit: 250, expected: 250 },
    { limit: 0, expected: 0 },
  ])(
    "prepares query limit $limit as $expected",
    async ({ limit, expected }) => {
      const result = await prepareApiCall({
        operationId: "SQLService/Query",
        body: { statement: "SELECT * FROM accounts LIMIT 5", limit },
      });

      expect(result.kind).toBe("prepared");
      if (result.kind !== "prepared") return;
      expect(JSON.parse(result.call.body)).toEqual({
        statement: "SELECT * FROM accounts LIMIT 5",
        limit: expected,
      });
    }
  );

  test.each([
    { workspace: 1000, project: 500, limit: undefined, expected: 500 },
    { workspace: 30, project: 500, limit: undefined, expected: 30 },
    { workspace: 1000, project: 30, limit: null, expected: 30 },
    { workspace: 0, project: -1, limit: undefined, expected: 100 },
    { workspace: 1000, project: 300, limit: 20, expected: 20 },
    { workspace: 1000, project: 300, limit: 500, expected: 300 },
    { workspace: 1000, project: 300, limit: 0, expected: 300 },
    { workspace: 1000, project: 300, limit: -1, expected: 300 },
    { workspace: 1000, project: 300, limit: "1000", expected: 300 },
    { workspace: 1000, project: 300, limit: "0", expected: 300 },
    { workspace: 1000, project: 300, limit: "20", expected: 20 },
    { workspace: 0, project: -1, limit: 0, expected: 0 },
  ])(
    "prepares limit $limit as $expected for workspace $workspace and target project $project",
    async ({ workspace, project, limit, expected }) => {
      mocks.fetchPolicyByParentAndType.mockImplementation(
        async ({ parentPath, policyType }) => {
          if (policyType !== PolicyType.DATA_QUERY)
            throw new Error("Wrong policy");
          if (!["workspaces/example", "projects/target"].includes(parentPath)) {
            throw new Error("Wrong policy scope");
          }
          return create(PolicySchema, {
            policy: {
              case: "queryDataPolicy",
              value: {
                maximumResultRows:
                  parentPath === "workspaces/example" ? workspace : project,
              },
            },
          });
        }
      );
      const body = {
        name: "instances/example/databases/app",
        statement: "SELECT * FROM accounts LIMIT 5",
        limit,
      };

      const result = await prepareApiCall({
        operationId: "SQLService/Query",
        body,
      });

      expect(result.kind).toBe("prepared");
      if (result.kind !== "prepared") return;
      expect(JSON.parse(result.call.body)).toEqual({
        ...body,
        limit: expected,
      });
      expect(body.limit).toBe(limit);
      expect(mocks.fetch).not.toHaveBeenCalled();
    }
  );

  test.each(["", "invalid", true, {}, 2147483648, 1.5])(
    "rejects invalid query limit %j before approval",
    async (limit) => {
      const result = await prepareApiCall({
        operationId: "SQLService/Query",
        body: {
          name: "instances/example/databases/app",
          statement: "SELECT 1",
          limit,
        },
      });

      expect(result.kind).toBe("error");
      expect(mocks.fetch).not.toHaveBeenCalled();
    }
  );

  test("ignores stored policy caps when query policies are not enabled", async () => {
    mocks.hasFeature.mockReturnValue(false);
    mocks.fetchPolicyByParentAndType.mockRejectedValue(
      new Error("Must not read policies")
    );

    const result = await prepareApiCall({
      operationId: "SQLService/Query",
      body: { name: "instances/example/databases/app", statement: "SELECT 1" },
    });

    expect(result.kind).toBe("prepared");
    if (result.kind !== "prepared") return;
    expect(JSON.parse(result.call.body).limit).toBe(100);
  });

  test.each([
    "instances/example/databases/app",
    "projects/target/instances/example/databases/app",
  ])("resolves policies for database resource %s", async (name) => {
    mocks.fetchPolicyByParentAndType.mockResolvedValue(
      create(PolicySchema, {
        policy: { case: "queryDataPolicy", value: { maximumResultRows: 17 } },
      })
    );

    const result = await prepareApiCall({
      operationId: "SQLService/Query",
      body: { name, statement: "SELECT 1" },
    });

    expect(result.kind).toBe("prepared");
    if (result.kind !== "prepared") return;
    expect(JSON.parse(result.call.body)).toEqual({
      name,
      statement: "SELECT 1",
      limit: 17,
    });
  });

  test("snapshots query arguments before awaiting policy lookups", async () => {
    let resolveSubscription!: (value: unknown) => void;
    mocks.loadSubscription.mockReturnValue(
      new Promise((resolve) => {
        resolveSubscription = resolve;
      })
    );
    const body = {
      name: "instances/example/databases/app",
      statement: "SELECT 1",
    };

    const pending = prepareApiCall({ operationId: "SQLService/Query", body });
    body.name = "instances/other/databases/other";
    body.statement = "DELETE FROM accounts";
    resolveSubscription({});
    const result = await pending;

    expect(result.kind).toBe("prepared");
    if (result.kind !== "prepared") return;
    expect(JSON.parse(result.call.body)).toEqual({
      name: "instances/example/databases/app",
      statement: "SELECT 1",
      limit: 100,
    });
  });

  test.each(["subscription", "database", "policy"])(
    "does not dispatch a query when its %s lookup fails",
    async (lookup) => {
      if (lookup === "subscription")
        mocks.loadSubscription.mockResolvedValue(undefined);
      if (lookup === "database")
        mocks.getDatabase.mockRejectedValue(new Error("Unavailable"));
      if (lookup === "policy")
        mocks.fetchPolicyByParentAndType.mockResolvedValue(undefined);

      const result = JSON.parse(
        await callApi({
          operationId: "SQLService/Query",
          body: {
            name: "instances/example/databases/app",
            statement: "SELECT 1",
          },
        })
      );

      expect(result.error).toMatch(/Failed to prepare/);
      expect(result.agentExecution).toMatchObject({
        result: "not-sent",
        effects: "none",
      });
      expect(mocks.fetch).not.toHaveBeenCalled();
    }
  );

  test.each(["AdminExecute", "Export"])(
    "does not add a query limit to SQLService/%s",
    async (method) => {
      mocks.getEndpointPath.mockReturnValue(
        `/bytebase.v1.SQLService/${method}`
      );
      const body = { statement: "SELECT * FROM accounts" };

      const result = await prepareApiCall({
        operationId: `SQLService/${method}`,
        body,
      });

      expect(result.kind).toBe("prepared");
      if (result.kind !== "prepared") return;
      expect(JSON.parse(result.call.body)).toEqual(body);
    }
  );

  test("dispatches the prepared payload without resolving the operation again", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.ProjectService/UpdateProject"
    );
    const result = await prepareApiCall({
      operationId: "ProjectService/UpdateProject",
      body: { project: { name: "projects/demo", title: "Before" } },
    });
    expect(result.kind).toBe("prepared");
    if (result.kind !== "prepared") return;
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.ProjectService/DeleteProject"
    );
    mocks.fetch.mockResolvedValue(createResponse({ status: 200, body: {} }));

    await dispatchPreparedApiCall(result.call);

    expect(mocks.getEndpointPath).toHaveBeenCalledTimes(1);
    expect(mocks.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/bytebase.v1.ProjectService/UpdateProject"),
      expect.objectContaining({ body: result.call.body })
    );
  });

  test("returns preparation failures as tool errors", async () => {
    mocks.getEndpointPath.mockReturnValue(undefined);

    const result = await prepareApiCall({
      operationId: "MissingService/Write",
    });

    expect(result.kind).toBe("error");
    if (result.kind !== "error") return;
    expect(JSON.parse(result.result)).toEqual({
      error:
        "Unknown operation: MissingService/Write. Use search_api to find valid operations.",
      agentExecution: {
        version: 1,
        result: "not-sent",
        effects: "none",
        reason: "preparation-failed",
      },
    });
    expect(mocks.fetch).not.toHaveBeenCalled();
  });

  test("rejects malformed tool arguments without calling the API", async () => {
    const result = await prepareApiCall({ operationId: 42 as never });

    expect(result.kind).toBe("error");
    expect(mocks.getEndpointPath).not.toHaveBeenCalled();
    expect(mocks.fetch).not.toHaveBeenCalled();
  });

  test("returns request serialization errors as tool errors", async () => {
    const body: Record<string, unknown> = {};
    body.self = body;

    const result = await prepareApiCall({
      operationId: "SQLService/Query",
      body,
    });

    expect(result.kind).toBe("error");
    expect(mocks.fetch).not.toHaveBeenCalled();
  });

  test("never reposts a protected Query after an unauthenticated response", async () => {
    mocks.fetch
      .mockResolvedValueOnce(
        createResponse({
          status: 401,
          body: { code: "unauthenticated", message: "access token not found" },
        })
      )
      .mockRejectedValueOnce(new TypeError("second response lost"));
    mocks.refreshTokens.mockResolvedValue(undefined);
    const validateBeforeSend = vi.fn(async () => true);

    const result = JSON.parse(
      await callApi(
        {
          operationId: "SQLService/Query",
          body: { statement: "SELECT 1" },
        },
        { validateBeforeSend }
      )
    );

    expect(result).toMatchObject({
      status: 401,
      response: { code: "unauthenticated" },
      agentExecution: { result: "failed", effects: "possible" },
    });
    expect(mocks.fetch).toHaveBeenCalledTimes(1);
    expect(validateBeforeSend).toHaveBeenCalledOnce();
    expect(mocks.refreshTokens).not.toHaveBeenCalled();
    expect(mocks.fetch).toHaveBeenNthCalledWith(
      1,
      expect.stringContaining("/bytebase.v1.SQLService/Query"),
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        redirect: "manual",
        headers: {
          "Content-Type": "application/json",
          "Connect-Protocol-Version": "1",
        },
        body: JSON.stringify({ statement: "SELECT 1", limit: 100 }),
      })
    );
  });

  test("does not recurse on the refresh endpoint", async () => {
    mocks.getEndpointPath.mockReturnValue("/bytebase.v1.AuthService/Refresh");
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 401,
        body: { message: "refresh token expired" },
      })
    );

    const result = JSON.parse(
      await callApi({
        operationId: "AuthService/Refresh",
      })
    );

    expect(result).toMatchObject({
      status: 401,
      error: "refresh token expired",
      response: { message: "refresh token expired" },
    });
    expect(mocks.refreshTokens).not.toHaveBeenCalled();
    expect(mocks.fetch).toHaveBeenCalledTimes(1);
  });

  test("preserves the original error shape when refresh fails", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.ProjectService/GetProject"
    );
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 401,
        body: { code: "unauthenticated", message: "access token not found" },
      })
    );
    mocks.refreshTokens.mockRejectedValue(new Error("refresh failed"));

    const result = JSON.parse(
      await callApi({
        operationId: "ProjectService/GetProject",
      })
    );

    expect(result).toMatchObject({
      status: 401,
      error: "access token not found",
      response: { code: "unauthenticated", message: "access token not found" },
      agentExecution: { result: "failed", reason: "rpc-error" },
    });
    expect(mocks.refreshTokens).toHaveBeenCalledTimes(1);
    expect(mocks.fetch).toHaveBeenCalledTimes(1);
  });

  test("keeps the metadata read's second response after token refresh", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.ProjectService/GetProject"
    );
    mocks.fetch
      .mockResolvedValueOnce(
        createResponse({ status: 401, body: { code: "unauthenticated" } })
      )
      .mockResolvedValueOnce(
        createResponse({ status: 200, body: { name: "projects/example" } })
      );
    mocks.refreshTokens.mockResolvedValue(undefined);

    const result = JSON.parse(
      await callApi({ operationId: "ProjectService/GetProject" })
    );

    expect(result).toMatchObject({
      status: 200,
      response: { name: "projects/example" },
    });
    expect(mocks.refreshTokens).toHaveBeenCalledOnce();
    expect(mocks.fetch).toHaveBeenCalledTimes(2);
  });

  test("a metadata read lost after refresh never reports the first 401", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.ProjectService/GetProject"
    );
    mocks.fetch
      .mockResolvedValueOnce(
        createResponse({ status: 401, body: { code: "unauthenticated" } })
      )
      .mockRejectedValueOnce(new TypeError("second response lost"));
    mocks.refreshTokens.mockResolvedValue(undefined);

    const result = JSON.parse(
      await callApi({ operationId: "ProjectService/GetProject" })
    );

    expect(result).not.toHaveProperty("status", 401);
    expect(result.agentExecution).toMatchObject({
      result: "unknown",
      reason: "fetch-failed",
    });
    expect(mocks.refreshTokens).toHaveBeenCalledOnce();
    expect(mocks.fetch).toHaveBeenCalledTimes(2);
  });

  test("can stop waiting for refresh without sending a second metadata read", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.ProjectService/GetProject"
    );
    mocks.fetch.mockResolvedValueOnce(
      createResponse({ status: 401, body: { code: "unauthenticated" } })
    );
    let finishRefresh!: () => void;
    let startRefresh!: () => void;
    const refreshing = new Promise<void>((resolve) => {
      startRefresh = resolve;
    });
    mocks.refreshTokens.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          finishRefresh = resolve;
          startRefresh();
        })
    );
    const controller = new AbortController();
    const pending = callApi(
      { operationId: "ProjectService/GetProject" },
      { signal: controller.signal }
    );
    await refreshing;
    controller.abort();

    const result = await Promise.race([
      pending.then((raw) => JSON.parse(raw)),
      new Promise<null>((resolve) => setTimeout(() => resolve(null), 0)),
    ]);
    finishRefresh();
    await pending;

    expect(result).toMatchObject({
      status: 401,
      agentExecution: { result: "failed" },
    });
    expect(mocks.fetch).toHaveBeenCalledOnce();
  });

  test.each([
    new TypeError("connection lost"),
    new DOMException("aborted", "AbortError"),
  ])("fetch_rejection_after_invocation_is_unknown (%s)", async (error) => {
    mocks.fetch.mockRejectedValueOnce(error);
    const result = JSON.parse(
      await callApi({ operationId: "SQLService/Query" })
    );

    expect(mocks.fetch).toHaveBeenCalledOnce();
    expect(result.agentExecution).toMatchObject({
      result: "unknown",
      effects: "possible",
      reason: "fetch-failed",
    });
  });

  test("malformed_200_is_not_success", async () => {
    mocks.fetch.mockResolvedValueOnce(
      createResponse({ status: 200, rejectJSON: true })
    );
    const result = await callApi({ operationId: "SQLService/Query" });

    expect(parseCallApiResult(result, "SQLService/Query")).toMatchObject({
      requiresRecovery: true,
      execution: {
        result: "unknown",
        effects: "possible",
        reason: "response-body-unreadable",
      },
    });
    expect(parseCallApiResult(result, "SQLService/Query").succeeded).not.toBe(
      true
    );
  });

  test.each([null, "text/plain"])(
    "keeps the ImportInstances error without trusting non-Connect media type %s",
    async (contentType) => {
      mocks.getEndpointPath.mockReturnValue(
        "/bytebase.v1.InstanceService/ImportInstances"
      );
      const response = new Response(
        '{"code":"permission_denied","message":"Proxy denied access"}',
        { status: 403 }
      );
      if (contentType) response.headers.set("Content-Type", contentType);
      else response.headers.delete("Content-Type");
      mocks.fetch.mockResolvedValueOnce(response);

      const result = await callApi({
        operationId: "InstanceService/ImportInstances",
      });

      expect(JSON.parse(result)).toMatchObject({
        status: 403,
        response: { code: "permission_denied", message: "Proxy denied access" },
      });
      expect(
        parseCallApiResult(result, "InstanceService/ImportInstances")
      ).toMatchObject({
        error: "Proxy denied access",
        requiresRecovery: true,
        execution: { result: "unknown", effects: "possible" },
      });
    }
  );

  test("a redirect is not followed or interpreted as a successful reply", async () => {
    mocks.fetch.mockResolvedValueOnce(
      new Response(null, { status: 307, headers: { Location: "/redirect" } })
    );

    const result = JSON.parse(
      await callApi({ operationId: "SQLService/Query" })
    );

    expect(mocks.fetch).toHaveBeenCalledOnce();
    expect(mocks.fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ redirect: "manual" })
    );
    expect(result.agentExecution).toMatchObject({
      result: "unknown",
      effects: "possible",
      reason: "unexpected-redirect",
    });
  });

  const preparedQuery = {
    operationId: "bytebase.v1.SQLService.Query",
    path: "/bytebase.v1.SQLService/Query",
    body: '{"statement":"SELECT 1"}',
  };

  test("cancellation before invocation is not sent", async () => {
    const result = JSON.parse(
      await dispatchPreparedApiCall(preparedQuery, {
        signal: AbortSignal.abort(),
        attemptId: "test-attempt",
      })
    );

    expect(mocks.fetch).not.toHaveBeenCalled();
    expect(result.agentExecution).toMatchObject({
      result: "not-sent",
      effects: "none",
      attemptId: "test-attempt",
    });
  });

  test("caller cancellation reaches fetch without proving no effect", async () => {
    const controller = new AbortController();
    mocks.fetch.mockImplementationOnce(
      (_, { signal }) =>
        new Promise((_, reject) => {
          signal.addEventListener("abort", () => reject(signal.reason), {
            once: true,
          });
        })
    );
    const pending = dispatchPreparedApiCall(preparedQuery, {
      signal: controller.signal,
    });
    await vi.waitFor(() => expect(mocks.fetch).toHaveBeenCalledOnce());
    controller.abort();

    const result = JSON.parse(await pending);

    expect(mocks.fetch).toHaveBeenCalledOnce();
    expect(result.agentExecution).toMatchObject({
      result: "unknown",
      effects: "possible",
      reason: "canceled-after-dispatch",
    });
  });

  test("the dispatch timeout remains active after composing the caller signal", async () => {
    vi.useFakeTimers();
    try {
      mocks.fetch.mockImplementationOnce(
        (_, { signal }) =>
          new Promise((_, reject) => {
            signal.addEventListener("abort", () => reject(signal.reason), {
              once: true,
            });
          })
      );
      const pending = dispatchPreparedApiCall(preparedQuery, {
        signal: new AbortController().signal,
      });
      await vi.waitFor(() => expect(mocks.fetch).toHaveBeenCalledOnce());
      await vi.advanceTimersByTimeAsync(30_000);

      const result = JSON.parse(await pending);

      expect(result.agentExecution).toMatchObject({
        result: "unknown",
        effects: "possible",
        reason: "timed-out",
      });
      expect(vi.getTimerCount()).toBe(0);
    } finally {
      vi.useRealTimers();
    }
  });

  test("preserves non-json error handling", async () => {
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 500,
        rejectJSON: true,
      })
    );

    const result = JSON.parse(
      await callApi({
        operationId: "SQLService/Query",
      })
    );

    expect(result).toMatchObject({
      status: 500,
      error: "HTTP 500",
      response: null,
    });
    expect(mocks.refreshTokens).not.toHaveBeenCalled();
  });

  test("coerces nested byte fields to base64 before sending the request", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.SavedQueryService/CreateSavedQuery"
    );
    mocks.getRequestSchema.mockReturnValue(
      schemas["bytebase.v1.CreateSavedQueryRequest"]
    );
    mocks.getSchema.mockImplementation(
      (schemaName: string) => schemas[schemaName]
    );
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 200,
        body: { name: "projects/demo/savedQueries/1" },
      })
    );

    await callApi({
      operationId: "SavedQueryService/CreateSavedQuery",
      body: {
        parent: "projects/demo",
        savedQuery: {
          content: "select 1;",
          title: "demo",
        },
      },
    });

    expect(getRequestBody()).toEqual({
      parent: "projects/demo",
      savedQuery: {
        content: "c2VsZWN0IDE7",
        title: "demo",
      },
    });
  });

  test("coerces byte fields inside arrays using the real batch create sheets schema", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.SheetService/BatchCreateSheets"
    );
    mocks.getRequestSchema.mockReturnValue(
      schemas["bytebase.v1.BatchCreateSheetsRequest"]
    );
    mocks.getSchema.mockImplementation(
      (schemaName: string) => schemas[schemaName]
    );
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 200,
        body: { sheets: [] },
      })
    );

    await callApi({
      operationId: "SheetService/BatchCreateSheets",
      body: {
        parent: "projects/demo",
        requests: [
          {
            parent: "projects/demo",
            sheet: {
              content: "select 1;",
              title: "sheet one",
            },
          },
          {
            parent: "projects/demo",
            sheet: {
              content: "select 2;",
              title: "sheet two",
            },
          },
        ],
      },
    });

    expect(getRequestBody()).toEqual({
      parent: "projects/demo",
      requests: [
        {
          parent: "projects/demo",
          sheet: {
            content: "c2VsZWN0IDE7",
            title: "sheet one",
          },
        },
        {
          parent: "projects/demo",
          sheet: {
            content: "c2VsZWN0IDI7",
            title: "sheet two",
          },
        },
      ],
    });
  });

  test("coerces byte fields through external oneof schemas in batch deparse requests", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.CelService/BatchDeparse"
    );
    mocks.getRequestSchema.mockReturnValue(
      schemas["bytebase.v1.BatchDeparseRequest"]
    );
    mocks.getSchema.mockImplementation(
      (schemaName: string) => schemas[schemaName]
    );
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 200,
        body: { expressions: [] },
      })
    );

    await callApi({
      operationId: "CelService/BatchDeparse",
      body: {
        expressions: [
          {
            id: "1",
            constExpr: {
              bytesValue: "hello",
            },
          },
        ],
      },
    });

    expect(getRequestBody()).toEqual({
      expressions: [
        {
          id: "1",
          constExpr: {
            bytesValue: "aGVsbG8=",
          },
        },
      ],
    });
  });

  test("coerces byte fields through top-level oneof sasl config branches", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.InstanceService/AddDataSource"
    );
    mocks.getRequestSchema.mockReturnValue(
      schemas["bytebase.v1.AddDataSourceRequest"]
    );
    mocks.getSchema.mockImplementation(
      (schemaName: string) => schemas[schemaName]
    );
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 200,
        body: { name: "instances/demo/dataSources/readonly" },
      })
    );

    await callApi({
      operationId: "InstanceService/AddDataSource",
      body: {
        name: "instances/demo",
        dataSource: {
          saslConfig: {
            krbConfig: {
              primary: "postgres",
              realm: "EXAMPLE.COM",
              keytab: "keytab-bytes",
            },
          },
        },
      },
    });

    expect(getRequestBody()).toEqual({
      name: "instances/demo",
      dataSource: {
        saslConfig: {
          krbConfig: {
            primary: "postgres",
            realm: "EXAMPLE.COM",
            keytab: "a2V5dGFiLWJ5dGVz",
          },
        },
      },
    });
  });

  test("coerces byte fields inside map values through the helper path", () => {
    const schema: SchemaInfo = {
      type: "object",
      description: "test schema for map coercion",
      properties: [
        {
          name: "entries",
          type: "object",
          additionalProperties: {
            name: "value",
            type: "test.Entry",
          },
        },
      ],
    };

    mocks.getSchema.mockImplementation((schemaName: string) => {
      if (schemaName === "test.Entry") {
        return {
          type: "object",
          description: "test entry schema",
          properties: [
            {
              name: "content",
              type: "string",
              format: "byte",
            },
          ],
        };
      }
      return schemas[schemaName];
    });

    expect(
      __testOnly.coerceRequestBody(
        {
          entries: {
            alpha: { content: "select 1;" },
            beta: { content: "select 2;" },
          },
        },
        schema
      )
    ).toEqual({
      entries: {
        alpha: { content: "c2VsZWN0IDE7" },
        beta: { content: "c2VsZWN0IDI7" },
      },
    });
  });
});
