import { Code, ConnectError } from "@connectrpc/connect";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { AIChatMessageRole } from "@/types/proto-es/v1/ai_service_pb";
import type { Conversation, Message } from "../types";

const mocks = vi.hoisted(() => ({
  chat: vi.fn(),
  createConversation: vi.fn(),
  createMessage: vi.fn(),
  updateMessage: vi.fn(),
  emit: vi.fn(),
  setShowHistoryDialog: vi.fn(),
  setPendingSendChat: vi.fn(),
  currentConversation: undefined as Conversation | undefined,
  context: undefined as Record<string, unknown> | undefined,
}));

vi.mock("@/api", () => ({
  aiServiceClientConnect: { chat: mocks.chat },
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  getCurrentSQLEditorTab: () => ({
    connection: {
      instance: "instances/test",
      database: "instances/test/databases/demo",
    },
  }),
  useCurrentSQLEditorTab: () => ({
    connection: {
      instance: "instances/test",
      database: "instances/test/databases/demo",
    },
  }),
}));

vi.mock("../store", () => ({
  useConversationStore: () => ({
    createConversation: mocks.createConversation,
    createMessage: mocks.createMessage,
    updateMessage: mocks.updateMessage,
  }),
}));

vi.mock("./context", () => ({
  useAIContext: () => mocks.context,
}));

vi.mock("../logic/prompt", () => ({
  declaration: () => "schema context",
}));

vi.mock("@/utils", () => ({
  nextAnimationFrame: () => Promise.resolve(),
}));

vi.mock("./ActionBar", () => ({ ActionBar: () => null }));
vi.mock("./ChatView/ChatView", () => ({ ChatView: () => null }));
vi.mock("./DynamicSuggestions", () => ({
  DynamicSuggestions: () => null,
}));
vi.mock("./HistoryPanel/HistoryPanel", () => ({ HistoryPanel: () => null }));
vi.mock("./PromptInput", () => ({
  PromptInput: ({ onEnter }: { onEnter: (value: string) => void }) => (
    <button type="button" onClick={() => onEnter("show me the tables")}>
      send
    </button>
  ),
}));

import { ChatPanel } from "./ChatPanel";

const emptyConversation = (): Conversation => ({
  id: "conversation-1",
  created_ts: 1,
  name: "",
  instance: "instances/test",
  database: "instances/test/databases/demo",
  messageList: [],
});

beforeEach(() => {
  const conversation = emptyConversation();
  mocks.currentConversation = conversation;
  mocks.chat.mockReset().mockResolvedValue({ content: "SELECT 1" });
  mocks.createConversation.mockReset();
  mocks.updateMessage.mockReset().mockResolvedValue(undefined);
  mocks.emit.mockReset();
  mocks.setShowHistoryDialog.mockReset();
  mocks.setPendingSendChat.mockReset();
  mocks.createMessage.mockReset().mockImplementation(async (input) => {
    const current = mocks.currentConversation!;
    const nextConversation: Conversation = {
      ...current,
      messageList: [...current.messageList],
    };
    const message: Message = {
      id: `message-${nextConversation.messageList.length + 1}`,
      created_ts: nextConversation.messageList.length + 1,
      ...input,
      conversation: nextConversation,
    };
    nextConversation.messageList.push(message);
    mocks.currentConversation = nextConversation;
    return message;
  });
  mocks.context = {
    aiSetting: { enabled: true },
    engine: undefined,
    databaseMetadata: undefined,
    schema: undefined,
    chat: {
      list: [conversation],
      ready: true,
      selected: conversation,
      setSelected: vi.fn(),
    },
    showHistoryDialog: false,
    setShowHistoryDialog: mocks.setShowHistoryDialog,
    pendingSendChat: undefined,
    setPendingSendChat: mocks.setPendingSendChat,
    pendingPreInput: undefined,
    setPendingPreInput: vi.fn(),
    events: { emit: mocks.emit },
  };
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("ChatPanel", () => {
  test("includes the newly persisted user message in the chat request", async () => {
    render(<ChatPanel />);

    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(mocks.chat).toHaveBeenCalledTimes(1));
    expect(mocks.chat.mock.calls[0]?.[0].messages).toEqual([
      expect.objectContaining({
        role: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_USER,
        content: "schema context\nshow me the tables",
      }),
    ]);
    expect(mocks.createMessage).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ content: "show me the tables" })
    );
    expect(mocks.createMessage.mock.calls[0]?.[0]).not.toHaveProperty("prompt");
  });

  test("rebuilds schema context for a conversation restored from storage", async () => {
    const conversation = emptyConversation();
    const firstUser: Message = {
      id: "message-1",
      created_ts: 1,
      author: "USER",
      content: "first question",
      status: "DONE",
      error: "",
      conversation,
    };
    const firstAnswer: Message = {
      id: "message-2",
      created_ts: 2,
      author: "AI",
      content: "first answer",
      status: "DONE",
      error: "",
      conversation,
    };
    conversation.messageList = [firstUser, firstAnswer];
    mocks.currentConversation = conversation;
    mocks.context = {
      ...mocks.context,
      chat: {
        ...(mocks.context?.chat as object),
        list: [conversation],
        selected: conversation,
      },
    };

    render(<ChatPanel />);
    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(mocks.chat).toHaveBeenCalledTimes(1));
    expect(mocks.chat.mock.calls[0]?.[0].messages).toEqual([
      expect.objectContaining({ content: "schema context\nfirst question" }),
      expect.objectContaining({ content: "first answer" }),
      expect.objectContaining({ content: "show me the tables" }),
    ]);
  });

  test("logs the original Connect error when the chat request fails", async () => {
    const error = new ConnectError("provider unavailable", Code.Internal);
    mocks.chat.mockRejectedValue(error);
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    render(<ChatPanel />);

    fireEvent.click(screen.getByRole("button", { name: "send" }));

    await waitFor(() => expect(mocks.updateMessage).toHaveBeenCalledTimes(1));
    expect(consoleError).toHaveBeenCalledWith(
      "[AI Assistant] chat failed:",
      error
    );
    expect(mocks.updateMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        status: "FAILED",
        error: "ConnectError: [internal] provider unavailable",
      })
    );
  });
});
