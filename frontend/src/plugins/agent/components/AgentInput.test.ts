import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import { useAgentStore } from "../store/agent";
import AgentInput from "./AgentInput.vue";

const {
  mockRunAgentLoop,
  mockBuildSystemPrompt,
  mockCreateToolExecutor,
  mockGetToolDefinitions,
} = vi.hoisted(() => ({
  mockRunAgentLoop: vi.fn(),
  mockBuildSystemPrompt: vi.fn(() => "system-prompt"),
  mockCreateToolExecutor: vi.fn(() => vi.fn()),
  mockGetToolDefinitions: vi.fn(() => []),
}));

vi.mock("../logic/agentLoop", () => ({
  runAgentLoop: mockRunAgentLoop,
}));

vi.mock("../logic/prompt", () => ({
  buildSystemPrompt: mockBuildSystemPrompt,
}));

vi.mock("../logic/tools", () => ({
  createToolExecutor: mockCreateToolExecutor,
  getToolDefinitions: mockGetToolDefinitions,
}));

vi.mock("vue-router", () => ({
  useRoute: () => ({ fullPath: "/projects/demo" }),
  useRouter: () => ({}),
}));

const flushPromises = async () => {
  await Promise.resolve();
  await Promise.resolve();
};

const i18n = createI18n({
  legacy: false,
  locale: "en-US",
  messages: {
    "en-US": {
      agent: {
        interrupted: "Interrupted",
        "input-placeholder": "Ask anything...",
        send: "Send",
        reply: "Reply",
        stop: "Stop",
        confirm: "Confirm",
        cancel: "Cancel",
        "pending-input-hint": "Reply below to continue this thread.",
        "pending-confirm-hint":
          "Choose confirm or cancel to continue this thread.",
        "pending-choose-hint": "Choose an option to continue this thread.",
      },
    },
  },
});

describe("AgentInput", () => {
  let pinia: ReturnType<typeof createPinia>;

  beforeEach(() => {
    localStorage.clear();
    pinia = createPinia();
    setActivePinia(pinia);
    mockRunAgentLoop.mockReset();
    mockBuildSystemPrompt.mockClear();
    mockCreateToolExecutor.mockClear();
    mockGetToolDefinitions.mockClear();
  });

  test("submits pending input as a tool result and resumes the same thread", async () => {
    const store = useAgentStore();
    const threadId = store.currentThreadId!;

    store.addMessage({
      threadId,
      role: "assistant",
      toolCalls: [
        {
          id: "tool-ask",
          name: "ask_user",
          arguments: JSON.stringify({
            prompt: "Which project should I use?",
            kind: "input",
          }),
        },
      ],
    });
    store.awaitUser(threadId, {
      toolCallId: "tool-ask",
      prompt: "Which project should I use?",
      kind: "input",
    });

    mockRunAgentLoop.mockImplementation(
      async (
        messages: unknown,
        _tools: unknown,
        _executor: unknown,
        callbacks?: { onText?: (text: string) => void }
      ) => {
        callbacks?.onText?.("Using project demo.");
        return {
          kind: "completed",
          text: "Using project demo.",
          success: true,
          explicit: true,
        };
      }
    );

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    await wrapper.find("textarea").setValue("demo-project");
    await wrapper.find("button").trigger("click");
    await flushPromises();

    const toolMessages = store
      .getMessages(threadId)
      .filter((message) => message.role === "tool");
    expect(toolMessages).toHaveLength(1);
    expect(toolMessages[0].toolCallId).toBe("tool-ask");
    expect(JSON.parse(toolMessages[0].content ?? "{}")).toEqual({
      kind: "input",
      answer: "demo-project",
    });
    expect(store.getPendingAsk(threadId)).toBeNull();
    expect(store.currentThreadId).toBe(threadId);
    expect(mockRunAgentLoop).toHaveBeenCalledTimes(1);

    const [messages] = mockRunAgentLoop.mock.calls[0];
    expect(messages).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ role: "system", content: "system-prompt" }),
        expect.objectContaining({
          role: "tool",
          toolCallId: "tool-ask",
        }),
      ])
    );
  });

  test("uses choose buttons to answer pending choose prompts", async () => {
    const store = useAgentStore();
    const threadId = store.currentThreadId!;

    store.addMessage({
      threadId,
      role: "assistant",
      toolCalls: [
        {
          id: "tool-choose",
          name: "ask_user",
          arguments: JSON.stringify({
            prompt: "Which environment should I use?",
            kind: "choose",
            options: [
              {
                label: "Production",
                value: "prod",
                description: "Use the production environment",
              },
              {
                label: "Staging",
                value: "staging",
              },
            ],
          }),
        },
      ],
    });
    store.awaitUser(threadId, {
      toolCallId: "tool-choose",
      prompt: "Which environment should I use?",
      kind: "choose",
      options: [
        {
          label: "Production",
          value: "prod",
          description: "Use the production environment",
        },
        {
          label: "Staging",
          value: "staging",
        },
      ],
    });

    mockRunAgentLoop.mockResolvedValue({
      kind: "completed",
      text: "Using production.",
      success: true,
      explicit: true,
    });

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    const buttons = wrapper.findAll("button");
    expect(buttons).toHaveLength(2);
    await buttons[0].trigger("click");
    await flushPromises();

    const toolMessages = store
      .getMessages(threadId)
      .filter((message) => message.role === "tool");
    expect(toolMessages).toHaveLength(1);
    expect(JSON.parse(toolMessages[0].content ?? "{}")).toEqual({
      kind: "choose",
      answer: "Production",
      value: "prod",
    });
    expect(store.getPendingAsk(threadId)).toBeNull();
    expect(mockRunAgentLoop).toHaveBeenCalledTimes(1);
  });

  test("uses confirm buttons to answer pending confirmation prompts", async () => {
    const store = useAgentStore();
    const threadId = store.currentThreadId!;

    store.addMessage({
      threadId,
      role: "assistant",
      toolCalls: [
        {
          id: "tool-confirm",
          name: "ask_user",
          arguments: JSON.stringify({
            prompt: "Delete the database?",
            kind: "confirm",
            confirmLabel: "Delete",
            cancelLabel: "Keep it",
          }),
        },
      ],
    });
    store.awaitUser(threadId, {
      toolCallId: "tool-confirm",
      prompt: "Delete the database?",
      kind: "confirm",
      confirmLabel: "Delete",
      cancelLabel: "Keep it",
    });

    mockRunAgentLoop.mockResolvedValue({
      kind: "completed",
      text: "Canceled.",
      success: true,
      explicit: true,
    });

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    const buttons = wrapper.findAll("button");
    expect(buttons).toHaveLength(2);
    await buttons[0].trigger("click");
    await flushPromises();

    const toolMessages = store
      .getMessages(threadId)
      .filter((message) => message.role === "tool");
    expect(toolMessages).toHaveLength(1);
    expect(JSON.parse(toolMessages[0].content ?? "{}")).toEqual({
      kind: "confirm",
      answer: "Delete",
      confirmed: true,
    });
    expect(store.getPendingAsk(threadId)).toBeNull();
    expect(mockRunAgentLoop).toHaveBeenCalledTimes(1);
  });
});
