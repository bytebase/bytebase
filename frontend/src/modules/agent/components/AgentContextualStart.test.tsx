import { fireEvent, render, screen } from "@testing-library/react";
import { createInstance } from "i18next";
import type { ComponentProps } from "react";
import { useTranslation } from "react-i18next";
import { beforeEach, describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import type { AgentPageResource } from "../logic/chatContext";
import type { AgentResourceReference } from "../logic/types";
import { useAgentStore } from "../store/agent";
import { DEFAULT_AGENT_PREFERENCES } from "../store/preferences";
import { AgentContextualStart } from "./AgentContextualStart";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
}));

beforeEach(() => {
  useAgentStore.setState({ preferences: { ...DEFAULT_AGENT_PREFERENCES } });
});

const composerContext = {
  context: "projects/sample",
  contextTitle: "Sample project",
  workspace: { name: "workspaces/default", title: "Default workspace" },
};

const renderContext = (
  page: AgentPageResource,
  reference?: AgentResourceReference
) => {
  const onStart = vi.fn();
  render(
    <AgentContextualStart
      page={page}
      title="Masking settings"
      reference={reference}
      loading={false}
      canStart
      hasCurrentChat={false}
      draft={{ text: "" }}
      onDraftChange={vi.fn()}
      {...composerContext}
      onStart={onStart}
      onContinue={vi.fn()}
    />
  );
  return onStart;
};

test("shows Continue last chat as the localized primary action", async () => {
  const i18n = createInstance();
  await i18n.init({
    lng: "en-US",
    resources: { "en-US": { translation: enUS } },
  });
  vi.mocked(useTranslation).mockReturnValueOnce({
    t: i18n.t,
  } as ReturnType<typeof useTranslation>);
  const onContinue = vi.fn();
  const onStart = vi.fn();
  render(
    <AgentContextualStart
      page={{
        kind: "project",
        name: "projects/sample",
        project: "projects/sample",
      }}
      title="Sample project"
      loading={false}
      canStart
      hasCurrentChat
      draft={{ text: "" }}
      onDraftChange={vi.fn()}
      {...composerContext}
      onStart={onStart}
      onContinue={onContinue}
    />
  );

  const button = screen.getByRole("button", {
    name: "Continue last chat",
  });
  expect(button).toHaveClass("bg-accent", "text-accent-text");
  fireEvent.click(button);
  expect(onContinue).toHaveBeenCalledOnce();
  expect(onStart).not.toHaveBeenCalled();
});

test("hides Continue when there is no eligible chat", () => {
  renderContext({ kind: "global-masking" });
  expect(
    screen.queryByRole("button", {
      name: "agent.contextual-start.continue-chat",
    })
  ).toBeNull();
});

describe("AgentContextualStart masking contexts", () => {
  test("shows three rule-editor tasks alongside the normal chatbox context", () => {
    const onStart = vi.fn();
    render(
      <AgentContextualStart
        page={{ kind: "global-masking" }}
        title="Global Masking"
        maskingRuleEditor
        loading={false}
        canStart
        hasCurrentChat={false}
        draft={{ text: "" }}
        onDraftChange={vi.fn()}
        {...composerContext}
        onStart={onStart}
        onContinue={vi.fn()}
      />
    );
    const prompts = [
      "agent.contextual-start.masking-rule-configure",
      "agent.contextual-start.masking-rule-effect",
      "agent.contextual-start.masking-rule-review",
    ];
    expect(screen.getByText("Global Masking")).toBeTruthy();
    expect(screen.queryByText("agent.contextual-start.masking-global-effect")).toBeNull();
    expect(screen.getAllByRole("button", { name: /agent.contextual-start.masking-rule-/ })).toHaveLength(3);
    for (const prompt of prompts) {
      fireEvent.click(screen.getByRole("button", { name: prompt }));
      expect(onStart).toHaveBeenLastCalledWith(prompt, undefined);
    }
    expect(screen.getByText("Sample project")).toBeVisible();
  });

  test.each([
    [
      { kind: "global-masking" },
      "agent.contextual-start.masking-global-heading",
      "agent.contextual-start.masking-global-effect",
    ],
    [
      { kind: "semantic-types" },
      "agent.contextual-start.masking-semantic-heading",
      "agent.contextual-start.masking-semantic-algorithm",
    ],
    [
      { kind: "data-classification" },
      "agent.contextual-start.masking-classification-heading",
      "agent.contextual-start.masking-classification-effect",
    ],
    [
      { kind: "database-catalog", project: "projects/sample", names: ["projects/sample/instances/primary/databases/app"] },
      "agent.contextual-start.masking-catalog-heading",
      "agent.contextual-start.masking-catalog-visibility",
    ],
    [
      { kind: "masking-exemption", project: "projects/sample" },
      "agent.contextual-start.masking-exemption-heading",
      "agent.contextual-start.masking-exemption-effect",
    ],
  ] as const)("shows tailored help for %j", (page, heading, suggestion) => {
    renderContext(
      page as AgentPageResource,
      page.kind === "database-catalog"
        ? {
            kind: "database",
            name: page.names[0],
            project: page.project,
            label: "App database",
          }
        : undefined
    );
    expect(screen.getByText(heading)).toBeTruthy();
    expect(screen.getByText(suggestion)).toBeTruthy();
  });

  test("sends the selected masking question with the existing page context", () => {
    const onStart = renderContext({ kind: "global-masking" } as AgentPageResource);
    fireEvent.click(
      screen.getByRole("button", {
        name: "agent.contextual-start.masking-global-effect",
      })
    );
    expect(onStart).toHaveBeenCalledWith(
      "agent.contextual-start.masking-global-effect",
      undefined
    );
  });
});

const renderPrompt = (props: Partial<ComponentProps<typeof AgentContextualStart>> = {}) => {
  const onStart = vi.fn();
  const onDraftChange = vi.fn();
  const view = render(
    <AgentContextualStart
      page={{ kind: "project", name: "projects/sample", project: "projects/sample" }}
      title="Sample project"
      loading={false}
      canStart
      hasCurrentChat={false}
      draft={{ text: "  A custom question  " }}
      onDraftChange={onDraftChange}
      {...composerContext}
      onStart={onStart}
      onContinue={vi.fn()}
      {...props}
    />
  );
  return { ...view, onStart, onDraftChange };
};

describe("contextual prompt composer", () => {
  test("uses the normal chatbox context, zero-token footer, and aligned placeholder", () => {
    const { container } = renderPrompt({ draft: { text: "" } });
    const input = screen.getByRole("textbox", { name: "agent.input-placeholder" });
    expect(input).toHaveAttribute("rows", "1");
    expect(input).toHaveClass("h-9", "min-h-9");
    expect(container.querySelector("[data-agent-input-placeholder]")).toHaveTextContent(
      "agent.input-placeholder · agent.input-keyboard-hint"
    );
    const footer = container.querySelector("[data-agent-input-footer]");
    expect(footer).toHaveTextContent("Sample project");
    expect(footer).toHaveTextContent("agent.chat-total-tokens");
    const contextTrigger = container.querySelector("[data-agent-context-trigger]")!;
    expect(contextTrigger).toBeDisabled();
    fireEvent.click(contextTrigger);
    fireEvent.keyDown(contextTrigger, { key: "Enter" });
    expect(document.querySelector("[data-agent-context-options]")).toBeNull();
    expect(input).toBeEnabled();
  });
  test("edits locally and only sends a trimmed question after explicit submission", () => {
    const { onStart, onDraftChange } = renderPrompt();
    const input = screen.getByRole("textbox", { name: "agent.input-placeholder" });
    expect(input.closest("[data-agent-composer]")).not.toBeNull();
    expect(onStart).not.toHaveBeenCalled();
    fireEvent.change(input, { target: { value: "My own question\nwith details" } });
    expect(onDraftChange).toHaveBeenCalledWith({ text: "My own question\nwith details", mentions: [] });
    expect(onStart).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));
    expect(onStart).toHaveBeenCalledExactlyOnceWith("A custom question", undefined, "default");
  });

  test("keeps the current resource bound to a custom question", () => {
    const reference: AgentResourceReference = { kind: "database", name: "instances/primary/databases/app", project: "projects/sample", label: "App" };
    const { onStart } = renderPrompt({ page: { kind: "database", project: "projects/sample", names: [reference.name] }, reference });
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));
    expect(onStart).toHaveBeenCalledExactlyOnceWith("A custom question", reference, "default", [reference]);
  });

  test.each([
    { draft: { text: " \n " } },
    { canStart: false },
    { showAIConfigurationRecovery: true },
    { page: { kind: "database", project: "projects/sample", names: ["instances/primary/databases/app"] }, loading: true },
    { page: { kind: "database", project: "projects/sample", names: ["instances/primary/databases/app"] } },
  ] satisfies Partial<ComponentProps<typeof AgentContextualStart>>[])("does not submit an unavailable or empty composer %j", (props) => {
    const { onStart } = renderPrompt(props);
    expect(screen.getByRole("button", { name: "agent.send" })).toBeDisabled();
    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });
    expect(onStart).not.toHaveBeenCalled();
  });

  test.each([
    { sendWithEnter: true, key: "Enter", sends: true },
    { sendWithEnter: true, key: "Enter", shiftKey: true, sends: false },
    { sendWithEnter: true, key: "Enter", ctrlKey: true, sends: false },
    { sendWithEnter: true, key: "Enter", isComposing: true, sends: false },
    { sendWithEnter: true, key: "Enter", keyCode: 229, sends: false },
    { sendWithEnter: false, key: "Enter", sends: false },
    { sendWithEnter: false, key: "Enter", ctrlKey: true, sends: true },
    { sendWithEnter: false, key: "Enter", metaKey: true, sends: true },
    { sendWithEnter: false, key: "Enter", ctrlKey: true, shiftKey: true, sends: false },
    { sendWithEnter: false, key: "Enter", ctrlKey: true, altKey: true, sends: false },
  ])("uses the configured shortcut without sending IME or newline input %j", ({ sendWithEnter, sends, ...event }) => {
    useAgentStore.setState({ preferences: { ...DEFAULT_AGENT_PREFERENCES, sendWithEnter } });
    const { onStart } = renderPrompt();
    const input = screen.getByRole("textbox");
    const allowsDefault = fireEvent.keyDown(input, event);
    expect(allowsDefault).toBe(!sends);
    expect(onStart).toHaveBeenCalledTimes(sends ? 1 : 0);
  });
});
