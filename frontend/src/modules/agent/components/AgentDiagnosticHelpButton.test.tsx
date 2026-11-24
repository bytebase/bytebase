import { create } from "@bufbuild/protobuf";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { useAppStore } from "@/stores/app";
import { SettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import { useAgentStore } from "../store/agent";
import type { AgentDiagnosticContext } from "../logic/diagnosticHelp";
import { AgentDiagnosticHelpButton } from "./AgentDiagnosticHelpButton";

const mocks = vi.hoisted(() => ({ fetchConfiguration: vi.fn(), label: "Ask Agent" }));
vi.mock("@/app/router", () => ({ router: { currentRoute: { value: { fullPath: "/projects/other/plans/999" } } } }));
vi.mock("@/stores/app", async () => {
  const { create: createStore } = await import("zustand");
  const useAppStore = createStore(() => ({
    workspace: { name: "workspaces/example" },
    currentUser: {
      name: "users/agent@example.com",
      email: "agent@example.com",
      workspace: "workspaces/example",
    },
    serverInfo: { aiEnabled: true },
    settingsByName: {},
    getOrFetchSettingByName: mocks.fetchConfiguration,
  }));
  return { useAppStore };
});
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
  const app = useAppStore.getState();
  useAppStore.setState({
    workspace: { ...app.workspace!, name: "workspaces/example" },
    currentUser: {
      ...app.currentUser!,
      name: "users/agent@example.com",
      email: "agent@example.com",
      workspace: "workspaces/example",
    },
    serverInfo: { ...app.serverInfo!, aiEnabled: true },
    settingsByName: {},
  });
  const store = useAgentStore.getState();
  store.setWorkspaceName("workspaces/example", "agent@example.com");
  useAgentStore.setState({ chats: [], currentChatId: null, messagesByChatId: {}, queuedUserMessagesByChatId: {}, visible: false, storagePressure: undefined, initializationFailureIdentity: undefined });
  mocks.fetchConfiguration.mockClear();
  mocks.label = "Ask Agent";
});
afterEach(cleanup);

describe("AgentDiagnosticHelpButton", () => {
  test("does no work on render and queues exactly one bound diagnostic on click", () => {
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

  test("hides help while AI is explicitly disabled and restores it when enabled", () => {
    useAppStore.setState({
      serverInfo: { ...useAppStore.getState().serverInfo!, aiEnabled: false },
    });
    render(<AgentDiagnosticHelpButton context={context} />);
    expect(screen.queryByRole("button")).toBeNull();
    expect(useAgentStore.getState().chats).toEqual([]);
    expect(mocks.fetchConfiguration).not.toHaveBeenCalled();
    act(() => {
      useAppStore.setState({
        serverInfo: { ...useAppStore.getState().serverInfo!, aiEnabled: true },
      });
    });
    expect(screen.getByRole("button", { name: "Ask Agent" })).toBeVisible();
  });

  test("hides help when the cached AI setting is disabled", () => {
    const disabled = create(SettingSchema, {
      name: "settings/AI",
      value: { value: { case: "ai", value: { enabled: false } } },
    });
    useAppStore.setState({ settingsByName: { [disabled.name]: disabled } });
    render(<AgentDiagnosticHelpButton context={context} />);
    expect(screen.queryByRole("button")).toBeNull();
    act(() => useAppStore.setState({ settingsByName: {} }));
    expect(screen.getByRole("button", { name: "Ask Agent" })).toBeVisible();
    expect(mocks.fetchConfiguration).not.toHaveBeenCalled();
  });

  test("keeps help available when AI metadata has not loaded", () => {
    useAppStore.setState({ serverInfo: undefined, settingsByName: {} });
    render(<AgentDiagnosticHelpButton context={context} />);
    expect(screen.getByRole("button", { name: "Ask Agent" })).toBeVisible();
    expect(mocks.fetchConfiguration).not.toHaveBeenCalled();
  });

  test.each(["budget", "quota", "unavailable"] as const)(
    "hides help for known %s storage pressure without changing history",
    (reason) => {
      const chats = useAgentStore.getState().chats;
      useAgentStore.setState({
        storagePressure: { reason, phase: "saving", bytes: 9_000_000, limitBytes: 3_000_000 },
      });
      render(<AgentDiagnosticHelpButton context={context} />);
      expect(screen.queryByRole("button")).toBeNull();
      expect(useAgentStore.getState().chats).toBe(chats);
      act(() => useAgentStore.setState({ storagePressure: undefined }));
      expect(screen.getByRole("button", { name: "Ask Agent" })).toBeVisible();
      expect(mocks.fetchConfiguration).not.toHaveBeenCalled();
    }
  );

  test.each([
    { workspaceName: "workspaces/other", userEmail: "agent@example.com" },
    { workspaceName: "workspaces/example", userEmail: "other@example.com" },
  ])("hides help until Agent ownership matches %j", (scope) => {
    useAgentStore.setState(scope);
    render(<AgentDiagnosticHelpButton context={context} />);
    expect(screen.queryByRole("button")).toBeNull();
    act(() => useAgentStore.setState({ workspaceName: "workspaces/example", userEmail: "agent@example.com" }));
    expect(screen.getByRole("button", { name: "Ask Agent" })).toBeVisible();
  });

  test("does not treat an uninitialized Agent as a known failure", () => {
    useAgentStore.setState({ workspaceName: null, userEmail: null });
    render(<AgentDiagnosticHelpButton context={context} />);
    expect(screen.getByRole("button", { name: "Ask Agent" })).toBeVisible();
    expect(useAgentStore.getState().workspaceName).toBeNull();
    expect(mocks.fetchConfiguration).not.toHaveBeenCalled();
  });

  test("hides a known initialization failure only for its owner and restores help after recovery", () => {
    useAgentStore.setState({ workspaceName: null, userEmail: null });
    const chats = useAgentStore.getState().chats;
    render(<AgentDiagnosticHelpButton context={context} />);
    act(() => useAgentStore.setState({
      initializationFailureIdentity: JSON.stringify(["workspaces/example", "agent@example.com"]),
    }));
    expect(screen.queryByRole("button")).toBeNull();
    act(() => useAgentStore.setState({
      initializationFailureIdentity: JSON.stringify(["workspaces/example", "other@example.com"]),
    }));
    expect(screen.getByRole("button", { name: "Ask Agent" })).toBeVisible();
    act(() => useAgentStore.setState({
      workspaceName: "workspaces/example",
      userEmail: "agent@example.com",
      initializationFailureIdentity: undefined,
    }));
    expect(screen.getByRole("button", { name: "Ask Agent" })).toBeVisible();
    expect(useAgentStore.getState().chats).toBe(chats);
    expect(mocks.fetchConfiguration).not.toHaveBeenCalled();
  });

  test("hides help when the current actor no longer belongs to the workspace", () => {
    render(<AgentDiagnosticHelpButton context={context} />);
    act(() => useAppStore.setState({ currentUser: { ...useAppStore.getState().currentUser!, workspace: "workspaces/other" } }));
    expect(screen.queryByRole("button")).toBeNull();
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

  test("keeps icon-only help labeled and compact without displaying duplicate text", () => {
    render(<AgentDiagnosticHelpButton context={context} iconOnly />);
    const help = screen.getByRole("button", { name: "Ask Agent" });
    expect(help).toHaveAttribute("title", "Ask Agent");
    expect(help).toHaveClass("h-7", "w-7");
    expect(help.querySelector("svg")).toBeInTheDocument();
    expect(help.querySelector("span")).toBeNull();
  });
});
