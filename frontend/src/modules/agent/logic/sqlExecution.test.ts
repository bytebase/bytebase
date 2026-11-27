import { create } from "@bufbuild/protobuf";
import { expect, test, vi } from "vitest";
import { useAppStore } from "@/stores/app";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { type AgentState, useAgentStore } from "../store/agent";
import { agentApiExecutionEvidence } from "./apiExecution";
import * as recovery from "./runOwnership";
import type { AgentSqlEditorSnapshot } from "./sqlEditorContext";
import { beginSqlResponseAction, buildAgentSqlRequest } from "./sqlExecution";
import { bindAgentCompletedQueryActions } from "./sqlResponse";
import * as admission from "./tools/apiAdmission";
import * as api from "./tools/callApi";
import type { AgentChat, AgentMessage, AgentRunBindings } from "./types";

const snapshot: AgentSqlEditorSnapshot = {
  version: 1,
  project: "projects/p",
  tab: { id: "t", mode: "SAVED_QUERY" },
  connection: {
    project: "projects/p",
    database: "instances/a/databases/d",
    engine: Engine.POSTGRES,
    querySchema: "public",
    container: "part",
    dataSourceId: "reader",
  },
  rowLimit: 504,
  queryOptions: {},
  batchActive: false,
};
test("Query uses only the captured single connection, options and exact SQL bytes", () => {
  const sql = " SELECT 1;\r\nSELECT 2; ";
  expect(buildAgentSqlRequest(snapshot, sql)).toEqual({
    operationId: "bytebase.v1.SQLService.Query",
    body: {
      name: "instances/a/databases/d",
      statement: sql,
      schema: "public",
      container: "part",
      dataSourceId: "reader",
      queryOption: {},
      limit: 504,
    },
  });
  expect(
    buildAgentSqlRequest({ ...snapshot, batchActive: true }, sql)
  ).toBeUndefined();
  expect(
    buildAgentSqlRequest({ ...snapshot, tab: { id: "t", mode: "ADMIN" } }, sql)
  ).toBeUndefined();
  expect(
    buildAgentSqlRequest({ ...snapshot, project: "projects/foreign" }, sql)
  ).toBeUndefined();
});

test("Run again from a completed result enters normal approval without dispatch", async () => {
  const captured: AgentSqlEditorSnapshot = {
    ...snapshot,
    editorTarget: {
      actor: "users/member",
      workspace: "workspaces/example",
      hostId: "host",
      mountId: "mount",
      project: "projects/p",
      tabId: "t",
      tabMode: "SAVED_QUERY",
      modelUri: "file:///query.sql",
      modelVersion: 1,
      connection: snapshot.connection,
      selection: { start: 0, end: 0, caret: 0 },
    },
  };
  const messages: AgentMessage[] = [
    {
      id: "initial-user",
      chatId: "chat",
      role: "user",
      content: "Show a value",
      createdTs: 1,
      metadata: { sqlEditor: captured },
    },
    {
      id: "call",
      chatId: "chat",
      role: "assistant",
      createdTs: 2,
      metadata: { runId: "original-run" },
      toolCalls: [
        {
          id: "query",
          name: "call_api",
          arguments: JSON.stringify({
            operationId: "SQLService/Query",
            body: {
              name: captured.connection!.database,
              statement: "SELECT 27;",
              schema: captured.connection!.querySchema,
              container: captured.connection!.container,
              dataSourceId: captured.connection!.dataSourceId,
              queryOption: captured.queryOptions,
            },
          }),
        },
      ],
    },
    {
      id: "result",
      chatId: "chat",
      role: "tool",
      toolCallId: "query",
      createdTs: 3,
      metadata: { runId: "original-run" },
      content: JSON.stringify({
        status: 200,
        agentExecution: agentApiExecutionEvidence("rpc-success"),
        response: {
          results: [
            {
              columnNames: ["value"],
              rows: [{ values: [{ int32Value: 27 }] }],
            },
          ],
        },
      }),
    },
    {
      id: "answer",
      chatId: "chat",
      role: "assistant",
      content: "Returned one value.",
      createdTs: 4,
      metadata: {
        runId: "original-run",
        sqlEditor: captured,
        sqlActionProvenance: {
          version: 1,
          runId: "original-run",
          runMode: "default",
          diagnostic: false,
        },
      },
    },
  ];
  const source = bindAgentCompletedQueryActions(messages[3], messages)[0];
  expect(source).toBeDefined();
  const slot = document.createElement("div");
  document.body.append(slot);
  const chat: AgentChat = {
    id: "chat",
    title: "Query",
    context: "projects/p",
    workspaceName: "workspaces/example",
    status: "idle",
    createdTs: 1,
    updatedTs: 4,
    archived: false,
    totalTokensUsed: 0,
  };
  const root = { path: "/sql-editor/projects/p", title: "SQL Editor" };
  let controller: AbortController | null = null;
  const awaitUser = vi.fn();
  const state: AgentState = {
    ...useAgentStore.getState(),
    workspaceName: "workspaces/example",
    currentChatId: "chat",
    chats: [chat],
    remoteBusyChatIds: {},
    abortControllersByChatId: {},
    queuedUserMessageInFlightByChatId: {},
    queuedUserMessagesByChatId: {},
    followUpQueuesByChatId: {},
    followUpDrainChatIds: {},
    getChat: () => chat,
    getMessages: () => messages,
    getPendingAsk: () => null,
    canUseAgentExecution: () => true,
    captureApiScope: () => () => true,
    syncState: () => true,
    checkAgentStorageAdmission: async () => true,
    flushRunPersistence: async () => true,
    sqlEditorHost: () => ({
      id: "host",
      slot,
      mainColumn: slot,
      resizer: slot,
      paneRatio: 0.4,
      captureContext: () => ({ kind: "ready", snapshot: captured }),
    }),
    addMessage(message) {
      const next = {
        ...message,
        id: `message-${messages.length}`,
        chatId: "chat",
        createdTs: messages.length + 1,
      };
      messages.push(next);
      return next;
    },
    async beginChatRun(
      _chatId: string,
      rootUserMessageId: string,
      bindings: AgentRunBindings
    ) {
      chat.status = "running";
      chat.runId = "new-run";
      chat.runCheckpoint = {
        version: 1,
        runId: "new-run",
        rootUserMessageId,
        throughMessageId: rootUserMessageId,
        bindings,
        suspension: "awaiting-user",
        maskingSections: [],
        phase: "working",
      };
      return "new-run";
    },
    setAbortController(_chatId, next) {
      const previous = controller;
      controller = next;
      return previous;
    },
    getAbortController: () => controller,
    awaitUser,
  };
  const app = useAppStore.getState();
  vi.spyOn(useAgentStore, "getState").mockReturnValue(state);
  vi.spyOn(useAppStore, "getState").mockReturnValue({
    ...app,
    currentUser: create(UserSchema, {
      name: "users/member",
      workspace: "workspaces/example",
      email: "member@example.com",
    }),
  });
  vi.spyOn(recovery, "supportsAgentRunRecovery").mockReturnValue(true);
  const prepare = vi
    .spyOn(api, "prepareApiCall")
    .mockImplementation(async (request) => ({
      kind: "prepared",
      call: {
        operationId: request.operationId,
        path: "/bytebase.v1.SQLService/Query",
        body: JSON.stringify(request.body),
      },
    }));
  const admit = vi
    .spyOn(admission, "admitPreparedAgentApiCall")
    .mockResolvedValue({
      kind: "ask_user",
      ask: {
        toolCallId: "new-call",
        kind: "confirm",
        prompt: "Run this exact query?",
        apiPermission: {
          actionId: "permission",
          operationId: "bytebase.v1.SQLService.Query",
        },
      },
    });
  const dispatch = vi.spyOn(api, "dispatchPreparedApiCall");
  try {
    expect(
      await beginSqlResponseAction(
        source,
        captured,
        root,
        "Run again",
        undefined
      )
    ).toBe(true);
    expect(prepare).toHaveBeenCalledWith(
      buildAgentSqlRequest(captured, "SELECT 27;")
    );
    expect(admit).toHaveBeenCalledTimes(1);
    expect(admit.mock.calls[0][0].sqlOrigin?.isCurrent()).toBe(true);
    expect(awaitUser).toHaveBeenCalledWith(
      "chat",
      expect.objectContaining({
        kind: "confirm",
        apiPermission: expect.objectContaining({ actionId: "permission" }),
      })
    );
    expect(dispatch).not.toHaveBeenCalled();
    const resultIndex = messages.findIndex(({ id }) => id === "result");
    messages[resultIndex] = {
      ...messages[resultIndex],
      content: JSON.stringify({ error: "Source changed" }),
    };
    expect(admit.mock.calls[0][0].sqlOrigin?.isCurrent()).toBe(false);
  } finally {
    vi.restoreAllMocks();
    slot.remove();
  }
});
