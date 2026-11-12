import { fireEvent, screen } from "@testing-library/react";
import { act, createRef, type ComponentProps, type ReactElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import type { AgentChat } from "../logic/types";
import { AgentChatbox, AgentChatboxFooter } from "./AgentChatbox";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, values?: Record<string, unknown>) => {
      const value = key.split(".").reduce<unknown>((result, part) =>
        typeof result === "object" && result !== null ? (result as Record<string, unknown>)[part] : undefined, enUS);
      return typeof value === "string" ? value.replace(/{{(\w+)}}/g, (_, name) => String(values?.[name] ?? "")) : key;
    },
  }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useProjectList: () => ({
    projects: [{ name: "projects/orders", title: "Orders" }],
    isLoading: false,
    isFetchingMore: false,
    hasMore: false,
    loadMore: vi.fn(),
  }),
}));

let root: Root | undefined;
afterEach(() => {
  act(() => root?.unmount());
  root = undefined;
  document.body.innerHTML = "";
  vi.unstubAllGlobals();
});

function render(value: ReactElement) {
  const container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  act(() => root!.render(value));
  return container;
}

function renderChatbox(overrides: Partial<ComponentProps<typeof AgentChatbox>> = {}) {
  const textareaRef = createRef<HTMLTextAreaElement>();
  const onSend = vi.fn();
  const props: ComponentProps<typeof AgentChatbox> = {
    textareaProps: {
      textareaRef,
      value: "",
      "aria-label": "Question",
      placeholder: "Native duplicate",
      onChange: vi.fn(),
    },
    placeholder: "Ask a question · Enter sends",
    footer: <span>Current context</span>,
    onSend,
    sendDisabled: false,
    ...overrides,
  };
  return { container: render(<AgentChatbox {...props} />), textareaRef, onSend };
}

describe("AgentChatbox", () => {
  test("reserves padded sibling space and intrinsic footer height when a long draft shrinks its container", () => {
    let resize: ResizeObserverCallback | undefined;
    vi.stubGlobal("ResizeObserver", class {
      constructor(callback: ResizeObserverCallback) { resize = callback; }
      observe() {}
      disconnect() {}
    });
    const containerRef = createRef<HTMLDivElement>();
    const textareaRef = createRef<HTMLTextAreaElement>();
    render(<div>
      <div style={{ padding: "12px" }}>Conversation</div>
      <div ref={containerRef}>
        <AgentChatbox
          containerRef={containerRef}
          textareaProps={{ textareaRef, value: "Long draft", "aria-label": "Question" }}
          placeholder="Ask a question"
          footer={<span>Context</span>}
          onSend={vi.fn()}
          sendDisabled={false}
        />
      </div>
    </div>);
    const input = textareaRef.current!;
    const container = containerRef.current!;
    Object.defineProperties(input, { scrollHeight: { value: 600 }, offsetHeight: { value: 300 }, clientHeight: { value: 300 } });
    Object.defineProperties(container, { scrollHeight: { value: 400 }, offsetHeight: { value: 350 }, clientHeight: { value: 348 } });
    Object.defineProperty(container.parentElement, "clientHeight", { value: 360 });
    act(() => resize!([], {} as ResizeObserver));
    expect(input.style.height).toBe("234px");
    expect(input.style.overflowY).toBe("auto");
  });

  test("uses the normal input placeholder overlay without a competing native placeholder", () => {
    const { container, textareaRef } = renderChatbox();
    const input = screen.getByRole("textbox", { name: "Question" });
    expect(textareaRef.current).toBe(input);
    expect(input.getAttribute("rows")).toBe("1");
    expect(input.getAttribute("placeholder")).toBeNull();
    expect(container.querySelector("[data-agent-input-placeholder]")?.textContent).toBe("Ask a question · Enter sends");
    expect(container.querySelector("[data-agent-composer-footer]")?.textContent).toContain("Current context");
  });

  test("does not overlay the placeholder on a typed question", () => {
    const { container } = renderChatbox({
      textareaProps: { textareaRef: createRef(), value: "A question", "aria-label": "Question" },
    });
    expect(container.querySelector("[data-agent-input-placeholder]")).toBeNull();
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("A question");
  });

  test("uses the processing placeholder and stop action during a run", () => {
    const onStop = vi.fn();
    const { container, onSend } = renderChatbox({ processing: true, onStop });
    expect(container.querySelector("[data-agent-input-placeholder]")?.textContent).toBe("Processing...");
    expect(container.querySelector("[data-agent-send]")).toBeNull();
    act(() => fireEvent.click(screen.getByRole("button", { name: "Stop" })));
    expect(onStop).toHaveBeenCalledTimes(1);
    expect(onSend).not.toHaveBeenCalled();
  });

  test("does not stop a remotely controlled run", () => {
    const onStop = vi.fn();
    renderChatbox({ processing: true, onStop, stopDisabled: true });
    const button = screen.getByRole("button", { name: "Stop" }) as HTMLButtonElement;
    expect(button.disabled).toBe(true);
    act(() => fireEvent.click(button));
    expect(onStop).not.toHaveBeenCalled();
  });

  test.each([true, false])("honors the send disabled state %s", (sendDisabled) => {
    const { onSend } = renderChatbox({ sendDisabled, sendLabel: "Reply" });
    const button = screen.getByRole("button", { name: "Reply" }) as HTMLButtonElement;
    expect(button.disabled).toBe(sendDisabled);
    act(() => fireEvent.click(button));
    expect(onSend).toHaveBeenCalledTimes(sendDisabled ? 0 : 1);
  });

  test("retains native editing, shortcut, and atomic resource-mention deletion handlers", () => {
    const onChange = vi.fn();
    const onKeyDown = vi.fn();
    const onDeleteMention = vi.fn();
    const { container } = renderChatbox({
      textareaProps: {
        textareaRef: createRef(),
        value: "@Orders question",
        mentions: [{ range: { start: 0, end: 7 }, label: "Orders" }],
        "aria-label": "Question",
        onChange,
        onKeyDown,
        onDeleteMention,
      },
      children: <div role="status">Resource picker</div>,
    });
    const input = screen.getByRole("textbox") as HTMLTextAreaElement;
    expect(container.querySelector("[data-agent-inline-mention]")?.textContent).toBe("@Orders");
    expect(screen.getByRole("status").textContent).toBe("Resource picker");
    act(() => fireEvent.change(input, { target: { value: "@Orders new question" } }));
    expect(onChange).toHaveBeenCalledTimes(1);
    act(() => fireEvent.keyDown(input, { key: "Enter" }));
    expect(onKeyDown).toHaveBeenCalledTimes(1);
    input.setSelectionRange(7, 7);
    act(() => fireEvent.keyDown(input, { key: "Backspace" }));
    expect(onDeleteMention).toHaveBeenCalledWith({ text: "question", caret: 0 });
  });
});

describe("AgentChatboxFooter", () => {
  const chat: AgentChat = {
    id: "chat-1", title: "Question", createdTs: 0, updatedTs: 0,
    status: "idle", archived: false, totalTokensUsed: 150,
  };

  test("renders the context selector and the supplied chat token usage together", async () => {
    const onSelectContext = vi.fn(() => true);
    const container = render(<AgentChatboxFooter
      context="workspaces/demo"
      contextTitle="Demo workspace"
      workspace={{ name: "workspaces/demo", title: "Demo workspace" }}
      contextDisabled={false}
      onSelectContext={onSelectContext}
      chat={chat}
    />);
    expect(container.querySelector("[data-agent-input-footer]")?.textContent).toContain("Chat tokens: 150");
    const trigger = container.querySelector<HTMLButtonElement>("[data-agent-context-trigger]")!;
    expect(trigger.textContent).toContain("Context: Demo workspace");
    await act(async () => trigger.click());
    const project = document.querySelector<HTMLButtonElement>('[data-agent-context-option="projects/orders"]')!;
    await act(async () => project.click());
    expect(onSelectContext).toHaveBeenCalledWith("projects/orders");
  });

  test("disables context switching and does not show usage from an absent chat", () => {
    const container = render(<AgentChatboxFooter
      contextTitle="Orders"
      contextDisabled
      onSelectContext={vi.fn(() => true)}
      chat={null}
    />);
    expect(container.querySelector<HTMLButtonElement>("[data-agent-context-trigger]")?.disabled).toBe(true);
    expect(container.textContent).not.toContain("Chat tokens:");
  });

  test("can display empty token usage for a new contextual chat", () => {
    const container = render(<AgentChatboxFooter
      contextTitle="Orders"
      contextDisabled={false}
      onSelectContext={vi.fn(() => true)}
      chat={null}
      showEmptyUsage
    />);
    expect(container.querySelector("[data-agent-input-footer]")?.textContent).toContain("Chat tokens: 0");
  });
});
