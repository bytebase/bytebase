import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { useAgentStore } from "../store/agent";
import type { AgentDiagnosticContext } from "../logic/diagnosticHelp";
import { AgentDiagnosticHelpButton } from "./AgentDiagnosticHelpButton";

const mocks = vi.hoisted(() => ({ fetchConfiguration: vi.fn(), label: "Ask Agent" }));
vi.mock("@/app/router", () => ({ router: { currentRoute: { value: { fullPath: "/projects/other/plans/999" } } } }));
vi.mock("@/stores/app", () => ({ useAppStore: { getState: () => ({ workspace: { name: "workspaces/example" }, serverInfo: { aiEnabled: false } }) } }));
vi.mock("../logic/chatContext", () => ({ getCurrentChatContext: () => "projects/other" }));
vi.mock("@/stores/modules/v1/setting", () => ({ useSettingV1Store: { getState: () => ({ getOrFetchSettingByName: mocks.fetchConfiguration }) } }));
vi.mock("react-i18next", async (importOriginal) => ({ ...await importOriginal<typeof import("react-i18next")>(), useTranslation: () => ({ t: () => mocks.label }) }));

const context: AgentDiagnosticContext = {
  resource: { kind: "plan", name: "projects/example/plans/42", project: "projects/example", label: "password=not-persisted" },
  kind: "task-run",
  sourceId: "projects/example/rollouts/42/stages/1/tasks/2/taskRuns/3",
  summary: "Task Run failed",
  detail: "password=not-queued",
};

beforeEach(() => {
  localStorage.clear();
  const store = useAgentStore.getState();
  store.setWorkspaceName("workspaces/example", "agent@example.com");
  useAgentStore.setState({ chats: [], currentChatId: null, messagesByChatId: {}, queuedUserMessagesByChatId: {}, visible: false });
  mocks.fetchConfiguration.mockClear();
  mocks.label = "Ask Agent";
});
afterEach(cleanup);

describe("AgentDiagnosticHelpButton", () => {
  test("does no work on render and queues exactly one bound diagnostic on click even when AI is unavailable", () => {
    render(<AgentDiagnosticHelpButton context={context} />);
    expect(useAgentStore.getState().chats).toEqual([]);
    expect(mocks.fetchConfiguration).not.toHaveBeenCalled();
    const help = screen.getByRole("button", { name: "Ask Agent" });
    expect(help).toHaveClass("h-7");
    expect(help).not.toBeDisabled();
    fireEvent.click(help);
    const store = useAgentStore.getState();
    expect(store.visible).toBe(true);
    expect(store.chats).toHaveLength(1);
    const chat = store.chats[0];
    expect(chat.context).toBe("projects/example");
    const queued = store.queuedUserMessagesByChatId[chat.id];
    expect(chat.capability).toEqual({ kind: "diagnostic", resource: { ...context.resource, label: context.resource.name } });
    expect(queued.resourceReferences).toEqual([chat.capability!.resource]);
    expect(queued.runMode).toBe("read-only");
    expect(JSON.stringify([chat.capability, queued])).not.toContain("not-persisted");
    expect(queued.text).not.toContain("not-queued");
    expect(Object.keys(store.queuedUserMessagesByChatId)).toEqual([chat.id]);
    expect(mocks.fetchConfiguration).not.toHaveBeenCalled();
    store.saveState();
    expect(JSON.stringify(localStorage)).not.toContain("not-persisted");
    expect(JSON.stringify(localStorage)).not.toContain("not-queued");
  });

  test("suppresses invalid resources instead of selecting the live page Plan", () => {
    render(<AgentDiagnosticHelpButton context={{ ...context, resource: { ...context.resource, name: "projects/example/plans/create" } }} />);
    expect(screen.queryByRole("button")).toBeNull();
    expect(useAgentStore.getState().chats).toEqual([]);
  });

  test("keeps a long localized label bounded and accessible", () => {
    mocks.label = "Ask Agent to investigate this particular database operation failure";
    render(<AgentDiagnosticHelpButton context={context} />);
    const help = screen.getByRole("button", { name: mocks.label });
    expect(help).toHaveClass("max-w-full", "min-w-0", "h-7");
    expect(help).toHaveAttribute("title", mocks.label);
    expect(help.querySelector("span")).toHaveClass("truncate");
  });
});
