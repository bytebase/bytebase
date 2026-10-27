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
});
