import { beforeEach, expect, test, vi } from "vitest";
import { captureAgentPageSnapshot } from "@/modules/agent/logic/capturePage";
import { useAgentStore } from "@/modules/agent/store/agent";
import { handoffSqlEditorAction } from "./agentHandoff";

vi.mock("@/modules/agent/logic/capturePage", () => ({
  captureAgentPageSnapshot: vi.fn(),
}));

beforeEach(() => {
  localStorage.clear();
  useAgentStore.setState(useAgentStore.getInitialState(), true);
  useAgentStore
    .getState()
    .setWorkspaceName("workspaces/demo", "member@example.com");
  vi.mocked(captureAgentPageSnapshot).mockReturnValue({
    path: "/sql-editor",
    title: "SQL Editor",
    sqlEditor: { version: 1, project: "projects/demo" },
  });
});

test("a second action leaves the queued prompt intact and opens another normal chat", () => {
  expect(handoffSqlEditorAction("explain-code", "SELECT 1", "Explain")).toBe(
    true
  );
  const first = useAgentStore.getState().currentChatId!;
  expect(handoffSqlEditorAction("find-problems", "SELECT 2", "Check")).toBe(
    true
  );
  const store = useAgentStore.getState();
  const second = store.currentChatId!;
  expect(second).not.toBe(first);
  expect(store.queuedUserMessagesByChatId[first]?.text).toBe("Explain");
  expect(store.queuedUserMessagesByChatId[second]?.text).toBe("Check");
  expect(store.getChat(second)?.capability).toBeUndefined();
});

test.each(["explain-code", "find-problems"] as const)(
  "queues %s with only explicit SQL on the source page",
  (action) => {
    expect(handoffSqlEditorAction(action, "SELECT 1;", "Check statement")).toBe(
      true
    );
    const store = useAgentStore.getState();
    const id = store.currentChatId!;
    expect(store.getChat(id)?.context).toBe("projects/demo");
    expect(store.queuedUserMessagesByChatId[id]).toMatchObject({
      text: "Check statement",
      page: {
        sqlEditor: {
          attachment: {
            kind: action === "explain-code" ? "explain" : "find-problems",
            sql: "SELECT 1;",
          },
        },
      },
    });
    expect(store.visible).toBe(true);
  }
);

test("new-chat selection starts a new normal chat even if the current conversation is idle", () => {
  const store = useAgentStore.getState();
  const previous = store.createChat({
    context: "projects/demo",
    workspaceName: "workspaces/demo",
  });
  store.addMessage({
    chatId: previous.id,
    role: "user",
    content: "Earlier question",
  });
  expect(handoffSqlEditorAction("new-chat", "SELECT 7", "")).toBe(true);
  const next = useAgentStore.getState().currentChatId!;
  expect(next).not.toBe(previous.id);
  expect(store.getDraft(previous.id).text).toBe("");
  expect(store.getMessages(next)).toHaveLength(0);
});

test("selection starts a frozen unsent draft and preserves an existing draft", () => {
  expect(handoffSqlEditorAction("new-chat", "SELECT 1", "")).toBe(true);
  const first = useAgentStore.getState().currentChatId!;
  expect(handoffSqlEditorAction("new-chat", "SELECT 2", "")).toBe(true);
  const store = useAgentStore.getState();
  const second = store.currentChatId!;
  expect(second).not.toBe(first);
  expect(store.getDraft(first).text).toBe("SELECT 1");
  expect(store.getDraft(second)).toMatchObject({
    text: "SELECT 2",
    queuedSqlPage: {
      sqlEditor: { attachment: { kind: "selection", sql: "SELECT 2" } },
    },
  });
  expect(Object.keys(store.queuedUserMessagesByChatId)).toHaveLength(0);
});

test("rejects oversized UTF-8 SQL and pending context before opening a pane", () => {
  expect(
    handoffSqlEditorAction("explain-code", "字".repeat(11000), "Explain")
  ).toBe(false);
  vi.mocked(captureAgentPageSnapshot).mockReturnValue(undefined);
  expect(handoffSqlEditorAction("find-problems", "SELECT 2", "Check")).toBe(
    false
  );
  expect(useAgentStore.getState().visible).toBe(false);
});
