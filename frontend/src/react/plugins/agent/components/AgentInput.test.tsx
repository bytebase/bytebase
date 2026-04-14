import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { DomRefSuggestion } from "../dom";
import { createAgentStore, useAgentStore } from "../store/agent";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  routerPush: vi.fn(),
  lazyExtractDomRefSuggestions: vi.fn<() => Promise<DomRefSuggestion[]>>(
    async () => []
  ),
  runAgentLoop: vi.fn(),
  isAgentAIConfigurationError: vi.fn(() => false),
  buildOutboundHistory: vi.fn(() => []),
  buildSystemPrompt: vi.fn(() => "system"),
  createToolExecutor: vi.fn(() => ({})),
  getToolDefinitions: vi.fn(() => []),
}));

let AgentInput: typeof import("./AgentInput").AgentInput;

function createMockStorage(): Storage {
  let store: Record<string, string> = {};
  return {
    get length() {
      return Object.keys(store).length;
    },
    key(index: number) {
      return Object.keys(store)[index] ?? null;
    },
    getItem(key: string) {
      return store[key] ?? null;
    },
    setItem(key: string, value: string) {
      store[key] = String(value);
    },
    removeItem(key: string) {
      delete store[key];
    },
    clear() {
      store = {};
    },
  };
}

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/router", () => ({
  router: {
    currentRoute: {
      value: {
        fullPath: "/demo",
      },
    },
    push: mocks.routerPush,
  },
}));

vi.mock("../dom", () => ({
  lazyExtractDomRefSuggestions: mocks.lazyExtractDomRefSuggestions,
}));

vi.mock("../logic/agentLoop", () => ({
  runAgentLoop: mocks.runAgentLoop,
}));

vi.mock("../logic/aiConfiguration", () => ({
  isAgentAIConfigurationError: mocks.isAgentAIConfigurationError,
}));

vi.mock("../logic/outboundHistory", () => ({
  buildOutboundHistory: mocks.buildOutboundHistory,
}));

vi.mock("../logic/prompt", () => ({
  buildSystemPrompt: mocks.buildSystemPrompt,
}));

vi.mock("../logic/tools", () => ({
  createToolExecutor: mocks.createToolExecutor,
  getToolDefinitions: mocks.getToolDefinitions,
}));

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

const setTextareaValue = (textarea: HTMLTextAreaElement, value: string) => {
  const descriptor = Object.getOwnPropertyDescriptor(
    HTMLTextAreaElement.prototype,
    "value"
  );
  descriptor?.set?.call(textarea, value);
};

const stubAnimationFrames = () => {
  let nextFrameId = 0;
  const callbacks = new Map<number, FrameRequestCallback>();

  vi.stubGlobal(
    "requestAnimationFrame",
    vi.fn((callback: FrameRequestCallback) => {
      const frameId = ++nextFrameId;
      callbacks.set(frameId, callback);
      return frameId;
    })
  );
  vi.stubGlobal(
    "cancelAnimationFrame",
    vi.fn((frameId: number) => {
      callbacks.delete(frameId);
    })
  );

  return async () => {
    const currentCallbacks = [...callbacks.values()];
    callbacks.clear();

    await act(async () => {
      currentCallbacks.forEach((callback) => callback(performance.now()));
    });
  };
};

beforeEach(async () => {
  vi.stubGlobal("localStorage", createMockStorage());
  useAgentStore.setState(createAgentStore().getState(), true);

  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.routerPush.mockReset();
  mocks.lazyExtractDomRefSuggestions.mockReset();
  mocks.lazyExtractDomRefSuggestions.mockResolvedValue([]);
  mocks.runAgentLoop.mockReset();
  mocks.isAgentAIConfigurationError.mockReset();
  mocks.isAgentAIConfigurationError.mockReturnValue(false);
  mocks.buildOutboundHistory.mockReset();
  mocks.buildOutboundHistory.mockReturnValue([]);
  mocks.buildSystemPrompt.mockReset();
  mocks.buildSystemPrompt.mockReturnValue("system");
  mocks.createToolExecutor.mockReset();
  mocks.createToolExecutor.mockReturnValue({});
  mocks.getToolDefinitions.mockReset();
  mocks.getToolDefinitions.mockReturnValue([]);

  ({ AgentInput } = await import("./AgentInput"));
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("AgentInput", () => {
  test("mounts mention suggestions into the agent layer root", async () => {
    const suggestions: DomRefSuggestion[] = [
      {
        ref: "button.submit",
        tag: "BUTTON",
        role: "button",
        label: "Submit",
        value: "",
      },
    ];
    mocks.lazyExtractDomRefSuggestions.mockResolvedValue(suggestions);

    const { render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const textarea = document.body.querySelector(
      "textarea"
    ) as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "@");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    const agentRoot = document.getElementById("bb-react-layer-agent");
    expect(agentRoot?.querySelector("[data-agent-mention-list]")).toBeTruthy();

    unmount();
  });

  test("repositions mention suggestions when the textarea rect changes", async () => {
    const suggestions: DomRefSuggestion[] = [
      {
        ref: "button.submit",
        tag: "BUTTON",
        role: "button",
        label: "Submit",
        value: "",
      },
    ];
    mocks.lazyExtractDomRefSuggestions.mockResolvedValue(suggestions);
    const flushAnimationFrame = stubAnimationFrames();

    const { render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const textarea = document.body.querySelector(
      "textarea"
    ) as HTMLTextAreaElement;

    let rect = {
      bottom: 234,
      height: 34,
      left: 100,
      right: 400,
      top: 200,
      width: 300,
      x: 100,
      y: 200,
      toJSON: () => "",
    };
    vi.spyOn(textarea, "getBoundingClientRect").mockImplementation(
      () => rect as DOMRect
    );

    await act(async () => {
      setTextareaValue(textarea, "@");
      textarea.setSelectionRange(1, 1);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    const mentionList = document.body.querySelector(
      "[data-agent-mention-list]"
    ) as HTMLDivElement;
    expect(mentionList.style.left).toBe("100px");
    expect(mentionList.style.top).toBe("196px");
    expect(mentionList.style.width).toBe("300px");

    rect = {
      ...rect,
      bottom: 294,
      left: 160,
      right: 440,
      top: 260,
      width: 280,
      x: 160,
      y: 260,
    };

    await flushAnimationFrame();

    expect(mentionList.style.left).toBe("160px");
    expect(mentionList.style.top).toBe("256px");
    expect(mentionList.style.width).toBe("280px");

    unmount();
  });

  test("shows a single-line overlay placeholder and hides it when typing", () => {
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const placeholder = container.querySelector(
      "[data-agent-input-placeholder]"
    );
    expect(placeholder?.textContent).toBe("agent.input-placeholder");
    expect(placeholder?.className).toContain("truncate");
    expect(placeholder?.className).toContain("pointer-events-none");

    const textarea = container.querySelector("textarea");
    expect(textarea).toBeInstanceOf(HTMLTextAreaElement);

    act(() => {
      setTextareaValue(textarea as HTMLTextAreaElement, "hello");
      textarea?.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(
      container.querySelector("[data-agent-input-placeholder]")
    ).toBeNull();

    unmount();
  });

  test("uses the same single-line height for the textarea and send button", () => {
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const row = container.querySelector("[data-agent-input-row]");
    expect(row?.className).toContain("items-end");
    expect(container.querySelector("textarea")?.className).toContain(
      "min-h-[34px]"
    );
    expect(container.querySelector("textarea")?.className).toContain(
      "max-h-[134px]"
    );
    expect(container.querySelector("textarea")?.className).toContain("block");
    expect(container.querySelector("button")?.className).toContain("h-[34px]");

    unmount();
  });

  test("sizes from CSS min/max heights and enables scrolling only at max height", () => {
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const textarea = container.querySelector("textarea");
    expect(textarea).toBeInstanceOf(HTMLTextAreaElement);

    const element = textarea as HTMLTextAreaElement;
    let scrollHeight = 32;
    Object.defineProperty(element, "scrollHeight", {
      configurable: true,
      get: () => scrollHeight,
    });
    Object.defineProperty(element, "offsetHeight", {
      configurable: true,
      get: () => 34,
    });
    Object.defineProperty(element, "clientHeight", {
      configurable: true,
      get: () => 32,
    });
    const getComputedStyleSpy = vi
      .spyOn(window, "getComputedStyle")
      .mockReturnValue({
        minHeight: "34px",
        maxHeight: "134px",
      } as CSSStyleDeclaration);

    act(() => {
      setTextareaValue(element, "hello");
      element.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(element.style.height).toBe("34px");
    expect(element.style.overflowY).toBe("hidden");

    scrollHeight = 200;

    act(() => {
      setTextareaValue(element, "a\nb\nc\nd\ne\nf\ng");
      element.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(element.style.height).toBe("134px");
    expect(element.style.overflowY).toBe("auto");

    getComputedStyleSpy.mockRestore();
    unmount();
  });
});
