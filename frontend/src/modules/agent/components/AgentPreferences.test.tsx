import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { act } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { getLayerRoot } from "@/components/ui/layer";
import { useAgentStore } from "../store/agent";
import { DEFAULT_AGENT_PREFERENCES, getAgentPreferencesKey } from "../store/preferences";
import { AgentPreferences } from "./AgentPreferences";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

describe("AgentPreferences", () => {
  beforeEach(() => {
    localStorage.clear();
    useAgentStore.setState({ workspaceName: null, userEmail: null });
    useAgentStore.getState().setWorkspaceName("workspaces/example", "member@example.com");
  });

  afterEach(cleanup);

  test("renders only the three agreed preferences with their defaults", () => {
    render(<AgentPreferences />);

    expect(screen.getByRole("region", { name: "agent.preferences.title" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "agent.preferences.title" })).toBeInTheDocument();
    expect(screen.queryByRole("status")).toBeNull();
    expect(screen.queryByText("agent.preferences.saved-locally")).toBeNull();
    expect(screen.getByText("agent.preferences.scope")).toBeVisible();
    expect(screen.queryByText("agent.preferences.send-shortcut")).toBeNull();
    expect(screen.getByRole("switch", { name: "agent.preferences.send-with-enter" })).toBeChecked();
    expect(screen.getByRole("spinbutton", { name: "agent.preferences.token-limit" })).toHaveValue(100);
    expect(screen.getByRole("spinbutton")).toHaveAccessibleDescription("agent.preferences.thousand-tokens");
    expect(screen.getByRole("radio", { name: "agent.preferences.show-landing" })).toBeChecked();
    expect(screen.getAllByRole("radio")).toHaveLength(2);
    expect(screen.queryByText(/query preview/i)).toBeNull();
  });

  test("saves valid changes locally for the current owner", () => {
    render(<AgentPreferences />);
    fireEvent.click(screen.getByRole("switch", { name: "agent.preferences.send-with-enter" }));
    fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "42" } });
    fireEvent.click(screen.getByRole("radio", { name: "agent.preferences.continue-last-chat" }));

    const expected = { sendWithEnter: false, tokenLimitPerRun: 42000, openingBehavior: "continue" };
    expect(useAgentStore.getState().preferences).toEqual(expected);
    const key = getAgentPreferencesKey("workspaces/example", "member@example.com");
    expect(JSON.parse(localStorage.getItem(key!)!)).toEqual(expected);
  });

  test("does not toggle Send with Enter from blank row space or helper text", () => {
    render(<AgentPreferences />);
    const control = screen.getByRole("switch", { name: "agent.preferences.send-with-enter" });
    const row = control.closest("[data-agent-preference-row=send]")!;
    const key = getAgentPreferencesKey("workspaces/example", "member@example.com")!;
    const savedPreferences = localStorage.getItem(key);

    fireEvent.click(row);
    expect(control).toBeChecked();
    fireEvent.click(row.firstElementChild!);
    expect(control).toBeChecked();
    fireEvent.click(screen.getByText("agent.preferences.enter-help"));
    expect(control).toBeChecked();
    expect(useAgentStore.getState().preferences.sendWithEnter).toBe(true);
    expect(localStorage.getItem(key)).toBe(savedPreferences);

    fireEvent.click(screen.getByText("agent.preferences.send-with-enter"));
    expect(control).not.toBeChecked();
    expect(useAgentStore.getState().preferences.sendWithEnter).toBe(false);
    fireEvent.click(control);
    expect(control).toBeChecked();
    expect(useAgentStore.getState().preferences.sendWithEnter).toBe(true);
  });

  test("keeps invalid numeric drafts visible without replacing the saved limit", () => {
    render(<AgentPreferences />);
    const input = screen.getByRole("spinbutton");
    fireEvent.change(input, { target: { value: "0" } });

    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("alert")).toHaveTextContent("agent.preferences.invalid-token-limit");
    expect(useAgentStore.getState().preferences.tokenLimitPerRun).toBe(100000);
    fireEvent.click(screen.getByRole("switch"));
    expect(input).toHaveValue(0);
    expect(input).toHaveAttribute("aria-invalid", "true");

    fireEvent.change(input, { target: { value: "50" } });
    expect(input).not.toHaveAttribute("aria-invalid", "true");
    expect(screen.queryByRole("alert")).toBeNull();
    expect(useAgentStore.getState().preferences.tokenLimitPerRun).toBe(50000);
  });

  test.each<[number, string]>([
    [1, "0.001"],
    [12345, "12.345"],
    [100010, "100.01"],
    [Number.MAX_SAFE_INTEGER, "9007199254740.991"],
  ])("preserves a stored limit of %s tokens when displaying and editing %s K tokens", (tokens, displayed) => {
    useAgentStore.getState().updatePreferences({ tokenLimitPerRun: tokens });
    render(<AgentPreferences />);
    const input = screen.getByRole("spinbutton") as HTMLInputElement;
    expect(input.value).toBe(displayed);
    expect(input).toBeValid();
    expect(input).not.toHaveAttribute("aria-invalid", "true");

    fireEvent.change(input, { target: { value: displayed } });
    fireEvent.click(screen.getByRole("switch"));
    expect(useAgentStore.getState().preferences.tokenLimitPerRun).toBe(tokens);
    expect(input.value).toBe(displayed);
    const key = getAgentPreferencesKey("workspaces/example", "member@example.com")!;
    expect(JSON.parse(localStorage.getItem(key)!).tokenLimitPerRun).toBe(tokens);
  });

  test.each<[string, number]>([
    ["150", 150000],
    ["12.345", 12345],
    ["0.001", 1],
    [".125", 125],
    ["1.0000", 1000],
    ["9007199254740.991", Number.MAX_SAFE_INTEGER],
  ])("stores %s K tokens as exactly %s tokens", (entered, tokens) => {
    render(<AgentPreferences />);
    fireEvent.change(screen.getByRole("spinbutton"), { target: { value: entered } });
    expect(screen.getByRole("spinbutton")).not.toHaveAttribute("aria-invalid", "true");
    expect(useAgentStore.getState().preferences.tokenLimitPerRun).toBe(tokens);
    const key = getAgentPreferencesKey("workspaces/example", "member@example.com")!;
    expect(JSON.parse(localStorage.getItem(key)!).tokenLimitPerRun).toBe(tokens);
  });

  test.each(["", "0", "-1", "0.0001", "1.2345", "9007199254740.992"])("keeps invalid K-token draft %s without changing the saved limit", (entered) => {
    render(<AgentPreferences />);
    const input = screen.getByRole("spinbutton") as HTMLInputElement;
    fireEvent.change(input, { target: { value: entered } });
    expect(input.value).toBe(entered);
    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("alert")).toHaveTextContent("agent.preferences.invalid-token-limit");
    expect(input).toHaveAccessibleDescription("agent.preferences.thousand-tokens agent.preferences.invalid-token-limit");
    expect(useAgentStore.getState().preferences.tokenLimitPerRun).toBe(100000);
  });

  test("resets preferences and validation only after confirmation without changing chats", async () => {
    useAgentStore.getState().updatePreferences({ sendWithEnter: false, tokenLimitPerRun: 10000, openingBehavior: "continue" });
    const chat = useAgentStore.getState().createChat({ title: "Keep this discussion" });
    render(<AgentPreferences />);
    fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "" } });
    const savedPreferences = localStorage.getItem(getAgentPreferencesKey("workspaces/example", "member@example.com")!);
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "agent.preferences.reset" })));

    const confirmation = screen.getByRole("dialog", { name: "agent.preferences.reset-confirmation" });
    expect(getLayerRoot("agent")).toContainElement(confirmation);
    expect(getLayerRoot("overlay")).not.toContainElement(confirmation);
    expect(useAgentStore.getState().preferences).toEqual({ sendWithEnter: false, tokenLimitPerRun: 10000, openingBehavior: "continue" });
    expect(localStorage.getItem(getAgentPreferencesKey("workspaces/example", "member@example.com")!)).toBe(savedPreferences);
    expect(screen.getByRole("spinbutton")).toHaveValue(null);
    expect(screen.getByRole("alert")).toHaveTextContent("agent.preferences.invalid-token-limit");

    await act(async () => fireEvent.click(within(confirmation).getByRole("button", { name: "common.reset" })));

    expect(useAgentStore.getState().preferences).toEqual(DEFAULT_AGENT_PREFERENCES);
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(screen.getByRole("spinbutton")).toHaveValue(100);
    expect(screen.queryByRole("alert")).toBeNull();
    expect(useAgentStore.getState().getChat(chat.id)?.title).toBe("Keep this discussion");
    expect(useAgentStore.getState().currentChatId).toBe(chat.id);
  });

  test("cancels reset without changing preferences or invalid drafts and returns focus", async () => {
    const preferences = { sendWithEnter: false, tokenLimitPerRun: 10000, openingBehavior: "continue" } as const;
    useAgentStore.getState().updatePreferences(preferences);
    render(<AgentPreferences />);
    fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "0" } });
    const trigger = screen.getByRole("button", { name: "agent.preferences.reset" });
    trigger.focus();
    await act(async () => fireEvent.click(trigger));
    const confirmation = screen.getByRole("dialog", { name: "agent.preferences.reset-confirmation" });
    await act(async () => fireEvent.click(within(confirmation).getByRole("button", { name: "common.cancel" })));

    expect(screen.queryByRole("dialog")).toBeNull();
    expect(useAgentStore.getState().preferences).toEqual(preferences);
    expect(screen.getByRole("spinbutton")).toHaveValue(0);
    expect(screen.getByRole("alert")).toHaveTextContent("agent.preferences.invalid-token-limit");
    expect(trigger).toHaveFocus();
  });

  test("dismisses reset confirmation with Escape without resetting", async () => {
    useAgentStore.getState().updatePreferences({ tokenLimitPerRun: 10000 });
    render(<AgentPreferences />);
    const trigger = screen.getByRole("button", { name: "agent.preferences.reset" });
    await act(async () => fireEvent.click(trigger));
    const confirmation = screen.getByRole("dialog", { name: "agent.preferences.reset-confirmation" });
    await act(async () => fireEvent.keyDown(confirmation, { key: "Escape" }));

    expect(screen.queryByRole("dialog")).toBeNull();
    expect(useAgentStore.getState().preferences.tokenLimitPerRun).toBe(10000);
  });

  test("reports when changes could not be saved in the browser", () => {
    render(<AgentPreferences />);
    act(() => useAgentStore.setState({ preferencesStorageError: true }));

    expect(screen.getByRole("alert")).toHaveTextContent("agent.preferences.storage-failed");
    expect(screen.queryByText("agent.preferences.saved-locally")).toBeNull();
  });
});
