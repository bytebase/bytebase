// @vitest-environment node
import { describe, expect, test } from "vitest";
import { getAgentApiRecoveryGuidance } from "./apiResponseGuidance";
import type { Message } from "./types";

const args = {
  operationId: "SettingService/UpdateSetting",
  body: { setting: { name: "settings/DATA_CLASSIFICATION" } },
};
const paid = JSON.stringify({
  status: 403,
  response: {
    code: "permission_denied",
    message:
      "feature FEATURE_DATA_CLASSIFICATION is a ENTERPRISE feature, please upgrade to access it",
  },
});
const observation = (
  result: string,
  id = "api",
  request: Record<string, unknown> = args,
  name = "call_api"
): Message[] => [
  {
    role: "assistant",
    toolCalls: [{ id, name, arguments: JSON.stringify(request) }],
  },
  { role: "tool", toolCallId: id, content: result },
];

describe("getAgentApiRecoveryGuidance", () => {
  const unknown = JSON.stringify({
    error: "Request timed out",
    agentExecution: {
      version: 1,
      result: "unknown",
      effects: "possible",
      reason: "timed-out",
    },
  });
  const notSent = JSON.stringify({
    error: "Permission denied",
    agentExecution: {
      version: 1,
      result: "not-sent",
      effects: "none",
      reason: "permission-rejected",
    },
  });

  test.each([
    ["declined", notSent],
    ["successful", JSON.stringify({ status: 200, response: {} })],
  ])(
    "preserves an earlier uncertain attempt after a later %s attempt",
    (_, later) => {
      const guidance = getAgentApiRecoveryGuidance([
        { role: "user" },
        ...observation(unknown),
        ...observation(later),
      ]);
      expect(guidance).toContain("outcome could not be verified");
      expect(guidance).toContain("read-only verification");
    }
  );

  test("preserves an earlier uncertain attempt while a later attempt is unfinished", () => {
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user" },
      ...observation(unknown),
      observation("unused")[0],
    ]);
    expect(guidance).toContain("outcome could not be verified");
  });

  test("does not retire a failed response after a later declined attempt", () => {
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user" },
      ...observation(paid),
      ...observation(notSent),
    ]);
    expect(guidance).toContain("paid-feature requirement");
    expect(guidance).toContain("The target was not called for the attempt");
    expect(guidance).toContain("does not resolve any earlier attempt");
  });

  test("retires a resolved feature blocker without erasing the failed attempt's possible effects", () => {
    const raw = JSON.stringify({
      ...JSON.parse(paid),
      agentExecution: {
        version: 1,
        result: "failed",
        effects: "possible",
        reason: "rpc-error",
        httpStatus: 403,
        connectCode: 7,
      },
    });
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user" },
      ...observation(raw),
      ...observation(JSON.stringify({ status: 200, response: {} })),
    ]);
    expect(guidance).not.toContain("paid-feature requirement");
    expect(guidance).not.toContain("unavailable feature");
    expect(guidance).toContain("effects have not been ruled out");
  });

  test("retires resolved query access advice while retaining statement-effect evidence", () => {
    const query = {
      operationId: "SQLService/Query",
      body: {
        name: "instances/example/databases/app",
        statement: "SELECT id FROM orders",
      },
    };
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user" },
      ...observation(
        JSON.stringify({
          status: 200,
          response: {
            results: [
              {
                error: "Denied",
                permissionDenied: {
                  requiredPermissions: ["bb.sql.select"],
                  resources: [query.body.name],
                },
              },
            ],
          },
          agentExecution: {
            version: 1,
            result: "succeeded",
            effects: "possible",
            reason: "query-result-error",
          },
        }),
        "query",
        query
      ),
      ...observation(
        JSON.stringify({ status: 200, response: { results: [] } }),
        "query",
        query
      ),
    ]);
    expect(guidance).not.toContain("get_just_in_time_overview");
    expect(guidance).not.toContain("Request temporary access");
    expect(guidance).toContain(
      "earlier statements may already have had effects"
    );
  });

  test("preserves partial batch evidence after a later successful identical request", () => {
    const request = {
      operationId: "InstanceService/ImportInstances",
      body: { imports: [{ instanceId: "first" }, { instanceId: "second" }] },
    };
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user" },
      ...observation(
        JSON.stringify({
          status: 200,
          response: {},
          agentExecution: {
            version: 1,
            result: "succeeded",
            effects: "partial",
            reason: "import-partial",
          },
        }),
        "batch",
        request
      ),
      ...observation(
        JSON.stringify({ status: 200, response: {} }),
        "batch",
        request
      ),
    ]);
    expect(guidance).toContain("partially completed");
    expect(guidance).toContain("Never repeat the whole batch");
  });

  test.each([
    ["navigate", { error: "Unknown route" }],
    ["get_project_recent_activity", { error: "A bound project is required." }],
    [
      "prepare_masking_rule_draft",
      {
        error: "Choose an exact target",
        needsClarification: true,
        targetEvidence: { candidates: [] },
      },
    ],
  ])("does not treat a local %s result as API failure", (name, result) => {
    expect(
      getAgentApiRecoveryGuidance([
        { role: "user" },
        ...observation(JSON.stringify(result), "local", {}, name),
      ])
    ).toBe("");
  });

  test("explains typed client failure without claiming an observed server response", () => {
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user" },
      ...observation(
        JSON.stringify({
          error: "Unable to load recent activity.",
          apiError: { source: "connect", code: "unavailable" },
        }),
        "activity",
        {},
        "get_project_recent_activity"
      ),
    ]);
    expect(guidance).toContain("API client reported");
    expect(guidance).toContain("availability failure");
    expect(guidance).not.toContain("The API reported");
    expect(guidance).toContain("does not establish an HTTP response");
  });

  test.each([false, true])(
    "keeps preflight provenance explicit in diagnostic mode %s",
    (diagnostic) => {
      const guidance = getAgentApiRecoveryGuidance(
        [
          { role: "user" },
          ...observation(
            JSON.stringify({
              error: "Permission denied",
              preflightStatus: 403,
              preflight: {
                code: "permission_denied",
                requiredPermissions: ["bb.settings.update"],
                resources: ["workspaces/example"],
              },
              agentExecution: {
                version: 1,
                result: "not-sent",
                effects: "none",
                reason: "preflight-rejected",
              },
            })
          ),
        ],
        { diagnostic }
      );
      expect(guidance).toContain("pre-dispatch permission check");
      expect(guidance).toContain("exact required permissions");
      expect(guidance).not.toContain("The API reported");
      expect(guidance).toContain("The target was not called");
    }
  );

  test.each([600, 400.5, "500", null, -1, 399])(
    "does not turn malformed status-only output %s into a failure",
    (status) => {
      expect(
        getAgentApiRecoveryGuidance([
          { role: "user" },
          ...observation(JSON.stringify({ status })),
        ])
      ).toBe("");
    }
  );

  test("keeps explicit failure evidence even with an invalid status", () => {
    expect(
      getAgentApiRecoveryGuidance([
        { role: "user" },
        ...observation(
          JSON.stringify({ status: 600, error: "Unable to load activity" })
        ),
      ])
    ).toContain("Explain the observed failure");
  });

  test.each([
    [
      409,
      { code: "aborted" },
      "resource conflict",
      "before proposing another attempt",
    ],
    [
      503,
      { code: "unavailable" },
      "availability failure",
      "before proposing another potentially mutating attempt",
    ],
    [
      504,
      { code: "deadline_exceeded" },
      "deadline",
      "before another potentially mutating attempt",
    ],
    [
      403,
      {
        code: "permission_denied",
        requiredPermissions: ["bb.settings.get"],
        resources: ["workspaces/example"],
      },
      "permission denial",
      "Suggest requesting the reported permission",
    ],
    [
      403,
      {
        code: "permission_denied",
        message:
          "feature FEATURE_DATA_CLASSIFICATION is a ENTERPRISE feature, please upgrade to access it",
      },
      "paid-feature requirement",
      "Suggest asking the workspace subscription administrator",
    ],
  ] as const)(
    "keeps diagnostic %s guidance explanatory, not operational",
    (status, response, explanation, operational) => {
      const guidance = getAgentApiRecoveryGuidance(
        [
          { role: "user" },
          ...observation(JSON.stringify({ status, response })),
        ],
        { diagnostic: true }
      );
      expect(guidance).toContain(explanation);
      expect(guidance).not.toContain(operational);
      expect(guidance).not.toContain(
        "Suggest one evidence-supported next step"
      );
      expect(guidance).toContain("one-time approval");
      expect(guidance).not.toContain("original read-only scope");
    }
  );

  test("does not propose a potentially mutating attempt after an unknown diagnostic outcome", () => {
    const raw = JSON.stringify({
      agentExecution: {
        version: 1,
        result: "unknown",
        effects: "possible",
        reason: "timed-out",
      },
    });
    const guidance = getAgentApiRecoveryGuidance(
      [{ role: "user" }, ...observation(raw)],
      { diagnostic: true }
    );
    expect(guidance).toContain("outcome could not be verified");
    expect(guidance).not.toContain(
      "before another potentially mutating attempt"
    );
    expect(guidance).not.toContain("Suggest one evidence-supported next step");
  });

  test.each([
    [400, "invalid_argument", "invalid input"],
    [400, "failed_precondition", "prerequisite"],
    [401, "unauthenticated", "session"],
    [403, "permission_denied", "access denial"],
    [404, "not_found", "resource"],
    [409, "already_exists", "resource conflict"],
    [409, "aborted", "resource conflict"],
    [429, "resource_exhausted", "reported limit"],
    [503, "unavailable", "temporarily unavailable"],
    [504, "deadline_exceeded", "deadline"],
    [499, "canceled", "canceled"],
  ])(
    "adds common fallback and code-specific guidance for %s/%s",
    (status, code, detail) => {
      const guidance = getAgentApiRecoveryGuidance([
        { role: "user" },
        ...observation(JSON.stringify({ status, response: { code } })),
      ]);
      expect(guidance).toContain("Explain the observed failure");
      expect(guidance).toContain("one evidence-supported next step");
      expect(guidance).toContain(detail);
      expect(guidance).not.toContain("get_just_in_time_overview");
      expect(guidance).not.toContain("required plan");
    }
  );

  test.each([
    JSON.stringify({ status: 502, error: "HTTP 502", response: null }),
    JSON.stringify({
      status: 500,
      error: "Server failure",
      response: { code: "internal" },
    }),
    JSON.stringify({ error: "Unable to load recent activity." }),
  ])("uses a general fallback for an otherwise unclassified failure", (raw) => {
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user" },
      ...observation(raw),
    ]);
    expect(guidance).toContain("Explain the observed failure");
    expect(guidance).toContain("one evidence-supported next step");
    expect(guidance).not.toContain("required plan");
  });

  test("keeps unknown effects distinct from confirmed rejection", () => {
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user" },
      ...observation(
        JSON.stringify({
          error: "Request timed out",
          agentExecution: {
            version: 1,
            result: "unknown",
            effects: "possible",
            reason: "timed-out",
          },
        })
      ),
    ]);
    expect(guidance).toContain("outcome could not be verified");
    expect(guidance).toContain("read-only verification");
    expect(guidance).not.toContain("required plan");
  });

  test("explains a SQL statement error inside HTTP 200 without proposing query access", () => {
    const query = {
      operationId: "SQLService/Query",
      body: {
        name: "instances/example/databases/app",
        statement: "SELECT invalid SQL",
      },
    };
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user" },
      ...observation(
        JSON.stringify({
          status: 200,
          response: { results: [{ error: "syntax error" }] },
        }),
        "query",
        query
      ),
    ]);
    expect(guidance).toContain("Explain the observed failure");
    expect(guidance).not.toContain("get_just_in_time_overview");
  });

  test("does not offer JIT recovery in diagnostic mode", () => {
    const query = {
      operationId: "SQLService/Query",
      body: {
        name: "instances/example/databases/app",
        statement: "SELECT id FROM orders",
      },
    };
    const raw = JSON.stringify({
      status: 200,
      response: {
        results: [
          {
            error: "Denied",
            permissionDenied: {
              requiredPermissions: ["bb.sql.select"],
              resources: [query.body.name],
            },
          },
        ],
      },
    });
    const guidance = getAgentApiRecoveryGuidance(
      [{ role: "user" }, ...observation(raw, "query", query)],
      { diagnostic: true }
    );
    expect(guidance).toContain("Explain the observed failure");
    expect(guidance).not.toContain("get_just_in_time_overview");
    expect(guidance).not.toContain("Request temporary access");
    expect(guidance).not.toContain("prepare_access_grant_draft");
  });

  test("supplies paid-feature guidance only after an explicit failed response", () => {
    const messages: Message[] = [
      { role: "user", content: "Configure classification" },
    ];
    expect(getAgentApiRecoveryGuidance(messages)).toBe("");
    const guidance = getAgentApiRecoveryGuidance([
      ...messages,
      ...observation(paid),
    ]);
    expect(guidance).toContain("required plan");
    expect(guidance).toContain("current plan");
    expect(guidance).toContain("Never retry");
    expect(guidance).not.toContain("Configure classification");
  });

  test("does not carry a prior user's blocker into a new turn", () => {
    expect(
      getAgentApiRecoveryGuidance([
        { role: "user", content: "Earlier" },
        ...observation(paid),
        { role: "user", content: "Now check databases" },
      ])
    ).toBe("");
  });

  test("retires a resolved failure for the same API request", () => {
    expect(
      getAgentApiRecoveryGuidance([
        { role: "user", content: "Update classification" },
        ...observation(paid),
        ...observation(JSON.stringify({ status: 200, response: {} }), "api", {
          body: { setting: { name: "settings/DATA_CLASSIFICATION" } },
          operationId: "bytebase.v1.SettingService.UpdateSetting",
        }),
      ])
    ).toBe("");
  });

  test("retains query opt-in guidance for all-denied HTTP 200 results", () => {
    const query = {
      operationId: "SQLService/Query",
      body: {
        name: "instances/example/databases/app",
        statement: "SELECT id FROM orders",
      },
    };
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user", content: "Query orders" },
      ...observation(
        JSON.stringify({
          status: 200,
          response: {
            results: [
              {
                error: "Permission denied",
                permissionDenied: {
                  requiredPermissions: ["bb.sql.select"],
                  resources: [query.body.name],
                },
              },
            ],
          },
        }),
        "query",
        query
      ),
    ]);
    expect(guidance).toContain("get_just_in_time_overview");
    expect(guidance).toContain("Request temporary access");
    expect(guidance).toContain("Wait for response.confirmed");
  });

  test("recognizes typed-reader errors without relying on a model tool choice", () => {
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user", content: "Recent activity" },
      {
        role: "assistant",
        toolCalls: [
          {
            id: "activity",
            name: "get_project_recent_activity",
            arguments: "{}",
          },
        ],
      },
      { role: "tool", toolCallId: "activity", content: paid },
    ]);
    expect(guidance).toContain("required plan");
  });

  test("does not turn unrelated errors or SQL values into upgrade advice", () => {
    for (const result of [
      JSON.stringify({
        status: 403,
        response: { code: "permission_denied", message: "Permission denied" },
      }),
      JSON.stringify({ status: 200, response: { rows: [{ value: paid }] } }),
      JSON.stringify({ error: "Network failure" }),
      "malformed",
    ]) {
      const guidance = getAgentApiRecoveryGuidance([
        { role: "user" },
        ...observation(result),
      ]);
      expect(guidance).not.toContain("required plan");
      expect(guidance).not.toContain("Request temporary access");
    }
  });

  test.each([
    JSON.stringify({ status: 200, response: { rows: [{ value: paid }] } }),
    JSON.stringify({ status: 200, response: {} }),
    "malformed",
    JSON.stringify({ apiRecovery: { kind: "feature-required" } }),
  ])("does not produce fallback without failure evidence", (result) => {
    expect(
      getAgentApiRecoveryGuidance([{ role: "user" }, ...observation(result)])
    ).toBe("");
  });

  test.each([
    [
      "canceled-before-dispatch",
      "not-sent",
      "none",
      "The target was not called",
    ],
    [
      "canceled-after-dispatch",
      "unknown",
      "possible",
      "outcome could not be verified",
    ],
  ])(
    "keeps cancellation timing explicit for %s",
    (reason, result, effects, expected) => {
      const guidance = getAgentApiRecoveryGuidance([
        { role: "user" },
        ...observation(
          JSON.stringify({
            agentExecution: { version: 1, reason, result, effects },
          })
        ),
      ]);
      expect(guidance).toContain(expected);
      expect(guidance).not.toContain("Request temporary access");
    }
  );

  test("preserves partially completed batch evidence", () => {
    const request = {
      operationId: "InstanceService/ImportInstances",
      body: { imports: [{ instanceId: "first" }, { instanceId: "second" }] },
    };
    const result = JSON.stringify({
      status: 200,
      response: {},
      agentExecution: {
        version: 1,
        result: "succeeded",
        effects: "partial",
        reason: "import-partial",
      },
    });
    const guidance = getAgentApiRecoveryGuidance([
      { role: "user" },
      ...observation(result, "batch", request),
    ]);
    expect(guidance).toContain("partially completed");
    expect(guidance).toContain("Never repeat the whole batch");
  });
});
