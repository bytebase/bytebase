import { expect, test } from "vitest";
import {
  type AgentApiAttempt,
  fingerprintPreparedApiCall,
  isAgentApiAttemptUncertain,
  mergeAgentApiAttempts,
  normalizeAgentApiAttempts,
  priorUncertainApiAttemptIds,
  reconcileApiResultLinks,
  storedApiReceipts,
} from "./apiAttempt";
import { agentApiExecutionEvidence } from "./apiExecution";

const operationId = "bytebase.v1.SQLService.Query";
const path = "/bytebase.v1.SQLService/Query";
const hash = (body: string) =>
  fingerprintPreparedApiCall({ operationId, path, body });
const pending: AgentApiAttempt = {
  version: 1,
  id: "id",
  logicalActionId: "action",
  ordinal: 1,
  scopeKey: "scope",
  userName: "users/member",
  chatId: "chat",
  runId: "run",
  toolCallId: "tool",
  rootUserMessageId: "root",
  operationId,
  path,
  fingerprint: "a".repeat(64),
  phase: "dispatching",
  startedAt: 1,
};
const settled: AgentApiAttempt = {
  ...pending,
  phase: "settled",
  settledAt: 2,
  execution: agentApiExecutionEvidence("rpc-success", {
    attemptId: "id",
    httpStatus: 200,
  }),
};

test.each([
  pending,
  {
    ...settled,
    execution: agentApiExecutionEvidence("fetch-failed", { attemptId: "id" }),
  },
  {
    ...settled,
    execution: agentApiExecutionEvidence("import-partial", {
      attemptId: "id",
      httpStatus: 200,
    }),
  },
])(
  "a legacy review timestamp does not settle uncertain evidence %#",
  (attempt) => {
    expect(
      isAgentApiAttemptUncertain({ ...attempt, reviewedExternallyAt: 42 })
    ).toBe(true);
  }
);

test("legacy review metadata on a valid receipt survives without changing its outcome", () => {
  const legacy = { ...settled, reviewedExternallyAt: 42 };
  expect(normalizeAgentApiAttempts([legacy], "chat")).toEqual([legacy]);
  expect(isAgentApiAttemptUncertain(legacy)).toBe(false);
});

test.each([
  {
    kind: "database-labels",
    name: "projects/owner/instances/example/databases/db",
    project: "projects/owner",
    labelsDigest: "c".repeat(64),
  },
  { kind: "unsupported-legacy-capsule" },
])(
  "obsolete observation data does not change a known outcome %#",
  (observationTarget) => {
    const receipt = {
      ...settled,
      operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
      path: "/bytebase.v1.DatabaseService/UpdateDatabase",
    };
    const [normalized] = normalizeAgentApiAttempts(
      [{ ...receipt, observationTarget }],
      receipt.chatId
    )!;
    expect(normalized).toEqual(receipt);
    expect(normalized).not.toHaveProperty("observationTarget");
    expect(isAgentApiAttemptUncertain(normalized)).toBe(false);
  }
);

test.each([undefined, 4])(
  "only an exact uncertain receipt contributes a repeat warning, reviewed=%s",
  (reviewedExternallyAt) => {
    const chats = [
      { apiAttempts: [{ ...pending, reviewedExternallyAt }] },
      { apiAttempts: normalizeAgentApiAttempts([null], "peer") },
    ];
    expect(priorUncertainApiAttemptIds(chats, pending.fingerprint)).toEqual([
      "id",
    ]);
    expect(priorUncertainApiAttemptIds(chats, "b".repeat(64))).toEqual([]);
    expect(priorUncertainApiAttemptIds(chats, "*")).toEqual([]);
  }
);

test("fingerprints structured canonical JSON without changing arrays, literals or operation", async () => {
  const first = await hash('{"x":[1,2],"labels":{"a":" A ","b":true}}');
  expect(first).toMatch(/^[a-f0-9]{64}$/);
  expect(await hash('{"labels":{"b":true,"a":" A "},"x":[1,2]}')).toBe(first);
  expect(await hash('{"x":[2,1],"labels":{"a":" A ","b":true}}')).not.toBe(
    first
  );
  expect(await hash('{"x":[1,2],"labels":{"a":"A","b":true}}')).not.toBe(first);
});

test("terminal evidence wins in either merge order; contradictions stay uncertain", () => {
  for (const [a, b] of [
    [pending, settled],
    [settled, pending],
  ]) {
    expect(mergeAgentApiAttempts([a], [b])).toEqual([settled]);
  }
  const other = {
    ...settled,
    execution: agentApiExecutionEvidence("fetch-failed", { attemptId: "id" }),
  };
  const [conflict] = mergeAgentApiAttempts([settled], [other])!;
  expect(conflict.execution?.reason).toBe("journal-conflict");
  expect(isAgentApiAttemptUncertain(conflict)).toBe(true);
  expect(
    mergeAgentApiAttempts([conflict], [settled])![0].execution?.reason
  ).toBe("journal-conflict");
});

test.each([
  null,
  {},
  [null],
  [{ ...pending, version: 2 }],
  [{ ...settled, execution: undefined }],
])("unreadable journal entries stay uncertain", (raw) => {
  const normalized = normalizeAgentApiAttempts(raw, "chat")!;
  expect(normalized.some(isAgentApiAttemptUncertain)).toBe(true);
  expect(normalizeAgentApiAttempts(normalized, "chat")).toEqual(normalized);
});

test("conflicting unsupported receipts survive a save despite an obsolete review timestamp", () => {
  const sources = [
    { id: "opaque", version: 2 },
    { id: "opaque", version: 3 },
  ];
  const [attempt] = normalizeAgentApiAttempts(sources, "chat")!;
  expect(attempt.execution?.reason).toBe("journal-unreadable");
  expect(storedApiReceipts(attempt)).toEqual(sources);
  const noted = { ...attempt, reviewedExternallyAt: 42 };
  expect(storedApiReceipts(noted)).toEqual(sources);
  const [restored] = normalizeAgentApiAttempts(
    storedApiReceipts(noted),
    "chat"
  )!;
  expect(restored).toMatchObject({
    opaqueReceipts: sources,
    execution: { reason: "journal-unreadable" },
  });
  const nested = { ...attempt, opaqueReceipts: [settled] };
  expect(storedApiReceipts(nested)).toEqual([nested]);
});

test("a wrong-scope or missing journal, an unlinked settlement, and an operation-inapplicable no-effect reason stay uncertain", () => {
  const row = {
    id: "result",
    role: "tool" as const,
    createdTs: 2,
    chatId: "chat",
    toolCallId: "tool",
    metadata: { runId: "run", apiAttemptId: "id" },
    content: JSON.stringify({ status: 200, agentExecution: settled.execution }),
  };
  const impossible = {
    ...settled,
    execution: agentApiExecutionEvidence("rpc-rejected-before-effects", {
      attemptId: "id",
      httpStatus: 500,
      connectCode: 3,
    }),
  };
  for (const result of [
    reconcileApiResultLinks("chat", "scope", undefined, [row]),
    reconcileApiResultLinks("chat", "other", [settled], [row]),
    reconcileApiResultLinks("chat", "scope", [settled], []),
    normalizeAgentApiAttempts([impossible], "chat"),
  ])
    expect(result?.some(isAgentApiAttemptUncertain)).toBe(true);
  expect(reconcileApiResultLinks("chat", "scope", [settled], [row])).toEqual([
    settled,
  ]);
});
