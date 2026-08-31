import { beforeEach, describe, expect, test, vi } from "vitest";
import { useAppStore } from "@/stores/app";
import { useConversationStore } from "./conversation";
import { saveConversationHistory } from "./conversationStorage";

const connection = {
  instance: "instances/test",
  database: "instances/test/databases/demo",
};

const storageKey = "bb.ai.conversations.workspaces/test.ai-user@example.com";

const storedConversation = (id: string) => {
  const conversation = {
    id,
    created_ts: 1,
    name: id,
    ...connection,
    messageList: [],
  } as ReturnType<
    typeof useConversationStore.getState
  >["conversationsById"][string];
  conversation.messageList = [
    {
      id: `${id}-message`,
      created_ts: 1,
      author: "USER",
      content: id,
      status: "DONE",
      error: "",
      conversation,
    },
  ];
  return conversation;
};

beforeEach(() => {
  localStorage.clear();
  vi.restoreAllMocks();
  useAppStore.setState({
    currentUser: {
      email: "ai-user@example.com",
      workspace: "workspaces/test",
    } as never,
    serverInfo: { saas: true } as never,
  });
  useConversationStore.setState({
    conversationsById: {},
    readyByConnection: {},
    storageKey: undefined,
  });
});

describe("AI conversation persistence", () => {
  test("restores a conversation after the in-memory store is reset", async () => {
    const conversation = await useConversationStore
      .getState()
      .createConversation({ ...connection, name: "saved" });
    await useConversationStore.getState().createMessage({
      author: "USER",
      content: "visible question",
      error: "",
      status: "DONE",
      conversation_id: conversation.id,
    });
    useConversationStore.setState({
      conversationsById: {},
      readyByConnection: {},
      storageKey: undefined,
    });

    const conversations = await useConversationStore
      .getState()
      .fetchConversationListByConnection(connection as never);

    expect(conversations).toHaveLength(1);
    expect(conversations[0]?.messageList[0]?.content).toBe("visible question");
    expect(localStorage.getItem(storageKey)).toContain("visible question");
  });

  test("removing an empty conversation during reload persists the deletion", async () => {
    await useConversationStore.getState().createConversation({
      ...connection,
      name: "",
    });
    useConversationStore.setState({
      conversationsById: {},
      readyByConnection: {},
      storageKey: undefined,
    });

    const conversations = await useConversationStore
      .getState()
      .fetchConversationListByConnection(connection as never);

    expect(conversations).toEqual([]);
    expect(useConversationStore.getState().conversationsById).toEqual({});
    expect(localStorage.getItem(storageKey)).not.toContain(connection.database);
  });

  test("reload marks an interrupted response as failed", async () => {
    const conversation = await useConversationStore
      .getState()
      .createConversation({ ...connection, name: "" });
    await useConversationStore.getState().createMessage({
      author: "AI",
      content: "",
      error: "",
      status: "LOADING",
      conversation_id: conversation.id,
    });
    useConversationStore.setState({
      conversationsById: {},
      readyByConnection: {},
      storageKey: undefined,
    });

    const conversations = await useConversationStore
      .getState()
      .fetchConversationListByConnection(connection as never);

    expect(conversations[0]?.messageList[0]).toMatchObject({
      status: "FAILED",
      error: "Request timeout",
    });
  });

  test("keeps chat usable when local storage writes fail", async () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("Storage quota exceeded", "QuotaExceededError");
    });

    const conversation = await useConversationStore
      .getState()
      .createConversation({ ...connection, name: "memory only" });
    const message = await useConversationStore.getState().createMessage({
      author: "USER",
      content: "SELECT 1",
      error: "",
      status: "DONE",
      conversation_id: conversation.id,
    });

    expect(message.content).toBe("SELECT 1");
    expect(
      useConversationStore.getState().conversationsById[conversation.id]
        ?.messageList
    ).toHaveLength(1);
  });

  test("does not write stale conversations into a different user key", async () => {
    const conversation = await useConversationStore
      .getState()
      .createConversation({ ...connection, name: "user A" });
    const message = await useConversationStore.getState().createMessage({
      author: "USER",
      content: "user A question",
      error: "",
      status: "DONE",
      conversation_id: conversation.id,
    });
    const userBKey = "bb.ai.conversations.workspaces/test.user-b@example.com";
    saveConversationHistory(userBKey, [storedConversation("user-b-history")]);
    const userBHistory = localStorage.getItem(userBKey);
    useAppStore.setState({
      currentUser: {
        email: "user-b@example.com",
        workspace: "workspaces/test",
      } as never,
    });

    message.content = "late user A response";
    await useConversationStore.getState().updateMessage(message);

    expect(localStorage.getItem(userBKey)).toBe(userBHistory);
  });

  test("hydrates the new identity before creating a conversation", async () => {
    const userA = await useConversationStore
      .getState()
      .createConversation({ ...connection, name: "user A" });
    await useConversationStore.getState().createMessage({
      author: "USER",
      content: "user A question",
      error: "",
      status: "DONE",
      conversation_id: userA.id,
    });
    const userBKey = "bb.ai.conversations.workspaces/test.user-b@example.com";
    saveConversationHistory(userBKey, [storedConversation("user-b-history")]);
    useAppStore.setState({
      currentUser: {
        email: "user-b@example.com",
        workspace: "workspaces/test",
      } as never,
    });

    await useConversationStore
      .getState()
      .createConversation({ ...connection, name: "new conversation" });

    const userBHistory = localStorage.getItem(userBKey) ?? "";
    expect(userBHistory).toContain("user-b-history");
    expect(userBHistory).not.toContain("user A question");
  });

  test("always includes the workspace in self-host storage keys", async () => {
    useAppStore.setState({
      currentUser: {
        email: "ai-user@example.com",
        workspace: "workspaces/self-host",
      } as never,
      serverInfo: { saas: false, workspace: "workspaces/self-host" } as never,
    });

    const conversation = await useConversationStore
      .getState()
      .createConversation({ ...connection, name: "self-host" });
    await useConversationStore.getState().createMessage({
      author: "USER",
      content: "self-host question",
      error: "",
      status: "DONE",
      conversation_id: conversation.id,
    });

    expect(
      localStorage.getItem(
        "bb.ai.conversations.workspaces/self-host.ai-user@example.com"
      )
    ).toContain("self-host question");
    expect(
      localStorage.getItem("bb.ai.conversations.ai-user@example.com")
    ).toBeNull();
  });
});
