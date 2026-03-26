import { mount } from "@vue/test-utils";
import { NButton, NMention } from "naive-ui";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import type { DomRefSuggestion } from "../dom";
import type { AgentLoopOutcome } from "../logic/types";
import { useAgentStore } from "../store/agent";
import AgentInput from "./AgentInput.vue";

const {
  mockRunAgentLoop,
  mockBuildSystemPrompt,
  mockCreateToolExecutor,
  mockGetToolDefinitions,
  mockLazyExtractDomRefSuggestions,
  mockRoute,
} = vi.hoisted(() => ({
  mockRunAgentLoop: vi.fn(),
  mockBuildSystemPrompt: vi.fn(() => "system-prompt"),
  mockCreateToolExecutor: vi.fn((_router?: unknown, _options?: unknown) =>
    vi.fn()
  ),
  mockGetToolDefinitions: vi.fn(() => []),
  mockLazyExtractDomRefSuggestions: vi.fn<() => Promise<DomRefSuggestion[]>>(
    async () => []
  ),
  mockRoute: { fullPath: "/projects/demo" },
}));

vi.mock("../dom", () => ({
  lazyExtractDomRefSuggestions: mockLazyExtractDomRefSuggestions,
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
  useRoute: () => mockRoute,
  useRouter: () => ({}),
}));

const getTextareaValue = (wrapper: ReturnType<typeof mount>) => {
  return (wrapper.find("textarea").element as HTMLTextAreaElement).value;
};
const findButtonByText = (wrapper: ReturnType<typeof mount>, text: string) => {
  return wrapper.findAll("button").find((button) => button.text() === text);
};
const flushPromises = async () => {
  await Promise.resolve();
  await Promise.resolve();
  await new Promise((resolve) => window.setTimeout(resolve, 0));
  await Promise.resolve();
};

const createDeferred = <T>() => {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((resolver) => {
    resolve = resolver;
  });
  return { promise, resolve };
};

const i18n = createI18n({
  legacy: false,
  locale: "en-US",
  messages: {
    "en-US": {
      common: {
        dismiss: "Dismiss",
      },
      agent: {
        interrupted: "Interrupted",
        "retry-last-chat-turn": "Retry last turn",
        "interrupted-retry-hint":
          "Retry reruns the interrupted turn with the current page state.",
        "input-placeholder": "Ask anything...",
        send: "Send",
        reply: "Reply",
        stop: "Stop",
        confirm: "Confirm",
        cancel: "Cancel",
        "pending-input-hint": "Reply below to continue this chat.",
        "pending-confirm-hint":
          "Choose confirm or cancel to continue this chat.",
        "pending-choose-hint": "Choose an option to continue this chat.",
      },
    },
  },
});

const createDomRefSuggestion = (
  overrides: Partial<DomRefSuggestion> = {}
): DomRefSuggestion => ({
  ref: "e1",
  tag: "BUTTON",
  label: "Suggestion",
  ...overrides,
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

  beforeEach(() => {
    mockLazyExtractDomRefSuggestions.mockReset();
    mockLazyExtractDomRefSuggestions.mockResolvedValue([]);
  });

  beforeEach(() => {
    mockRoute.fullPath = "/projects/demo";
    document.title = "Demo Page";
  });

  test("renders the composer as a Naive UI textarea with a primary send button", () => {
    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    const inputRow = wrapper.find("[data-agent-input-row]");
    expect(inputRow.exists()).toBe(true);

    const input = wrapper.findComponent(NMention);
    expect(input.exists()).toBe(true);
    expect(input.props("type")).toBe("textarea");
    expect(input.props("placement")).toBe("top-start");
    expect(input.props("to")).toBe("body");

    const sendButton = wrapper.findComponent(NButton);
    expect(sendButton.exists()).toBe(true);
    expect(sendButton.props("type")).toBe("primary");
    expect(sendButton.text()).toBe("Send");
    expect(sendButton.classes()).not.toContain("h-[38px]");
  });

  test("submits pending input as a tool result and resumes the same chat", async () => {
    const store = useAgentStore();
    const chatId = store.currentChatId!;

    store.addMessage({
      chatId,
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
    store.awaitUser(chatId, {
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
        };
      }
    );

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    await wrapper.find("textarea").setValue("demo-project");
    await findButtonByText(wrapper, "Reply")!.trigger("click");
    await flushPromises();

    const toolMessages = store
      .getMessages(chatId)
      .filter((message) => message.role === "tool");
    expect(toolMessages).toHaveLength(1);
    expect(toolMessages[0].toolCallId).toBe("tool-ask");
    expect(JSON.parse(toolMessages[0].content ?? "{}")).toEqual({
      kind: "input",
      answer: "demo-project",
    });
    expect(store.getPendingAsk(chatId)).toBeNull();
    expect(store.currentChatId).toBe(chatId);
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

  test("resumes pending asks with the current page snapshot", async () => {
    const store = useAgentStore();
    const chatId = store.currentChatId!;

    store.updateChatPage(chatId, {
      path: "/projects/original",
      title: "Original Page",
    });
    store.awaitUser(chatId, {
      toolCallId: "tool-ask",
      prompt: "Which project should I use?",
      kind: "input",
    });

    mockRoute.fullPath = "/projects/other";
    document.title = "Other Page";
    mockRunAgentLoop.mockResolvedValue({
      kind: "completed",
      text: "Using project demo.",
      success: true,
    });

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    await wrapper.find("textarea").setValue("demo-project");
    await findButtonByText(wrapper, "Reply")!.trigger("click");
    await flushPromises();

    expect(mockBuildSystemPrompt).toHaveBeenCalledWith({
      path: "/projects/other",
      title: "Other Page",
    });
    expect(store.getChat(chatId)?.page).toEqual({
      path: "/projects/other",
      title: "Other Page",
    });
  });

  test("increments the current chat token total from loop usage", async () => {
    const store = useAgentStore();
    const chatId = store.currentChatId!;

    mockRunAgentLoop.mockResolvedValue({
      kind: "completed",
      text: "Done",
      success: true,
      totalTokensUsed: 123,
    });

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    await wrapper.find("textarea").setValue("inspect this page");
    await findButtonByText(wrapper, "Send")!.trigger("click");
    await flushPromises();

    expect(store.getChat(chatId)?.totalTokensUsed).toBe(123);
  });

  test("refreshes the saved chat page snapshot after navigate before a follow-up turn", async () => {
    const store = useAgentStore();
    const chatId = store.currentChatId!;
    let runCount = 0;

    store.updateChatPage(chatId, {
      path: "/projects/original",
      title: "Original Page",
    });

    mockCreateToolExecutor.mockImplementation((_router: unknown, _options) => {
      const options = _options as { onNavigate?: () => void } | undefined;
      return vi.fn(async (name: string) => {
        if (name !== "navigate") {
          throw new Error(`Unexpected tool: ${name}`);
        }
        mockRoute.fullPath = "/projects/navigated";
        document.title = "Navigated Page";
        options?.onNavigate?.();
        return {
          kind: "tool_result",
          result: JSON.stringify({
            navigated: true,
            currentPath: "/projects/navigated",
          }),
        };
      });
    });
    mockRunAgentLoop.mockImplementation(
      async (
        _messages: unknown,
        _tools: unknown,
        executor: (
          name: string,
          args: Record<string, unknown>,
          toolCallId: string
        ) => Promise<{ kind: string; result?: string }>,
        callbacks?: {
          onAssistantMessage?: (message: {
            content?: string;
            toolCalls?: unknown[];
          }) => void;
          onToolResult?: (toolCallId: string, result: string) => void;
        }
      ) => {
        runCount += 1;
        if (runCount === 1) {
          callbacks?.onAssistantMessage?.({
            content: "",
            toolCalls: [
              {
                id: "tool-nav",
                name: "navigate",
                arguments: JSON.stringify({ path: "/projects/navigated" }),
              },
            ],
          });
          const result = await executor(
            "navigate",
            { path: "/projects/navigated" },
            "tool-nav"
          );
          if (result.kind === "tool_result" && result.result) {
            callbacks?.onToolResult?.("tool-nav", result.result);
          }
          return {
            kind: "awaiting_user",
            ask: {
              toolCallId: "tool-ask",
              prompt: "Continue?",
              kind: "input",
            },
          };
        }
        return {
          kind: "completed",
          text: "Done",
          success: true,
        };
      }
    );

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    await wrapper.find("textarea").setValue("navigate first");
    await findButtonByText(wrapper, "Send")!.trigger("click");
    await flushPromises();

    expect(store.getChat(chatId)?.page).toEqual({
      path: "/projects/navigated",
      title: "Navigated Page",
    });
    expect(store.getPendingAsk(chatId)).toEqual({
      toolCallId: "tool-ask",
      prompt: "Continue?",
      kind: "input",
    });

    mockRoute.fullPath = "/projects/elsewhere";
    document.title = "Elsewhere Page";

    await wrapper.find("textarea").setValue("continue");
    await findButtonByText(wrapper, "Reply")!.trigger("click");
    await flushPromises();

    expect(mockBuildSystemPrompt).toHaveBeenNthCalledWith(1, {
      path: "/projects/demo",
      title: "Demo Page",
    });
    expect(mockBuildSystemPrompt).toHaveBeenNthCalledWith(2, {
      path: "/projects/elsewhere",
      title: "Elsewhere Page",
    });
  });

  test("shows retry CTA and reruns interrupted turns from the last stable input", async () => {
    const store = useAgentStore();
    const chatId = store.currentChatId!;

    store.updateChatPage(chatId, {
      path: "/projects/original",
      title: "Original Page",
    });
    store.addMessage({
      chatId,
      role: "user",
      content: "inspect this page",
      metadata: {
        route: "/projects/original",
      },
    });
    store.addMessage({
      chatId,
      role: "assistant",
      content: "Partial answer",
      metadata: {
        route: "/projects/original",
        runId: "run-1",
      },
    });
    store.addMessage({
      chatId,
      role: "tool",
      toolCallId: "tool-1",
      content: JSON.stringify({ partial: true }),
      metadata: {
        route: "/projects/original",
        runId: "run-1",
      },
    });
    store.interruptChatRun(chatId);
    store.getChat(chatId)!.runId = "run-1";

    mockRoute.fullPath = "/projects/current";
    document.title = "Current Page";
    mockRunAgentLoop.mockResolvedValue({
      kind: "completed",
      text: "Done",
      success: true,
    });

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    expect(wrapper.text()).toContain("Retry reruns the interrupted turn");
    await findButtonByText(wrapper, "Retry last turn")!.trigger("click");
    await flushPromises();

    expect(mockBuildSystemPrompt).toHaveBeenCalledWith({
      path: "/projects/current",
      title: "Current Page",
    });

    const [messages] = mockRunAgentLoop.mock.calls[0];
    expect(messages).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ role: "system", content: "system-prompt" }),
        expect.objectContaining({
          role: "user",
          content: "inspect this page",
        }),
      ])
    );
    expect(messages).not.toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: "assistant",
          content: "Partial answer",
        }),
        expect.objectContaining({
          role: "tool",
          toolCallId: "tool-1",
        }),
      ])
    );
    expect(
      store
        .getMessages(chatId)
        .some((message) => message.content === "Partial answer")
    ).toBe(false);
    expect(
      store
        .getMessages(chatId)
        .some((message) => message.toolCallId === "tool-1")
    ).toBe(false);
    expect(store.getChat(chatId)?.interrupted).toBe(false);
    expect(store.getChat(chatId)?.page).toEqual({
      path: "/projects/current",
      title: "Current Page",
    });
  });

  test("preserves the latest page snapshot when an interrupted run ends on a new page", async () => {
    const store = useAgentStore();
    const chatId = store.currentChatId!;

    store.updateChatPage(chatId, {
      path: "/projects/original",
      title: "Original Page",
    });

    mockRunAgentLoop.mockImplementation(async () => {
      mockRoute.fullPath = "/projects/navigated";
      document.title = "Navigated Page";
      return { kind: "aborted" };
    });

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    await wrapper.find("textarea").setValue("inspect this page");
    await findButtonByText(wrapper, "Send")!.trigger("click");
    await flushPromises();

    expect(store.getChat(chatId)?.interrupted).toBe(true);
    expect(store.getChat(chatId)?.page).toEqual({
      path: "/projects/navigated",
      title: "Navigated Page",
    });
  });

  test("dismissing interrupted state removes partial run output", async () => {
    const store = useAgentStore();
    const chatId = store.currentChatId!;

    store.addMessage({
      chatId,
      role: "user",
      content: "inspect this page",
    });
    store.addMessage({
      chatId,
      role: "assistant",
      content: "Partial answer",
      metadata: {
        runId: "run-1",
      },
    });
    store.addMessage({
      chatId,
      role: "tool",
      toolCallId: "tool-1",
      content: JSON.stringify({ partial: true }),
      metadata: {
        runId: "run-1",
      },
    });
    store.interruptChatRun(chatId);
    store.getChat(chatId)!.runId = "run-1";

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    await findButtonByText(wrapper, "Dismiss")!.trigger("click");
    await flushPromises();

    expect(store.getChat(chatId)?.interrupted).toBe(false);
    expect(store.getChat(chatId)?.runId).toBeNull();
    expect(
      store.getMessages(chatId).map((message) => ({
        role: message.role,
        content: message.content,
        toolCallId: message.toolCallId,
      }))
    ).toEqual([
      {
        role: "user",
        content: "inspect this page",
        toolCallId: undefined,
      },
    ]);
  });

  test("keeps the latest run cancellable when an earlier aborted run settles late", async () => {
    const store = useAgentStore();
    const chatId = store.currentChatId!;
    const firstRun = createDeferred<AgentLoopOutcome>();
    const secondRun = createDeferred<AgentLoopOutcome>();
    let firstSignal: AbortSignal | undefined;
    let secondSignal: AbortSignal | undefined;

    mockRunAgentLoop
      .mockImplementationOnce(
        async (
          _messages: unknown,
          _tools: unknown,
          _executor: unknown,
          _callbacks: unknown,
          signal?: AbortSignal
        ) => {
          firstSignal = signal;
          return firstRun.promise;
        }
      )
      .mockImplementationOnce(
        async (
          _messages: unknown,
          _tools: unknown,
          _executor: unknown,
          _callbacks: unknown,
          signal?: AbortSignal
        ) => {
          secondSignal = signal;
          return secondRun.promise;
        }
      );

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    await wrapper.find("textarea").setValue("first request");
    await findButtonByText(wrapper, "Send")!.trigger("click");
    await flushPromises();

    expect(store.getAbortController(chatId)?.signal).toBe(firstSignal);
    expect(store.loading).toBe(true);

    await findButtonByText(wrapper, "Stop")!.trigger("click");
    await flushPromises();

    expect(firstSignal?.aborted).toBe(true);
    expect(store.getAbortController(chatId)).toBeNull();
    expect(store.loading).toBe(false);

    await wrapper.find("textarea").setValue("second request");
    await findButtonByText(wrapper, "Send")!.trigger("click");
    await flushPromises();

    expect(store.getAbortController(chatId)?.signal).toBe(secondSignal);
    expect(store.loading).toBe(true);

    firstRun.resolve({ kind: "aborted" });
    await flushPromises();

    expect(store.getAbortController(chatId)?.signal).toBe(secondSignal);
    expect(store.loading).toBe(true);
    expect(
      store
        .getMessages(chatId)
        .some((message) => message.content === "_Interrupted_")
    ).toBe(false);

    secondRun.resolve({
      kind: "completed",
      text: "Done",
      success: true,
    });
    await flushPromises();

    expect(store.getAbortController(chatId)).toBeNull();
    expect(store.loading).toBe(false);
  });

  test("clears stale composer input when pending asks disappear", async () => {
    const store = useAgentStore();
    const chatId = store.currentChatId!;

    store.awaitUser(chatId, {
      toolCallId: "tool-ask",
      prompt: "Which project should I use?",
      kind: "input",
      defaultValue: "demo-project",
    });

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    expect(getTextareaValue(wrapper)).toBe("demo-project");
    await wrapper.find("textarea").setValue("stale input");

    store.clearPendingAsk(chatId);
    await flushPromises();

    expect(getTextareaValue(wrapper)).toBe("");
  });

  test("clears stale composer input when switching chats", async () => {
    const store = useAgentStore();
    const firstChatId = store.currentChatId!;

    store.awaitUser(firstChatId, {
      toolCallId: "tool-ask",
      prompt: "Which project should I use?",
      kind: "input",
      defaultValue: "demo-project",
    });

    const secondChat = store.createChat({ title: "Second" });
    store.setCurrentChat(firstChatId);

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    expect(getTextareaValue(wrapper)).toBe("demo-project");
    await wrapper.find("textarea").setValue("stale input");

    store.setCurrentChat(secondChat.id);
    await flushPromises();
    expect(getTextareaValue(wrapper)).toBe("");

    store.setCurrentChat(firstChatId);
    await flushPromises();
    expect(getTextareaValue(wrapper)).toBe("demo-project");
  });

  test("filters DOM ref mention options and keeps bracket insertion semantics", async () => {
    mockLazyExtractDomRefSuggestions.mockResolvedValue([
      createDomRefSuggestion({
        ref: "e1",
        tag: "BUTTON",
        role: "button",
        label: "Save changes",
      }),
      createDomRefSuggestion({
        ref: "e2",
        tag: "A",
        role: "link",
        label: "Cancel",
      }),
    ]);

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    const textarea = wrapper.find("textarea");
    const mention = wrapper.findComponent(NMention);
    await textarea.setValue("Click @sav");
    (textarea.element as HTMLTextAreaElement).setSelectionRange(10, 10);
    await textarea.trigger("keyup");
    await flushPromises();

    expect(mockLazyExtractDomRefSuggestions).toHaveBeenCalled();
    const options = mention.props("options") as {
      value: string;
      label: string;
      suggestion: DomRefSuggestion;
    }[];
    expect(options).toHaveLength(1);
    expect(options[0]?.value).toBe("[e1]");
    expect(options[0]?.suggestion.label).toBe("Save changes");

    mention.vm.$emit("update:value", "Click [e1] ");
    await flushPromises();

    expect(getTextareaValue(wrapper)).toBe("Click [e1] ");
  });

  test("keeps DOM ref mention options ordered for keyboard navigation", async () => {
    mockLazyExtractDomRefSuggestions.mockResolvedValue([
      createDomRefSuggestion({
        ref: "e1",
        tag: "BUTTON",
        role: "button",
        label: "Save changes",
      }),
      createDomRefSuggestion({
        ref: "e2",
        tag: "BUTTON",
        role: "button",
        label: "Delete changes",
      }),
    ]);

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    const textarea = wrapper.find("textarea");
    const mention = wrapper.findComponent(NMention);
    await textarea.setValue("Use @");
    (textarea.element as HTMLTextAreaElement).setSelectionRange(5, 5);
    await textarea.trigger("keyup");
    await flushPromises();

    const options = mention.props("options") as { value: string }[];
    expect(options.map((option) => option.value)).toEqual(["[e1]", "[e2]"]);
  });

  test("keeps all DOM ref mention options accessible and scrollable", async () => {
    mockLazyExtractDomRefSuggestions.mockResolvedValue(
      Array.from({ length: 10 }, (_, index) =>
        createDomRefSuggestion({
          ref: `e${index + 1}`,
          tag: "BUTTON",
          role: "button",
          label: `Option ${index + 1}`,
        })
      )
    );

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    const textarea = wrapper.find("textarea");
    const mention = wrapper.findComponent(NMention);
    await textarea.setValue("Use @");
    (textarea.element as HTMLTextAreaElement).setSelectionRange(5, 5);
    await textarea.trigger("keyup");
    await flushPromises();

    const options = mention.props("options") as { value: string }[];
    expect(options).toHaveLength(10);
    expect(options[0]?.value).toBe("[e1]");
    expect(options[9]?.value).toBe("[e10]");
    expect(mention.props("scrollbarProps")).toEqual({
      containerStyle: { maxHeight: "320px" },
    });
  });

  test("keeps the active DOM ref option scrolled into view during keyboard navigation", async () => {
    const scrollIntoView = vi.fn();
    const pendingOption = document.createElement("div");
    pendingOption.className = "n-base-select-option--pending";
    pendingOption.scrollIntoView = scrollIntoView;
    const menu = document.createElement("div");
    menu.className = "n-base-select-menu";
    menu.appendChild(pendingOption);
    document.body.appendChild(menu);

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    const textarea = wrapper.find("textarea");
    const mention = wrapper.findComponent(NMention);
    mention.vm.$emit("update:show", true);
    await textarea.trigger("keydown", { key: "ArrowDown" });
    await flushPromises();

    expect(scrollIntoView).toHaveBeenCalledWith({ block: "nearest" });

    menu.remove();
  });

  test("does not send when Enter selects a DOM ref mention", async () => {
    mockRunAgentLoop.mockResolvedValue({
      kind: "completed",
      text: "Done",
      success: true,
    });
    mockLazyExtractDomRefSuggestions.mockResolvedValue([
      createDomRefSuggestion({
        ref: "e1",
        tag: "BUTTON",
        role: "button",
        label: "Save changes",
      }),
    ]);

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    const textarea = wrapper.find("textarea");
    const mention = wrapper.findComponent(NMention);
    await textarea.setValue("Inspect @save");
    (textarea.element as HTMLTextAreaElement).setSelectionRange(13, 13);
    await textarea.trigger("keyup");
    await flushPromises();

    mention.vm.$emit("update:show", true);
    await textarea.trigger("keydown", { key: "Enter" });
    mention.vm.$emit("update:value", "Inspect [e1] ");
    mention.vm.$emit("update:show", false);
    await flushPromises();
    expect(mockRunAgentLoop).not.toHaveBeenCalled();
    expect(getTextareaValue(wrapper)).toBe("Inspect [e1] ");

    await textarea.trigger("keydown", { key: "Enter" });
    await flushPromises();

    expect(mockRunAgentLoop).toHaveBeenCalledTimes(1);
  });

  test("uses choose buttons to answer pending choose prompts", async () => {
    const store = useAgentStore();
    const chatId = store.currentChatId!;

    store.addMessage({
      chatId,
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
    store.awaitUser(chatId, {
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
      .getMessages(chatId)
      .filter((message) => message.role === "tool");
    expect(toolMessages).toHaveLength(1);
    expect(JSON.parse(toolMessages[0].content ?? "{}")).toEqual({
      kind: "choose",
      answer: "Production",
      value: "prod",
    });
    expect(store.getPendingAsk(chatId)).toBeNull();
    expect(mockRunAgentLoop).toHaveBeenCalledTimes(1);
  });

  test("uses confirm buttons to answer pending confirmation prompts", async () => {
    const store = useAgentStore();
    const chatId = store.currentChatId!;

    store.addMessage({
      chatId,
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
    store.awaitUser(chatId, {
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
      .getMessages(chatId)
      .filter((message) => message.role === "tool");
    expect(toolMessages).toHaveLength(1);
    expect(JSON.parse(toolMessages[0].content ?? "{}")).toEqual({
      kind: "confirm",
      answer: "Delete",
      confirmed: true,
    });
    expect(store.getPendingAsk(chatId)).toBeNull();
    expect(mockRunAgentLoop).toHaveBeenCalledTimes(1);
  });

  test("keeps the running chat selected while another chat is running", async () => {
    const store = useAgentStore();
    const firstChatId = store.currentChatId!;
    const secondChat = store.createChat({
      title: "Second thread",
      select: false,
    });
    const firstRun = createDeferred<AgentLoopOutcome>();

    mockRunAgentLoop.mockImplementationOnce(async () => firstRun.promise);

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    await wrapper.find("textarea").setValue("first request");
    await findButtonByText(wrapper, "Send")!.trigger("click");
    await flushPromises();

    expect(store.isChatRunning(firstChatId)).toBe(true);

    store.setCurrentChat(secondChat.id);
    await flushPromises();

    const textarea = wrapper.find("textarea");
    expect(store.currentChatId).toBe(firstChatId);
    expect((textarea.element as HTMLTextAreaElement).disabled).toBe(true);
    expect(mockRunAgentLoop).toHaveBeenCalledTimes(1);
    expect(store.getMessages(secondChat.id)).toEqual([]);

    firstRun.resolve({ kind: "completed", text: "First done", success: true });
    await flushPromises();
  });
});
