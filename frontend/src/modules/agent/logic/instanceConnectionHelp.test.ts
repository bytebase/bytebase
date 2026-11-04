import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  createChat: vi.fn(() => ({ id: "chat-1" })),
  queueUserMessage: vi.fn(),
  open: vi.fn(),
  getCurrentChatContext: vi.fn(() => "projects/sample"),
  app: {
    workspace: { name: "workspaces/default" },
    currentUser: undefined,
    serverInfo: undefined,
  },
  route: {
    fullPath: "/projects/sample/instances/create",
  },
}));

vi.mock("@/app/router", () => ({
  router: {
    currentRoute: {
      get value() {
        return mocks.route;
      },
    },
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => mocks.app,
  },
}));

vi.mock("../store/agent", () => ({
  useAgentStore: {
    getState: () => ({
      createChat: mocks.createChat,
      queueUserMessage: mocks.queueUserMessage,
      open: mocks.open,
    }),
  },
}));

vi.mock("./chatContext", () => ({
  getCurrentChatContext: mocks.getCurrentChatContext,
}));

import {
  buildInstanceConnectionHelpPrompt,
  openAgentWithPrompt,
} from "./instanceConnectionHelp";

describe("instanceConnectionHelp", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.createChat.mockReturnValue({ id: "chat-1" });
    mocks.getCurrentChatContext.mockReturnValue("projects/sample");
    mocks.app.workspace = { name: "workspaces/default" };
    mocks.route = { fullPath: "/projects/sample/instances/create" };
  });

  test("includes sanitized error details and tells the agent not to ask for them again", () => {
    const prompt = buildInstanceConnectionHelpPrompt({
      engine: "PostgreSQL",
      stage: "connection test",
      failureCategory: "network_unreachable",
      failureMessage:
        "dial tcp db.internal.example.com:5432: i/o timeout; password=top-secret; Authorization: Bearer secret-token; mysql://user:dsn-secret@db.internal.example.com:3306/app",
    });

    expect(prompt).toContain("PostgreSQL");
    expect(prompt).toContain("database endpoint could not be reached");
    expect(prompt).toContain(
      "dial tcp db.internal.example.com:5432: i/o timeout"
    );
    expect(prompt).toContain("password=[redacted]");
    expect(prompt).toContain("Bearer [redacted]");
    expect(prompt).toContain("[connection string redacted]");
    expect(prompt).toContain("Do not ask me to paste or repeat the error");
    expect(prompt).toContain(
      "inspect the current page's visible non-secret connection settings"
    );
    expect(prompt).toContain("passwords, tokens, private keys");
    expect(prompt).not.toContain("top-secret");
    expect(prompt).not.toContain("secret-token");
    expect(prompt).not.toContain("dsn-secret");
    expect(prompt).not.toContain("have not been shared");
  });

  test("redacts credentials in HTTPS connection URLs", () => {
    const prompt = buildInstanceConnectionHelpPrompt({
      engine: "Trino",
      stage: "connection test",
      failureCategory: "ssl_tls_failed",
      failureMessage:
        "TLS handshake failed for https://dbadmin:very-secret@trino.example.com:8443/catalog",
    });

    expect(prompt).toContain(
      "https://[redacted]@trino.example.com:8443/catalog"
    );
    expect(prompt).not.toContain("dbadmin");
    expect(prompt).not.toContain("very-secret");
  });

  test("opens a new read-only Agent chat in the active project context", () => {
    openAgentWithPrompt("Help with the connection failure.");

    expect(mocks.createChat).toHaveBeenCalledWith({
      page: {
        path: "/projects/sample/instances/create",
        title: document.title,
      },
      context: "projects/sample",
      workspaceName: "workspaces/default",
    });
    expect(mocks.queueUserMessage).toHaveBeenCalledWith({
      chatId: "chat-1",
      text: "Help with the connection failure.",
      page: {
        path: "/projects/sample/instances/create",
        title: document.title,
      },
      runMode: "read-only",
    });
    expect(mocks.open).toHaveBeenCalledOnce();
  });

  test("binds page-launched assistance to its selected resource", () => {
    const resourceReference = {
      kind: "plan" as const,
      name: "projects/sample/plans/123",
      project: "projects/sample",
      label: "Add event source tracking",
    };

    openAgentWithPrompt("Review this Plan.", [resourceReference]);

    expect(mocks.queueUserMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        chatId: "chat-1",
        text: "Review this Plan.",
        resourceReferences: [resourceReference],
        runMode: "read-only",
      })
    );
  });
});
