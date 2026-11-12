import { SHA256 } from "crypto-js";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  type AgentSqlFixSource,
  type AgentSqlFixTargetState,
  notifyAgentSqlFixTargetChanged,
  registerAgentSqlFixTarget,
} from "../logic/sqlFix";
import { useAgentStore } from "../store/agent";
import { AgentSqlFixReviewAction } from "./AgentSqlFixReviewAction";

const mocks = vi.hoisted(() => ({ notify: vi.fn() }));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/stores", () => ({ pushNotification: mocks.notify }));
vi.mock("@/stores/app", () => {
  const state = {
    currentUser: { name: "users/member@example.com" },
    workspaceResourceName: () => "workspaces/demo",
  };
  const useAppStore = <T,>(selector: (value: typeof state) => T) => selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});
vi.mock("@/components/monaco/DiffMonaco", () => ({
  DiffMonaco: ({ original, modified }: { original: string; modified: string }) => (
    <div><pre aria-label="original SQL">{original}</pre><pre aria-label="proposed SQL">{modified}</pre></div>
  ),
}));

const original = "CREATE TABLE sample (\n  id INT,\n);";
const proposed = "CREATE TABLE sample (\n  id INT\n);";
const hash = SHA256(original).toString();
const resource = { kind: "plan" as const, name: "projects/p/plans/42", project: "projects/p", label: "projects/p/plans/42" };
let chatId: string;
let source: AgentSqlFixSource;
let currentSql: string;
let targetStatus: AgentSqlFixTargetState["status"];
let unregister: () => void;
let revealCount: number;
let applyCount: number;

beforeEach(() => {
  mocks.notify.mockClear();
  useAgentStore.setState({
    workspaceName: "workspaces/demo", userEmail: "member@example.com",
    chats: [], messagesByChatId: {}, currentChatId: null,
    sqlFixSourcesByChatId: {}, sqlFixDraftsById: {},
    queuedUserMessagesByChatId: {}, queuedUserMessageInFlightByChatId: {},
    abortControllersByChatId: {}, remoteBusyChatIds: {}, visible: true, minimized: false,
  });
  chatId = useAgentStore.getState().createChat({ context: resource.project, workspaceName: "workspaces/demo", capability: { kind: "diagnostic", resource } }).id;
  source = {
    identity: JSON.stringify(["workspaces/demo", "users/member@example.com"]),
    pageKey: "projects/p/plans/42", resource, sourceId: "projects/p/plans/42/planCheckRun",
    plan: resource.name, specId: "spec-1", sheet: `projects/p/sheets/${hash}`,
    targets: ["instances/i/databases/app"], sha256: hash, sql: original,
  };
  useAgentStore.setState({ sqlFixDraftsById: { "fix-1": { id: "fix-1", chatId, source, sql: proposed } } });
  currentSql = original; targetStatus = "ready"; revealCount = 0; applyCount = 0;
  unregister = registerAgentSqlFixTarget({
    capture: () => undefined,
    reveal: () => { revealCount++; return true; },
    inspect: () => ({ status: targetStatus, currentSql }),
    apply: (_source, sql) => {
      applyCount++;
      if (targetStatus !== "ready" || currentSql !== original) return false;
      currentSql = sql; return true;
    },
  });
});

afterEach(() => { cleanup(); unregister(); });

const show = () => render(<AgentSqlFixReviewAction action={{ kind: "review-sql-fix", draftId: "fix-1" }} chatId={chatId} />);
const open = () => fireEvent.click(screen.getByRole("button", { name: "agent.sql-fix.review" }));

describe("AgentSqlFixReviewAction", () => {
  test("reviews exact original/proposed SQL without changing the editor until Apply", async () => {
    show();
    expect(revealCount).toBe(0);
    expect(applyCount).toBe(0);
    open();
    expect(revealCount).toBe(1);
    expect((await screen.findByLabelText("original SQL")).textContent).toBe(original);
    expect(screen.getByLabelText("proposed SQL").textContent).toBe(proposed);
    expect(currentSql).toBe(original);
    expect(applyCount).toBe(0);
    fireEvent.click(screen.getByRole("button", { name: "agent.sql-fix.apply" }));
    expect(currentSql).toBe(proposed);
    expect(applyCount).toBe(1);
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(useAgentStore.getState().minimized).toBe(true);
  });

  test("Cancel leaves the editor and chat unchanged", () => {
    show(); open();
    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
    expect(currentSql).toBe(original);
    expect(applyCount).toBe(0);
    expect(useAgentStore.getState().minimized).toBe(false);
  });

  test.each(["stale", "readonly", "permission-denied", "loading", "unavailable"] as const)("blocks Apply for a %s editor", (status) => {
    targetStatus = status;
    show(); open();
    expect((screen.getByRole("button", { name: "agent.sql-fix.apply" }) as HTMLButtonElement).disabled).toBe(true);
    expect(currentSql).toBe(original);
    expect(applyCount).toBe(0);
  });

  test("rejects a target that becomes stale immediately before Apply", () => {
    show(); open();
    currentSql = "SELECT 1;"; targetStatus = "stale";
    fireEvent.click(screen.getByRole("button", { name: "agent.sql-fix.apply" }));
    expect(currentSql).toBe("SELECT 1;");
    expect(applyCount).toBe(0);
    expect(screen.getByRole("dialog")).toBeDefined();
    expect(screen.getByRole("alert").textContent).toContain("agent.sql-fix.stale");
  });

  test("updates Apply eligibility when editor permissions change", () => {
    show(); open();
    act(() => { targetStatus = "permission-denied"; notifyAgentSqlFixTargetChanged(); });
    expect((screen.getByRole("button", { name: "agent.sql-fix.apply" }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByRole("alert").textContent).toContain("agent.sql-fix.permission-denied");
  });

  test("does not reconstruct a missing runtime source from a persisted action", () => {
    useAgentStore.setState({ sqlFixDraftsById: {} });
    show(); open();
    expect(screen.queryByLabelText("original SQL")).toBeNull();
    expect(screen.getByRole("alert").textContent).toContain("agent.sql-fix.original-unavailable");
    expect(screen.getAllByText("agent.sql-fix.original-unavailable")).toHaveLength(1);
    expect((screen.getByRole("button", { name: "agent.sql-fix.apply" }) as HTMLButtonElement).disabled).toBe(true);
    expect(applyCount).toBe(0);
  });

  test("rejects Apply after another chat becomes selected", () => {
    show(); open();
    act(() => { useAgentStore.getState().createChat({ context: "projects/other" }); });
    expect((screen.getByRole("button", { name: "agent.sql-fix.apply" }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.queryByLabelText("original SQL")).toBeNull();
    expect(applyCount).toBe(0);
  });

  test("does not minimize or claim success when the editor rejects Apply", () => {
    unregister();
    unregister = registerAgentSqlFixTarget({ capture: () => undefined, reveal: () => {}, inspect: () => ({ status: "ready", currentSql }), apply: () => false });
    show(); open();
    fireEvent.click(screen.getByRole("button", { name: "agent.sql-fix.apply" }));
    expect(currentSql).toBe(original);
    expect(screen.getByRole("dialog")).toBeDefined();
    expect(screen.getByRole("alert").textContent).toContain("agent.sql-fix.apply-failed");
    expect(useAgentStore.getState().minimized).toBe(false);
  });

  test("waits for guarded navigation before opening the Agent-layer review", async () => {
    let finishReveal!: () => void;
    unregister();
    unregister = registerAgentSqlFixTarget({ capture: () => undefined, reveal: () => new Promise<void>(resolve => { finishReveal = resolve; }), inspect: () => ({ status: "ready", currentSql }), apply: () => false });
    show(); open();
    expect(screen.queryByRole("dialog")).toBeNull();
    await act(async () => finishReveal());
    expect(screen.getByRole("dialog")).toBeDefined();
    expect(applyCount).toBe(0);
  });

  test("does not open a pending review after the selected chat changes", async () => {
    let finishReveal!: () => void;
    unregister();
    unregister = registerAgentSqlFixTarget({ capture: () => undefined, reveal: () => new Promise<void>(resolve => { finishReveal = resolve; }), inspect: () => ({ status: "ready", currentSql }), apply: () => false });
    show(); open();
    await act(async () => {
      useAgentStore.getState().createChat({ context: "projects/other" });
      finishReveal();
    });
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(applyCount).toBe(0);
  });

  test("does not cover an unsaved-leave confirmation after canceled navigation", async () => {
    unregister();
    unregister = registerAgentSqlFixTarget({ capture: () => undefined, reveal: async () => false, inspect: () => ({ status: "ready", currentSql }), apply: () => false });
    show();
    await act(async () => open());
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(applyCount).toBe(0);
  });

  test("hands off to the current SQL without applying the stale proposal", async () => {
    targetStatus = "stale";
    show(); open();
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "agent.sql-fix.review-current" })));
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(useAgentStore.getState().minimized).toBe(true);
    expect(applyCount).toBe(0);
  });

  test("keeps the Agent available when revealing current SQL fails", async () => {
    let fail = false;
    targetStatus = "stale";
    unregister();
    unregister = registerAgentSqlFixTarget({ capture: () => undefined, reveal: () => fail ? Promise.reject(new Error("navigation failed")) : undefined, inspect: () => ({ status: "stale", currentSql }), apply: () => false });
    show(); open(); fail = true;
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "agent.sql-fix.review-current" })));
    expect(screen.getByRole("alert").textContent).toContain("agent.sql-fix.apply-failed");
    expect(useAgentStore.getState().minimized).toBe(false);
    expect(applyCount).toBe(0);
  });

  test("does not minimize when the old page cannot reveal current SQL", async () => {
    targetStatus = "stale";
    unregister();
    unregister = registerAgentSqlFixTarget({
      capture: () => undefined,
      reveal: () => undefined,
      inspect: () => ({ status: "stale", currentSql }),
      apply: () => false,
    });
    show(); open();
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "agent.sql-fix.review-current" })));
    expect(useAgentStore.getState().minimized).toBe(false);
    expect(applyCount).toBe(0);
  });
});
