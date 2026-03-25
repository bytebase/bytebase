import { mount } from "@vue/test-utils";
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
        "retry-last-turn": "Retry last turn",
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

  test("resumes pending asks with the current page snapshot", async () => {
    const store = useAgentStore();
    const threadId = store.currentThreadId!;

    store.updateThreadPage(threadId, {
      path: "/projects/original",
      title: "Original Page",
    });
    store.awaitUser(threadId, {
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
    await wrapper.find("button").trigger("click");
    await flushPromises();

    expect(mockBuildSystemPrompt).toHaveBeenCalledWith({
      path: "/projects/other",
      title: "Other Page",
    });
    expect(store.getThread(threadId)?.page).toEqual({
      path: "/projects/other",
      title: "Other Page",
    });
  });

  test("increments the current thread token total from loop usage", async () => {
    const store = useAgentStore();
    const threadId = store.currentThreadId!;

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

    expect(store.getThread(threadId)?.totalTokensUsed).toBe(123);
  });

  test("refreshes the saved thread page snapshot after navigate before a follow-up turn", async () => {
    const store = useAgentStore();
    const threadId = store.currentThreadId!;
    let runCount = 0;

    store.updateThreadPage(threadId, {
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
    await wrapper.find("button").trigger("click");
    await flushPromises();

    expect(store.getThread(threadId)?.page).toEqual({
      path: "/projects/navigated",
      title: "Navigated Page",
    });
    expect(store.getPendingAsk(threadId)).toEqual({
      toolCallId: "tool-ask",
      prompt: "Continue?",
      kind: "input",
    });

    mockRoute.fullPath = "/projects/elsewhere";
    document.title = "Elsewhere Page";

    await wrapper.find("textarea").setValue("continue");
    await wrapper.find("button").trigger("click");
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
    const threadId = store.currentThreadId!;

    store.updateThreadPage(threadId, {
      path: "/projects/original",
      title: "Original Page",
    });
    store.addMessage({
      threadId,
      role: "user",
      content: "inspect this page",
      metadata: {
        route: "/projects/original",
      },
    });
    store.addMessage({
      threadId,
      role: "assistant",
      content: "Partial answer",
      metadata: {
        route: "/projects/original",
        runId: "run-1",
      },
    });
    store.addMessage({
      threadId,
      role: "tool",
      toolCallId: "tool-1",
      content: JSON.stringify({ partial: true }),
      metadata: {
        route: "/projects/original",
        runId: "run-1",
      },
    });
    store.interruptRun(threadId);
    store.getThread(threadId)!.runId = "run-1";

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
        .getMessages(threadId)
        .some((message) => message.content === "Partial answer")
    ).toBe(false);
    expect(
      store
        .getMessages(threadId)
        .some((message) => message.toolCallId === "tool-1")
    ).toBe(false);
    expect(store.getThread(threadId)?.interrupted).toBe(false);
    expect(store.getThread(threadId)?.page).toEqual({
      path: "/projects/current",
      title: "Current Page",
    });
  });

  test("preserves the latest page snapshot when an interrupted run ends on a new page", async () => {
    const store = useAgentStore();
    const threadId = store.currentThreadId!;

    store.updateThreadPage(threadId, {
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

    expect(store.getThread(threadId)?.interrupted).toBe(true);
    expect(store.getThread(threadId)?.page).toEqual({
      path: "/projects/navigated",
      title: "Navigated Page",
    });
  });

  test("dismissing interrupted state removes partial run output", async () => {
    const store = useAgentStore();
    const threadId = store.currentThreadId!;

    store.addMessage({
      threadId,
      role: "user",
      content: "inspect this page",
    });
    store.addMessage({
      threadId,
      role: "assistant",
      content: "Partial answer",
      metadata: {
        runId: "run-1",
      },
    });
    store.addMessage({
      threadId,
      role: "tool",
      toolCallId: "tool-1",
      content: JSON.stringify({ partial: true }),
      metadata: {
        runId: "run-1",
      },
    });
    store.interruptRun(threadId);
    store.getThread(threadId)!.runId = "run-1";

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    await findButtonByText(wrapper, "Dismiss")!.trigger("click");
    await flushPromises();

    expect(store.getThread(threadId)?.interrupted).toBe(false);
    expect(store.getThread(threadId)?.runId).toBeNull();
    expect(
      store.getMessages(threadId).map((message) => ({
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
    const threadId = store.currentThreadId!;
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

    expect(store.getAbortController(threadId)?.signal).toBe(firstSignal);
    expect(store.loading).toBe(true);

    await findButtonByText(wrapper, "Stop")!.trigger("click");
    await flushPromises();

    expect(firstSignal?.aborted).toBe(true);
    expect(store.getAbortController(threadId)).toBeNull();
    expect(store.loading).toBe(false);

    await wrapper.find("textarea").setValue("second request");
    await findButtonByText(wrapper, "Send")!.trigger("click");
    await flushPromises();

    expect(store.getAbortController(threadId)?.signal).toBe(secondSignal);
    expect(store.loading).toBe(true);

    firstRun.resolve({ kind: "aborted" });
    await flushPromises();

    expect(store.getAbortController(threadId)?.signal).toBe(secondSignal);
    expect(store.loading).toBe(true);
    expect(
      store
        .getMessages(threadId)
        .some((message) => message.content === "_Interrupted_")
    ).toBe(false);

    secondRun.resolve({
      kind: "completed",
      text: "Done",
      success: true,
    });
    await flushPromises();

    expect(store.getAbortController(threadId)).toBeNull();
    expect(store.loading).toBe(false);
  });

  test("clears stale composer input when pending asks disappear", async () => {
    const store = useAgentStore();
    const threadId = store.currentThreadId!;

    store.awaitUser(threadId, {
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

    store.clearPendingAsk(threadId);
    await flushPromises();

    expect(getTextareaValue(wrapper)).toBe("");
  });

  test("clears stale composer input when switching threads", async () => {
    const store = useAgentStore();
    const firstThreadId = store.currentThreadId!;

    store.awaitUser(firstThreadId, {
      toolCallId: "tool-ask",
      prompt: "Which project should I use?",
      kind: "input",
      defaultValue: "demo-project",
    });

    const secondThread = store.createThread({ title: "Second" });
    store.setCurrentThread(firstThreadId);

    const wrapper = mount(AgentInput, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    expect(getTextareaValue(wrapper)).toBe("demo-project");
    await wrapper.find("textarea").setValue("stale input");

    store.setCurrentThread(secondThread.id);
    await flushPromises();
    expect(getTextareaValue(wrapper)).toBe("");

    store.setCurrentThread(firstThreadId);
    await flushPromises();
    expect(getTextareaValue(wrapper)).toBe("demo-project");
  });

  test("shows filtered DOM ref suggestions and inserts the selected ref", async () => {
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
    await textarea.setValue("Click @sav");
    (textarea.element as HTMLTextAreaElement).setSelectionRange(10, 10);
    await textarea.trigger("select");
    await flushPromises();

    expect(mockLazyExtractDomRefSuggestions).toHaveBeenCalled();
    const items = wrapper.findAll('[data-testid="dom-ref-autocomplete-item"]');
    expect(items).toHaveLength(1);
    expect(items[0].text()).toContain("[e1]");
    expect(items[0].text()).toContain("Save changes");

    await textarea.trigger("keydown", { key: "Enter" });
    await flushPromises();

    expect(getTextareaValue(wrapper)).toBe("Click [e1] ");
    expect(wrapper.find('[data-testid="dom-ref-autocomplete"]').exists()).toBe(
      false
    );
  });

  test("uses arrow keys to change the active DOM ref suggestion before selecting", async () => {
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
    await textarea.setValue("Use @");
    (textarea.element as HTMLTextAreaElement).setSelectionRange(5, 5);
    await textarea.trigger("select");
    await flushPromises();

    await textarea.trigger("keydown", { key: "ArrowDown" });
    await textarea.trigger("keydown", { key: "Enter" });
    await flushPromises();

    expect(getTextareaValue(wrapper)).toBe("Use [e2] ");
  });

  test("does not send while the DOM ref menu is open and escape closes it", async () => {
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
    await textarea.setValue("Inspect @save");
    (textarea.element as HTMLTextAreaElement).setSelectionRange(13, 13);
    await textarea.trigger("select");
    await flushPromises();

    await textarea.trigger("keydown", { key: "Escape" });
    await flushPromises();
    expect(wrapper.find('[data-testid="dom-ref-autocomplete"]').exists()).toBe(
      false
    );

    await textarea.trigger("keydown", { key: "Enter" });
    await flushPromises();

    expect(mockRunAgentLoop).toHaveBeenCalledTimes(1);
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

  test("keeps the running thread selected while another thread is running", async () => {
    const store = useAgentStore();
    const firstThreadId = store.currentThreadId!;
    const secondThread = store.createThread({
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

    expect(store.isThreadRunning(firstThreadId)).toBe(true);

    store.setCurrentThread(secondThread.id);
    await flushPromises();

    const textarea = wrapper.find("textarea");
    expect(store.currentThreadId).toBe(firstThreadId);
    expect((textarea.element as HTMLTextAreaElement).disabled).toBe(true);
    expect(mockRunAgentLoop).toHaveBeenCalledTimes(1);
    expect(store.getMessages(secondThread.id)).toEqual([]);

    firstRun.resolve({ kind: "completed", text: "First done", success: true });
    await flushPromises();
  });
});
