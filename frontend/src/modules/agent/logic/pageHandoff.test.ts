import { beforeEach, describe, expect, test } from "vitest";
import { createAgentStore, useAgentStore } from "../store/agent";
import {
  captureAgentPageHandoffContext,
  minimizeAgentAfterPageHandoff,
} from "./pageHandoff";

beforeEach(() => {
  const initial = createAgentStore().getState();
  useAgentStore.setState({
    chats: initial.chats,
    messagesByChatId: initial.messagesByChatId,
    pendingAskByChatId: initial.pendingAskByChatId,
    draftsByChatId: initial.draftsByChatId,
    currentChatId: initial.currentChatId,
    workspaceName: "workspaces/example",
    userEmail: "member@example.com",
    visible: true,
    minimized: false,
    abortControllersByChatId: {},
  });
});

describe("page handoff", () => {
  test("minimizes without interrupting or clearing the source conversation", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/example" });
    store.addMessage({
      chatId: chat.id,
      role: "user",
      content: "Help me configure a rule.",
    });
    store.setDraft(chat.id, { text: "Keep this draft" });
    const controller = new AbortController();
    store.setAbortController(chat.id, controller);
    const messages = store.getMessages(chat.id);
    const context = captureAgentPageHandoffContext(chat.id);
    expect(context?.chatId).toBe(chat.id);
    expect(useAgentStore.getState().minimized).toBe(false);
    minimizeAgentAfterPageHandoff(context);
    const current = useAgentStore.getState();
    expect(current.visible).toBe(true);
    expect(current.minimized).toBe(true);
    expect(current.currentChatId).toBe(chat.id);
    expect(current.getMessages(chat.id)).toEqual(messages);
    expect(current.getDraft(chat.id).text).toBe("Keep this draft");
    expect(controller.signal.aborted).toBe(false);
  });

  test.each(["chat", "workspace", "account", "closed"] as const)(
    "does not minimize after the %s changes",
    (change) => {
      const store = useAgentStore.getState();
      store.createChat({ context: "projects/example" });
      const context = captureAgentPageHandoffContext();
      if (change === "chat") store.createChat({ context: "projects/another" });
      if (change === "workspace")
        useAgentStore.setState({ workspaceName: "workspaces/another" });
      if (change === "account")
        useAgentStore.setState({ userEmail: "another@example.com" });
      if (change === "closed") useAgentStore.setState({ visible: false });
      minimizeAgentAfterPageHandoff(context);
      expect(useAgentStore.getState().minimized).toBe(false);
    }
  );

  test("does not capture another chat or a closed window", () => {
    const store = useAgentStore.getState();
    const first = store.createChat({ context: "projects/example" });
    store.createChat({ context: "projects/another" });
    expect(captureAgentPageHandoffContext(first.id)).toBeUndefined();
    useAgentStore.setState({ visible: false });
    expect(captureAgentPageHandoffContext()).toBeUndefined();
    minimizeAgentAfterPageHandoff(undefined);
    expect(useAgentStore.getState().minimized).toBe(false);
  });
});
