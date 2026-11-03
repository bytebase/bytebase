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

  test("summarizes connection failures without sharing raw error or form data", () => {
    const prompt = buildInstanceConnectionHelpPrompt({
      engine: "PostgreSQL",
      stage: "connection test",
      failureCategory: "network_unreachable",
    });

    expect(prompt).toContain("PostgreSQL");
    expect(prompt).toContain("database endpoint could not be reached");
    expect(prompt).toContain("passwords, tokens, private keys");
    expect(prompt).not.toContain("host.example.com");
    expect(prompt).not.toContain("i/o timeout");
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
});
