import { fireEvent, render, screen } from "@testing-library/react";
import { createInstance } from "i18next";
import type { ComponentProps } from "react";
import { useTranslation } from "react-i18next";
import { beforeEach, describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import esES from "@/locales/es-ES.json";
import jaJP from "@/locales/ja-JP.json";
import viVN from "@/locales/vi-VN.json";
import zhCN from "@/locales/zh-CN.json";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { AgentPageResource } from "../logic/chatContext";
import type { AgentSqlContextCapture, AgentSqlEditorSnapshot } from "../logic/sqlEditorContext";
import type { AgentResourceReference } from "../logic/types";
import { useAgentStore } from "../store/agent";
import { DEFAULT_AGENT_PREFERENCES } from "../store/preferences";
import { AgentContextualStart } from "./AgentContextualStart";
import { useSqlEditorSuggestions } from "./useSqlEditorSuggestions";

vi.mock("./useSqlEditorSuggestions", () => ({ useSqlEditorSuggestions: vi.fn(() => ({ status: "idle" })) }));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
}));

beforeEach(() => {
  vi.mocked(useSqlEditorSuggestions).mockReturnValue({ status: "idle" });
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

describe("SQL Editor starters", () => {
  const project = { version: 1 as const, project: "projects/sample" };
  const connected: AgentSqlEditorSnapshot = {
    ...project,
    tab: { id: "query", mode: "SAVED_QUERY" },
    connection: { database: "instances/main/databases/app", project: project.project, engine: Engine.POSTGRES },
  };
  const selection: AgentSqlEditorSnapshot = {
    ...connected,
    editorTarget: {
      actor: "users/member@example.com", workspace: "workspaces/default", hostId: "sql", mountId: "mounted",
      project: project.project, tabId: "query", tabMode: "SAVED_QUERY", modelUri: "inmemory://sql/query", modelVersion: 1,
      connection: connected.connection, selection: { start: 1, end: 10, caret: 10 },
    },
  };
  const props = {
    page: { kind: "project" as const, name: project.project, project: project.project },
    title: "Sample project", loading: false, canStart: true, hasCurrentChat: true,
    draft: { text: "Keep my question" }, onDraftChange: vi.fn(), ...composerContext,
  };
  const base = ["write-query", "find-database", "fix-error"];
  const database = ["explore-database", "write-query", "table-relationships"];
  const selected = ["explain-selection", "review-selection", "improve-selection"];

  test.each([
    { locale: "en-US", messages: enUS, label: "Show databases in this project", errorLabel: "Help me fix a SQL error" },
    { locale: "es-ES", messages: esES, label: "Muestra las bases de datos de este proyecto", errorLabel: "Ayúdame a corregir un error de SQL" },
    { locale: "ja-JP", messages: jaJP, label: "このプロジェクトのデータベースを表示して", errorLabel: "SQL エラーの修正を手伝って" },
    { locale: "vi-VN", messages: viVN, label: "Hiển thị cơ sở dữ liệu trong dự án này", errorLabel: "Giúp tôi sửa lỗi SQL" },
    { locale: "zh-CN", messages: zhCN, label: "显示此项目中的数据库", errorLabel: "帮我修复 SQL 错误" },
  ])("offers localized SQL prompts without a connection in $locale", async ({ locale, messages, label, errorLabel }) => {
    const i18n = createInstance();
    await i18n.init({ lng: locale, resources: { [locale]: { translation: messages } } });
    vi.mocked(useTranslation).mockReturnValueOnce({ t: i18n.t, i18n } as ReturnType<typeof useTranslation>);
    const onStart = vi.fn();
    render(<AgentContextualStart {...props} sqlEditor={{ kind: "ready", snapshot: project }} onStart={onStart} onContinue={vi.fn()} />);
    const prompt = screen.getByRole("button", { name: label });
    expect(prompt).toHaveAttribute("data-agent-sql-starter", "find-database");
    expect(onStart).not.toHaveBeenCalled();
    fireEvent.click(prompt);
    expect(onStart).toHaveBeenCalledWith(label, undefined, "default", undefined, "find-database");
    const errorPrompt = screen.getByRole("button", { name: errorLabel });
    expect(document.querySelectorAll("[data-agent-contextual-suggestion]")[2]).toBe(errorPrompt);
    fireEvent.click(errorPrompt);
    expect(onStart).toHaveBeenLastCalledWith(errorLabel, undefined, "default", undefined, "fix-error");
  });

  test.each([
    { name: "no database", capture: { kind: "ready", snapshot: project }, bridge: false, ids: base },
    { name: "no Project or database", capture: { kind: "ready", snapshot: { version: 1 } }, bridge: false, ids: ["write-query", "fix-error"] },
    { name: "connected database", capture: { kind: "ready", snapshot: connected }, bridge: false, ids: database },
    { name: "selected SQL", capture: { kind: "ready", snapshot: selection }, bridge: true, ids: selected },
    { name: "no native reader", capture: { kind: "ready", snapshot: selection }, bridge: false, ids: database },
    { name: "empty selection", capture: { kind: "ready", snapshot: { ...selection, editorTarget: { ...selection.editorTarget!, selection: { start: 1, end: 1, caret: 1 } } } }, bridge: true, ids: database },
    { name: "pending context", capture: { kind: "pending" }, bridge: true, ids: base },
  ])("offers only applicable prompts for $name", ({ capture, bridge, ids }) => {
    const onStart = vi.fn();
    const onContinue = vi.fn();
    render(<AgentContextualStart {...props} sqlEditor={capture as AgentSqlContextCapture}
      canCaptureSqlSelection={bridge} contextPending={capture.kind === "pending"} onStart={onStart} onContinue={onContinue} />);
    expect(screen.getByRole("heading", { name: "agent.contextual-start.sql-heading" })).toBeVisible();
    expect(screen.queryByText("agent.landing-starters.project-next")).toBeNull();
    const rows = Array.from(document.querySelectorAll<HTMLButtonElement>("[data-agent-contextual-suggestion]"));
    expect(rows.map(row => row.dataset.agentSqlStarter)).toEqual(ids);
    for (const [index, row] of rows.entries()) {
      expect(row.disabled).toBe(capture.kind === "pending");
      fireEvent.click(row);
      if (capture.kind === "ready") expect(onStart).toHaveBeenLastCalledWith(
        `agent.contextual-start.sql-${ids[index]}`, undefined, "default", undefined, ids[index]
      );
    }
    const resume = screen.getByRole("button", { name: "agent.contextual-start.continue-chat" });
    expect(resume).toHaveClass("bg-accent");
    expect(resume).toHaveProperty("disabled", capture.kind === "pending");
    expect(document.querySelector("textarea")).toHaveValue("Keep my question");
    if (capture.kind === "pending") expect(onStart).not.toHaveBeenCalled();
  });

  test("updates SQL starters without remounting the existing composer", () => {
    const onStart = vi.fn();
    const view = render(<AgentContextualStart {...props} sqlEditor={{ kind: "ready", snapshot: project }} onStart={onStart} onContinue={vi.fn()} />);
    const textarea = document.querySelector("textarea");
    const context = document.querySelector<HTMLButtonElement>("[data-agent-context-trigger]");
    view.rerender(<AgentContextualStart {...props} sqlEditor={{ kind: "ready", snapshot: connected }} onStart={onStart} onContinue={vi.fn()} />);
    expect(document.querySelector("textarea")).toBe(textarea);
    expect(textarea).toHaveValue("Keep my question");
    expect(document.querySelector("[data-agent-context-trigger]")).toBe(context);
    expect(context).toBeDisabled();
  });

  test("renders ready schema questions using the same rows and composer", () => {
    const source = { workspaceName: composerContext.workspace.name, userEmail: "member@example.com", connection: connected.connection! };
    const questions = ["What is the average salary in each department?", "Which departments have the most employees?"];
    vi.mocked(useSqlEditorSuggestions).mockReturnValue({ status: "ready", source, questions });
    const onStart = vi.fn();
    render(<AgentContextualStart {...props} sqlEditor={{ kind: "ready", snapshot: connected }} sqlSuggestionsEnabled onStart={onStart} onContinue={vi.fn()} />);
    const rows = [...document.querySelectorAll<HTMLButtonElement>("[data-agent-contextual-suggestion]")];
    expect(rows.map(row => row.textContent)).toEqual(questions);
    expect(rows.every(row => row.dataset.agentSqlStarter === "database-question")).toBe(true);
    expect(rows[0]).toHaveClass("h-auto", "whitespace-normal", "text-xs");
    fireEvent.click(rows[0]);
    expect(onStart).toHaveBeenCalledWith(questions[0], undefined, "default", undefined, "database-question", source);
    expect(document.querySelector("textarea")).toHaveValue("Keep my question");
    expect(document.querySelector("[data-agent-context-trigger]")).toBeDisabled();
  });

  test("shows three loading rows without temporary generic questions and keeps the draft usable", () => {
    vi.mocked(useSqlEditorSuggestions).mockReturnValue({ status: "loading" });
    const onStart = vi.fn();
    const view = render(<AgentContextualStart {...props} sqlEditor={{ kind: "ready", snapshot: connected }} sqlSuggestionsEnabled onStart={onStart} onContinue={vi.fn()} />);
    const textarea = document.querySelector("textarea");
    expect(document.querySelectorAll("[data-agent-sql-suggestion-placeholder]")).toHaveLength(3);
    expect(screen.queryByRole("button", { name: "agent.contextual-start.sql-explore-database" })).toBeNull();
    expect(document.querySelectorAll("[data-agent-contextual-suggestion]")).toHaveLength(0);
    expect(textarea).toBeEnabled();
    expect(textarea).toHaveValue("Keep my question");
    const questions = ["What is the average employee salary?"];
    vi.mocked(useSqlEditorSuggestions).mockReturnValue({ status: "ready", source: { workspaceName: composerContext.workspace.name, userEmail: "member@example.com", connection: connected.connection! }, questions });
    view.rerender(<AgentContextualStart {...props} sqlEditor={{ kind: "ready", snapshot: connected }} sqlSuggestionsEnabled onStart={onStart} onContinue={vi.fn()} />);
    expect(document.querySelector("textarea")).toBe(textarea);
    expect(document.querySelectorAll("[data-agent-sql-suggestion-placeholder]")).toHaveLength(0);
    expect(screen.getByRole("button", { name: questions[0] })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "agent.contextual-start.sql-explore-database" })).toBeNull();
  });

  test("uses connected defaults only after terminal fallback", () => {
    vi.mocked(useSqlEditorSuggestions).mockReturnValue({ status: "fallback" });
    render(<AgentContextualStart {...props} sqlEditor={{ kind: "ready", snapshot: connected }} sqlSuggestionsEnabled onStart={vi.fn()} onContinue={vi.fn()} />);
    expect(document.querySelectorAll("[data-agent-sql-suggestion-placeholder]")).toHaveLength(0);
    expect([...document.querySelectorAll<HTMLElement>("[data-agent-contextual-suggestion]")].map(row => row.dataset.agentSqlStarter)).toEqual(database);
    expect(document.querySelector("textarea")).toBeEnabled();
  });

  test.each(["selection", "disconnected", "pending", "unconfigured", "hidden", "console"])("does not show database loading rows in %s state", state => {
    vi.mocked(useSqlEditorSuggestions).mockReturnValue({ status: "loading" });
    const capture: AgentSqlContextCapture | undefined = state === "console" ? undefined : state === "pending" ? { kind: "pending" } : { kind: "ready", snapshot: state === "selection" ? selection : state === "disconnected" ? project : connected };
    render(<AgentContextualStart {...props} sqlEditor={capture} canCaptureSqlSelection sqlSuggestionsEnabled={state !== "hidden"} showAIConfigurationRecovery={state === "unconfigured"} contextPending={state === "pending"} onStart={vi.fn()} onContinue={vi.fn()} />);
    expect(document.querySelectorAll("[data-agent-sql-suggestion-placeholder]")).toHaveLength(0);
    if (state === "disconnected") expect(document.querySelector('[data-agent-sql-starter="find-database"]')).not.toBeNull();
  });

  test.each(["selection", "disconnected", "pending", "unconfigured", "hidden", "console"])("keeps generated questions out of %s state", state => {
    vi.mocked(useSqlEditorSuggestions).mockReturnValue({ status: "ready", source: { workspaceName: composerContext.workspace.name, userEmail: "member@example.com", connection: connected.connection! }, questions: ["An old database question?"] });
    const capture: AgentSqlContextCapture | undefined = state === "console" ? undefined : state === "pending" ? { kind: "pending" } : { kind: "ready", snapshot: state === "selection" ? selection : state === "disconnected" ? project : connected };
    render(<AgentContextualStart {...props} sqlEditor={capture} canCaptureSqlSelection sqlSuggestionsEnabled={state !== "hidden"} showAIConfigurationRecovery={state === "unconfigured"} contextPending={state === "pending"} onStart={vi.fn()} onContinue={vi.fn()} />);
    expect(screen.queryByText("An old database question?")).toBeNull();
    expect(vi.mocked(useSqlEditorSuggestions).mock.lastCall?.[0].enabled).toBe(false);
    if (state === "selection") expect(document.querySelector('[data-agent-sql-starter="explain-selection"]')).not.toBeNull();
  });
});

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
  expect(button).toHaveClass("bg-accent", "text-accent-text", "h-7", "px-2", "text-xs");
  fireEvent.click(button);
  expect(onContinue).toHaveBeenCalledOnce();
  expect(onStart).not.toHaveBeenCalled();
});

test("uses compact wrapping actions for contextual suggestions and resource mentions", () => {
  const name = "instances/primary/databases/app";
  renderContext(
    { kind: "database", project: "projects/sample", names: [name] },
    { kind: "database", name, project: "projects/sample", label: "App database" }
  );
  const buttons = document.querySelectorAll("[data-agent-contextual-suggestion], [data-agent-contextual-mention]");
  expect(buttons.length).toBeGreaterThan(1);
  for (const button of buttons) {
    expect(button).toHaveClass("h-auto", "px-2", "py-1.5", "text-xs", "whitespace-normal");
    expect(button).not.toHaveClass("py-2", "text-sm");
    expect(button.querySelector("svg:last-child")).toHaveClass("size-4");
  }
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
