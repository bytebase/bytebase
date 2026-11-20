import { describe, expect, it } from "vitest";
import type { AgentApiAttempt } from "../logic/apiAttempt";
import { agentApiExecutionEvidence } from "../logic/apiExecution";
import type { AgentChat, AgentDraft, AgentRunCheckpoint } from "../logic/types";
import type { PersistedAgentState } from "./agent";
import {
  getAgentHistoryEvictionCandidates,
  removeAgentHistoryChats,
} from "./historyRetention";

const chat = (id: string, overrides: Partial<AgentChat> = {}): AgentChat => ({
  id,
  title: id,
  createdTs: 1,
  updatedTs: 2,
  status: "idle",
  archived: false,
  totalTokensUsed: 42,
  runId: null,
  ...overrides,
});

const stateWith = (chats: AgentChat[]): PersistedAgentState => ({
  currentChatId: null,
  chats,
  messagesByChatId: Object.fromEntries(
    chats.map(({ id }) => [
      id,
      [
        {
          id: `message-${id}`,
          chatId: id,
          createdTs: 1,
          role: "assistant",
          content: "Completed response",
        },
      ],
    ])
  ),
  pendingAskByChatId: {},
  draftsByChatId: {},
  queuedUserMessagesByChatId: {},
});

const checkpoint: AgentRunCheckpoint = {
  version: 1,
  runId: "run-example",
  suspension: "interrupted",
  rootUserMessageId: "user-example",
  throughMessageId: "message-example",
  bindings: {
    page: { path: "/", title: "Workspace" },
    references: [],
    runMode: "default",
  },
  maskingSections: [],
  phase: "working",
};

describe("Agent history eviction candidates", () => {
  it.each(["rpc-success", "fetch-failed", "import-partial"] as const)(
    "legacy review metadata does not determine cleanup eligibility for %s",
    (reason) => {
      const execution = agentApiExecutionEvidence(reason, {
        attemptId: "attempt",
        ...(reason !== "fetch-failed" && { httpStatus: 200 }),
      });
      const receipt: AgentApiAttempt = {
        version: 1,
        id: "attempt",
        logicalActionId: "action",
        ordinal: 1,
        scopeKey: "scope",
        userName: "users/member@example.com",
        chatId: "history",
        runId: "run",
        toolCallId: "tool",
        rootUserMessageId: "root",
        operationId:
          reason === "import-partial"
            ? "bytebase.v1.InstanceService.ImportInstances"
            : "bytebase.v1.DatabaseService.UpdateDatabase",
        path:
          reason === "import-partial"
            ? "/bytebase.v1.InstanceService/ImportInstances"
            : "/bytebase.v1.DatabaseService/UpdateDatabase",
        fingerprint: "a".repeat(64),
        phase: "settled",
        startedAt: 1,
        settledAt: 2,
        execution,
        reviewedExternallyAt: 42,
      };
      const state = stateWith([chat("history", { apiAttempts: [receipt] })]);
      state.messagesByChatId.history = [
        {
          id: "result",
          chatId: "history",
          role: "tool",
          createdTs: 2,
          toolCallId: "tool",
          metadata: { runId: "run", apiAttemptId: "attempt" },
          content: JSON.stringify({
            ...(reason !== "fetch-failed" && { status: 200 }),
            agentExecution: execution,
          }),
        },
      ];
      expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([
        "history",
      ]);
      expect(
        removeAgentHistoryChats(state, new Set(["history"])).chats
      ).toHaveLength(0);
      state.messagesByChatId.history = [];
      expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([]);
      expect(
        removeAgentHistoryChats(state, new Set(["history"])).chats
      ).toHaveLength(1);
      state.chats[0].apiAttempts = [
        {
          ...receipt,
          phase: "dispatching",
          execution: undefined,
          settledAt: undefined,
        },
      ];
      expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([
        "history",
      ]);
    }
  );

  it("does not indefinitely retain an interrupted call after its run is no longer active", () => {
    const state = stateWith([chat("old", { interrupted: true })]);
    state.messagesByChatId.old = [
      { id: "root", role: "user", chatId: "old", createdTs: 1 },
      {
        id: "assistant",
        role: "assistant",
        chatId: "old",
        createdTs: 2,
        toolCalls: [
          {
            id: "tool",
            name: "call_api",
            arguments: '{"operationId":"DatabaseService/UpdateDatabase"}',
          },
        ],
      },
    ];
    expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([
      "old",
    ]);
    expect(removeAgentHistoryChats(state, new Set(["old"])).chats).toEqual([]);
    state.messagesByChatId.old.push({
      id: "new",
      chatId: "old",
      createdTs: 3,
      role: "user",
    });
    expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([
      "old",
    ]);
    state.messagesByChatId.old.pop();
    state.pendingAskByChatId.old = {
      kind: "confirm",
      prompt: "Permit?",
      toolCallId: "tool",
    };
    expect(removeAgentHistoryChats(state, new Set(["old"])).chats).toEqual([]);
    delete state.pendingAskByChatId.old;
    state.messagesByChatId.old.push({
      id: "result",
      role: "tool",
      chatId: "old",
      toolCallId: "tool",
      content: '{"status":200}',
      createdTs: 3,
    });
    expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([
      "old",
    ]);
  });
  it("unknown journal formats survive automatic and bulk reclamation", () => {
    const state = stateWith([
      chat("protected", { archived: true }),
      chat("done"),
    ]);
    Object.assign(state.chats[0], { apiAttempts: { version: 9 } });
    expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([
      "done",
    ]);
    expect(
      removeAgentHistoryChats(state, new Set(["protected", "done"])).chats.map(
        (c) => c.id
      )
    ).toEqual(["protected"]);
  });
  it("includes completed and archived chats without reading tool results", () => {
    const state = stateWith([
      chat("completed"),
      chat("archived", { archived: true }),
      chat("error", { status: "error", lastError: "Failed" }),
    ]);
    state.messagesByChatId.completed = [
      {
        id: "tool-completed",
        chatId: "completed",
        createdTs: 1,
        role: "tool",
        toolCallId: "call-completed",
        content: "opaque non-JSON tool result",
        metadata: { runId: "finished-run" },
      },
    ];

    expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([
      "archived",
      "completed",
      "error",
    ]);
  });

  it("orders oldest updated time, created time, and stable chat ID first", () => {
    const state = stateWith([
      chat("newest", { createdTs: 1, updatedTs: 30 }),
      chat("tie-b", { createdTs: 5, updatedTs: 20 }),
      chat("oldest", { createdTs: 9, updatedTs: 10 }),
      chat("earlier-created", { createdTs: 3, updatedTs: 20 }),
      chat("tie-a", { createdTs: 5, updatedTs: 20 }),
    ]);
    const before = structuredClone(state);

    expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([
      "oldest",
      "earlier-created",
      "tie-a",
      "tie-b",
      "newest",
    ]);
    expect(state).toEqual(before);
  });

  it.each<{
    reason: string;
    protect: (state: PersistedAgentState) => void;
  }>([
    {
      reason: "current idle chat",
      protect: (state) => {
        state.currentChatId = "protected";
      },
    },
    ...(["running", "paused", "awaiting_user"] as const).map((status) => ({
      reason: `${status} chat`,
      protect: (state: PersistedAgentState) => {
        state.chats[0].status = status;
      },
    })),
    {
      reason: "run ownership",
      protect: (state) => {
        state.chats[0].runId = "run-example";
      },
    },
    {
      reason: "recoverable checkpoint",
      protect: (state) => {
        state.chats[0].runCheckpoint = checkpoint;
      },
    },
    {
      reason: "pending approval",
      protect: (state) => {
        state.pendingAskByChatId.protected = {
          toolCallId: "call-example",
          prompt: "Confirm the request?",
          kind: "confirm",
        };
      },
    },
    {
      reason: "queued message",
      protect: (state) => {
        state.queuedUserMessagesByChatId.protected = {
          id: "queued-example",
          chatId: "protected",
          text: "Next request",
          page: { path: "/", title: "Workspace" },
        };
      },
    },
    {
      reason: "active chat",
      protect: (state) => {
        state.activeChatIds = ["protected"];
      },
    },
    {
      reason: "paused follow-up queue",
      protect: (state) => {
        state.followUpQueuesByChatId = {
          protected: {
            paused: true,
            messages: [
              {
                id: "follow-up",
                chatId: "protected",
                text: "Next request",
                page: { path: "/", title: "Workspace" },
              },
            ],
          },
        };
      },
    },
    {
      reason: "incomplete message",
      protect: (state) => {
        state.messagesByChatId.protected[0].metadata = { pending: true };
      },
    },
    {
      reason: "unknown status",
      protect: (state) => {
        state.chats[0].status = "future-status" as AgentChat["status"];
      },
    },
  ])("protects $reason", ({ protect }) => {
    const state = stateWith([chat("protected"), chat("eligible")]);
    protect(state);

    expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([
      "eligible",
    ]);
  });

  it.each<{ reason: string; draft: AgentDraft }>([
    { reason: "text", draft: { text: "Unsent request" } },
    { reason: "whitespace text", draft: { text: " " } },
    {
      reason: "project",
      draft: {
        text: "",
        project: { name: "projects/example", title: "Example" },
      },
    },
    {
      reason: "resource",
      draft: {
        text: "",
        resource: {
          kind: "database",
          name: "instances/example/databases/example",
          project: "projects/example",
          label: "Example",
        },
      },
    },
    {
      reason: "mentions",
      draft: {
        text: "",
        mentions: [
          {
            resource: {
              kind: "project",
              name: "projects/example",
              project: "projects/example",
              label: "Example",
            },
            range: { start: 0, end: 7 },
          },
        ],
      },
    },
  ])("protects an unselected chat with draft $reason", ({ draft }) => {
    const state = stateWith([chat("draft"), chat("eligible")]);
    state.draftsByChatId.draft = draft;

    expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([
      "eligible",
    ]);
  });

  it("does not protect an empty draft", () => {
    const state = stateWith([chat("empty")]);
    state.draftsByChatId.empty = { text: "", mentions: [] };

    expect(getAgentHistoryEvictionCandidates(state, new Set())).toEqual([
      "empty",
    ]);
  });

  it.each([
    "remote run",
    "abort controller",
    "in-flight queue",
    "actionable SQL fix draft",
    "unsaved outcome",
    "open chat in another tab",
  ])("protects externally supplied %s IDs", () => {
    const state = stateWith([chat("protected"), chat("eligible")]);

    expect(
      getAgentHistoryEvictionCandidates(state, new Set(["protected"]))
    ).toEqual(["eligible"]);
  });
});

describe("Agent history removal", () => {
  it("removes requested chat-owned state without changing retained records or unknown fields", () => {
    const state = {
      ...stateWith([chat("remove"), chat("keep")]),
      currentChatId: "keep",
      activeChatIds: ["remove", "keep"],
      futureState: { remove: "preserve unknown state" },
    };
    state.draftsByChatId = {
      remove: { text: "Remove draft" },
      keep: { text: "Retain draft" },
    };
    state.pendingAskByChatId = {
      remove: { toolCallId: "remove-call", prompt: "Remove?", kind: "confirm" },
      keep: { toolCallId: "keep-call", prompt: "Keep?", kind: "confirm" },
    };
    state.queuedUserMessagesByChatId = {
      remove: {
        id: "remove-queue",
        chatId: "remove",
        text: "Remove",
        page: { path: "/", title: "Workspace" },
      },
      keep: {
        id: "keep-queue",
        chatId: "keep",
        text: "Keep",
        page: { path: "/", title: "Workspace" },
      },
    };
    const before = structuredClone(state);

    const retained = removeAgentHistoryChats(state, new Set(["remove"]));

    expect(retained.chats.map(({ id }) => id)).toEqual(["keep"]);
    expect(Object.keys(retained.messagesByChatId)).toEqual(["keep"]);
    expect(Object.keys(retained.draftsByChatId)).toEqual(["keep"]);
    expect(Object.keys(retained.pendingAskByChatId)).toEqual(["keep"]);
    expect(Object.keys(retained.queuedUserMessagesByChatId)).toEqual(["keep"]);
    expect(retained.currentChatId).toBe("keep");
    expect(retained.activeChatIds).toEqual(["keep"]);
    expect(retained).toHaveProperty("futureState", state.futureState);
    expect(retained.chats[0]).toBe(state.chats[1]);
    expect(retained.messagesByChatId.keep).toBe(state.messagesByChatId.keep);
    expect(retained.messagesByChatId.keep[0].id).toBe("message-keep");
    expect(retained.draftsByChatId.keep).toBe(state.draftsByChatId.keep);
    expect(retained.pendingAskByChatId.keep).toBe(
      state.pendingAskByChatId.keep
    );
    expect(retained.queuedUserMessagesByChatId.keep).toBe(
      state.queuedUserMessagesByChatId.keep
    );
    expect(state).toEqual(before);
  });

  it("clears current chat only when that chat is explicitly removed", () => {
    const state = stateWith([chat("remove")]);
    state.currentChatId = "remove";

    expect(removeAgentHistoryChats(state, new Set(["remove"]))).toMatchObject({
      currentChatId: null,
      chats: [],
      messagesByChatId: {},
    });
    expect(state.currentChatId).toBe("remove");
  });

  it("keeps unaffected bucket references and absent active-chat state", () => {
    const state = stateWith([chat("remove"), chat("keep")]);

    const retained = removeAgentHistoryChats(state, new Set(["remove"]));

    expect(retained.draftsByChatId).toBe(state.draftsByChatId);
    expect(retained.pendingAskByChatId).toBe(state.pendingAskByChatId);
    expect(retained.queuedUserMessagesByChatId).toBe(
      state.queuedUserMessagesByChatId
    );
    expect(retained).not.toHaveProperty("activeChatIds");
  });

  it.each([new Set<string>(), new Set(["unknown-chat"])])(
    "returns the original state when no chat-owned state is removed",
    (ids) => {
      const state = stateWith([chat("keep")]);

      expect(removeAgentHistoryChats(state, ids)).toBe(state);
    }
  );
});
