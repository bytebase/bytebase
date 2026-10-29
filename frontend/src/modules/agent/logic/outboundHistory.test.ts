import { describe, expect, test } from "vitest";
import {
  buildOutboundHistory,
  sanitizeOutboundHistory,
} from "./outboundHistory";
import type { AgentMessage } from "./types";

const userMessage = (metadata?: AgentMessage["metadata"]): AgentMessage => ({
  id: "turn-1",
  chatId: "chat-1",
  createdTs: 1,
  role: "user",
  content: "What changed?",
  metadata,
});

describe("sanitizeOutboundHistory", () => {
  test("adds one-turn Project reference without changing the chat question or later turns", () => {
    const message = userMessage({
      resourceReference: {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
    });
    const followUp = { ...userMessage(), id: "turn-2", content: "And now?" };

    const [outbound, next] = sanitizeOutboundHistory([message, followUp]);

    expect(message.content).toBe("What changed?");
    expect(outbound.content).toContain('"kind":"project"');
    expect(outbound.content).toContain('"name":"projects/billing"');
    expect(outbound.content).not.toContain("Billing");
    expect(outbound.content).toContain("this user message only");
    expect(next.content).toBe("And now?");
  });

  test("adds a canonical selected resource as data without changing the visible turn", () => {
    const message = userMessage({
      resourceReference: {
        kind: "issue",
        name: "projects/orders/issues/12",
        project: "projects/orders",
        label: "A changed title",
      },
    });

    const [outbound] = sanitizeOutboundHistory([message]);

    expect(message.content).toBe("What changed?");
    expect(outbound.content).toContain("What changed?");
    expect(outbound.content).toContain("projects/orders/issues/12");
    expect(outbound.content).toContain("projects/orders");
    expect(outbound.content).not.toContain("A changed title");
  });

  test("forwards a selected project and database together for the current turn", () => {
    const message = userMessage({
      resourceReferences: [
        {
          kind: "project",
          name: "projects/orders",
          project: "projects/orders",
          label: "Orders",
        },
        {
          kind: "database",
          name: "instances/prod/databases/orders",
          project: "projects/orders",
          label: "Orders DB",
        },
      ],
    });

    const [outbound] = sanitizeOutboundHistory([message], "projects/other");

    expect(message.content).toBe("What changed?");
    expect(outbound.content).toContain('"kind":"project"');
    expect(outbound.content).toContain('"kind":"database"');
    expect(outbound.content).toContain('"name":"projects/orders"');
    expect(outbound.content).toContain(
      '"name":"instances/prod/databases/orders"'
    );
    expect(outbound.content).not.toContain("Orders DB");
  });

  test("forwards a selected project and its issue together for the current turn", () => {
    const message = userMessage({
      resourceReferences: [
        {
          kind: "issue",
          name: "projects/orders/issues/12",
          project: "projects/orders",
          label: "Change orders table",
        },
        {
          kind: "project",
          name: "projects/orders",
          project: "projects/orders",
          label: "Orders",
        },
      ],
    });

    const [outbound] = sanitizeOutboundHistory([message], "workspaces/default");

    expect(outbound.content).toContain('"kind":"project"');
    expect(outbound.content).toContain('"kind":"issue"');
    expect(outbound.content).toContain('"name":"projects/orders/issues/12"');
    expect(outbound.content).not.toContain("Change orders table");
  });

  test("does not forward malformed cached references", () => {
    const message = userMessage({
      resourceReference: {
        kind: "issue",
        name: "projects/other/issues/12",
        project: "projects/orders",
        label: "Wrong owner",
      },
    });

    expect(sanitizeOutboundHistory([message])).toEqual([
      {
        role: "user",
        content: "What changed?",
        toolCallId: undefined,
        toolCalls: undefined,
      },
    ]);
  });

  test("tells the agent when the user stopped an earlier request without forwarding its incomplete tools", () => {
    const stopped = userMessage({ interruptionReason: "user" });
    const partialToolCall: AgentMessage = {
      ...userMessage(),
      id: "call-1",
      role: "assistant",
      content: undefined,
      toolCalls: [{ id: "tool-1", name: "call_api", arguments: "{}" }],
    };
    const partialToolResult: AgentMessage = {
      ...userMessage(),
      id: "result-1",
      role: "tool",
      toolCallId: "tool-1",
      content: '{"status":200}',
    };
    const followUp = {
      ...userMessage(),
      id: "turn-2",
      content: "What happened?",
    };

    const history = sanitizeOutboundHistory([
      stopped,
      partialToolCall,
      partialToolResult,
      followUp,
    ]);

    expect(history).toHaveLength(2);
    expect(history[0].content).toContain("stopped by the user");
    expect(history[0].content).toContain("may have completed");
    expect(history[1].content).toBe("What happened?");
    expect(stopped.content).toBe("What changed?");
  });

  test("distinguishes an unexpected interruption from a user stop", () => {
    const interrupted = userMessage({ interruptionReason: "unexpected" });

    const history = sanitizeOutboundHistory([
      interrupted,
      { ...userMessage(), id: "turn-2", content: "Continue" },
    ]);

    expect(history[0].content).toContain("interrupted unexpectedly");
    expect(history[0].content).not.toContain("stopped by the user");
    expect(history[1].content).toBe("Continue");
  });

  test("does not treat a marked current user message as historical", () => {
    const current = userMessage({ interruptionReason: "user" });

    expect(sanitizeOutboundHistory([current])[0].content).toBe("What changed?");
  });
});

describe("buildOutboundHistory", () => {
  const projectReference = (name: string): AgentMessage["metadata"] => ({
    resourceReference: {
      kind: "project",
      name,
      project: name,
      label: name,
    },
  });

  const projectTurn = [
    {
      ...userMessage(projectReference("projects/billing")),
      content: "How many databases?",
    },
    {
      ...userMessage(),
      id: "call-1",
      role: "assistant" as const,
      content: undefined,
      toolCalls: [{ id: "tool-1", name: "call_api", arguments: "{}" }],
    },
    {
      ...userMessage(),
      id: "result-1",
      role: "tool" as const,
      toolCallId: "tool-1",
      content: '{"databases":["billing-db"]}',
    },
    {
      ...userMessage(),
      id: "answer-1",
      role: "assistant" as const,
      content: "Billing has one database.",
    },
  ] satisfies AgentMessage[];

  test("keeps an unqualified workspace follow-up free of the prior project turn", () => {
    const current = {
      ...userMessage(),
      id: "turn-2",
      content: "How many instances?",
    };

    expect(
      buildOutboundHistory(
        "system",
        [...projectTurn, current],
        "workspaces/default"
      )
    ).toEqual([
      { role: "system", content: "system" },
      {
        role: "user",
        content: "How many instances?",
        toolCallId: undefined,
        toolCalls: undefined,
      },
    ]);
  });

  test("keeps an unqualified project follow-up free of a database in another project", () => {
    const databaseTurn = [
      {
        ...userMessage({
          resourceReference: {
            kind: "database" as const,
            name: "instances/prod/databases/billing",
            project: "projects/billing",
            label: "billing",
          },
        }),
        content: "Inspect @billing",
      },
      {
        ...userMessage(),
        id: "answer-1",
        role: "assistant" as const,
        content: "Billing is healthy.",
      },
      { ...userMessage(), id: "turn-2", content: "How many databases?" },
    ];

    expect(
      buildOutboundHistory("system", databaseTurn, "projects/orders").map(
        (message) => message.content
      )
    ).toEqual(["system", "How many databases?"]);
  });

  test("keeps an unqualified project follow-up free of a selected database in that project", () => {
    const databaseTurn = [
      {
        ...userMessage({
          resourceReference: {
            kind: "database" as const,
            name: "instances/prod/databases/orders",
            project: "projects/orders",
            label: "orders",
          },
        }),
        content: "Inspect @orders",
      },
      {
        ...userMessage(),
        id: "answer-1",
        role: "assistant" as const,
        content: "Orders is active.",
      },
      { ...userMessage(), id: "turn-2", content: "What needs attention?" },
    ];

    expect(
      buildOutboundHistory("system", databaseTurn, "projects/orders").map(
        (message) => message.content
      )
    ).toEqual(["system", "What needs attention?"]);
  });

  test("keeps an unqualified project follow-up free of a selected issue in that project", () => {
    const issueTurn = [
      {
        ...userMessage({
          resourceReference: {
            kind: "issue" as const,
            name: "projects/orders/issues/12",
            project: "projects/orders",
            label: "Change orders table",
          },
        }),
        content: "Review @Change orders table",
      },
      {
        ...userMessage(),
        id: "answer-1",
        role: "assistant" as const,
        content: "The issue is waiting for approval.",
      },
      { ...userMessage(), id: "turn-2", content: "What needs attention?" },
    ];

    expect(
      buildOutboundHistory("system", issueTurn, "projects/orders").map(
        (message) => message.content
      )
    ).toEqual(["system", "What needs attention?"]);
  });

  test("keeps an unqualified workspace follow-up free of a selected project and issue", () => {
    const issueTurn = [
      userMessage({
        resourceReferences: [
          {
            kind: "project" as const,
            name: "projects/orders",
            project: "projects/orders",
            label: "Orders",
          },
          {
            kind: "issue" as const,
            name: "projects/orders/issues/12",
            project: "projects/orders",
            label: "Change orders table",
          },
        ],
      }),
      {
        ...userMessage(),
        id: "answer-1",
        role: "assistant" as const,
        content: "The issue is waiting for approval.",
      },
      { ...userMessage(), id: "turn-2", content: "What needs attention?" },
    ];

    expect(
      buildOutboundHistory("system", issueTurn, "workspaces/default").map(
        (message) => message.content
      )
    ).toEqual(["system", "What needs attention?"]);
  });

  test("keeps an unqualified follow-up free of a project and database pair", () => {
    const multiReferenceTurn = [
      userMessage({
        resourceReferences: [
          {
            kind: "project" as const,
            name: "projects/billing",
            project: "projects/billing",
            label: "Billing",
          },
          {
            kind: "database" as const,
            name: "instances/prod/databases/billing",
            project: "projects/billing",
            label: "billing-db",
          },
        ],
      }),
      {
        ...userMessage(),
        id: "answer-1",
        role: "assistant" as const,
        content: "Billing database is active.",
      },
      { ...userMessage(), id: "turn-2", content: "How many databases?" },
    ];

    expect(
      buildOutboundHistory(
        "system",
        multiReferenceTurn,
        "workspaces/default"
      ).map((message) => message.content)
    ).toEqual(["system", "How many databases?"]);
  });

  test("keeps unrelated chat-context turns while dropping an off-context project turn", () => {
    const earlier = [
      {
        ...userMessage(),
        id: "workspace-question",
        content: "What is this workspace?",
      },
      {
        ...userMessage(),
        id: "workspace-answer",
        role: "assistant" as const,
        content: "Default workspace.",
      },
    ];
    const current = {
      ...userMessage(),
      id: "turn-2",
      content: "How many instances?",
    };
    const history = buildOutboundHistory(
      "system",
      [...earlier, ...projectTurn, current],
      "workspaces/default"
    );

    expect(history.map((message) => message.content)).toEqual([
      "system",
      "What is this workspace?",
      "Default workspace.",
      "How many instances?",
    ]);
  });

  test("retains a same-context project turn after a contextless chat was promoted", () => {
    const history = buildOutboundHistory(
      "system",
      [
        ...projectTurn,
        { ...userMessage(), id: "turn-2", content: "How many instances?" },
      ],
      "projects/billing"
    );

    expect(history).toHaveLength(projectTurn.length + 2);
    expect(history[1].content).toContain('"name":"projects/billing"');
    expect(history.at(-1)?.content).toBe("How many instances?");
  });

  test("retains the previous project question for an explicit comparative @ project follow-up", () => {
    const current = {
      ...userMessage(projectReference("projects/orders")),
      id: "turn-2",
      content: "How about @Orders?",
    };
    const history = buildOutboundHistory(
      "system",
      [...projectTurn, current],
      "workspaces/default"
    );

    expect(history).toHaveLength(projectTurn.length + 2);
    expect(history[1].content).toContain('"name":"projects/billing"');
    expect(history.at(-1)?.content).toContain('"name":"projects/orders"');
  });

  test("retains a project explicitly named in the current unqualified message", () => {
    const history = buildOutboundHistory(
      "system",
      [
        ...projectTurn,
        {
          ...userMessage(),
          id: "turn-2",
          content: "How many instances are in Orders?",
        },
      ],
      "workspaces/default"
    );

    expect(history.at(-1)?.content).toBe("How many instances are in Orders?");
  });

  test("keeps the interruption cause without leaking an off-context request", () => {
    const interruptedProjectTurn = [
      userMessage({
        interruptionReason: "user",
        resourceReference: {
          kind: "project" as const,
          name: "projects/billing",
          project: "projects/billing",
          label: "Billing",
        },
      }),
      {
        ...userMessage(),
        id: "call-1",
        role: "assistant" as const,
        toolCalls: [{ id: "tool-1", name: "call_api", arguments: "{}" }],
      },
      { ...userMessage(), id: "turn-2", content: "How many instances?" },
    ];

    const history = buildOutboundHistory(
      "system",
      interruptedProjectTurn,
      "workspaces/default"
    );
    expect(history.map((message) => message.content)).toEqual([
      "system",
      expect.stringContaining("stopped by the user"),
      "How many instances?",
    ]);
    expect(history[1].content).not.toContain("projects/billing");
    expect(history[1].content).not.toContain("What changed?");
  });
});
