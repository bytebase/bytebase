import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { I18nextProvider } from "react-i18next";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE } from "@/app/router/handles";
import i18n from "@/lib/i18n";
import { MaskingExemptionDraftButton } from "./MaskingExemptionDraftButton";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  userName: "users/member@example.com",
  workspaceName: "workspaces/sample",
  currentChatId: "chat-1",
  chat: { id: "chat-1", workspaceName: "workspaces/sample" },
  path: "/initial",
  resolve: vi.fn(), push: vi.fn(),
  getDraft: vi.fn(), getSource: vi.fn(), revalidate: vi.fn(), consume: vi.fn(),
  ready: vi.fn(), open: vi.fn(), apply: vi.fn(), minimize: vi.fn(),
  updatePolicy: vi.fn(), recordHandoff: vi.fn(),
  handoff: undefined as { draftId: string; outcome: "opened" | "applied"; created?: true } | undefined,
}));

vi.mock("@/stores/app", () => {
  const state = {
    get currentUser() { return { name: mocks.userName }; },
    workspaceResourceName: () => mocks.workspaceName,
  };
  const useAppStore = <T,>(selector: (value: typeof state) => T) => selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});
vi.mock("../store/agent", () => {
  const state = {
    get currentChatId() { return mocks.currentChatId; },
    visible: true,
    get workspaceName() { return mocks.workspaceName; },
    get userEmail() { return mocks.userName; },
    getChat: (id: string) => id === mocks.chat.id ? mocks.chat : null,
    getMessages: () => [{ id: "message-1", metadata: { maskingExemptionDraftHandoff: mocks.handoff } }],
    recordMaskingExemptionDraftHandoff: mocks.recordHandoff,
    minimize: mocks.minimize,
  };
  const useAgentStore = <T,>(selector: (value: typeof state) => T) => selector(state);
  useAgentStore.getState = () => state;
  return { useAgentStore };
});
vi.mock("@/app/router", () => ({ router: {
  resolve: mocks.resolve, push: mocks.push,
  get currentRoute() { return { value: { fullPath: mocks.path } }; },
} }));
vi.mock("@/api", () => ({ orgPolicyServiceClientConnect: { updatePolicy: mocks.updatePolicy } }));
vi.mock("../logic/maskingExemptionDraft", () => ({
  getMaskingExemptionDraft: mocks.getDraft,
  getMaskingExemptionDraftSource: mocks.getSource,
  revalidateMaskingExemptionDraft: mocks.revalidate,
  consumeMaskingExemptionDraft: mocks.consume,
}));
vi.mock("../logic/maskingExemptionDraftTarget", () => ({
  isMaskingExemptionDraftTargetReady: mocks.ready,
  openMaskingExemptionDraftInEditor: mocks.open,
  applyMaskingExemptionDraftInEditor: mocks.apply,
}));

const context = { workspaceName: "workspaces/sample", userName: "users/member@example.com", chatId: "chat-1" };
const targetPath = "/projects/sample/masking-exemption/create";
const draft = {
  project: "projects/sample",
  scope: { kind: "database", database: "instances/sample/databases/sample" },
  members: ["user:member@example.com", "user:reviewer@example.com"],
  condition: 'resource.column_name == "email"',
  description: "Temporary investigation",
  expirationTimestamp: "2026-12-01T00:00:00Z",
};
const source = { sessionId: "editor-1", revision: "revision-1", project: "projects/sample", chatId: "chat-1" };
const roots: Root[] = [];
function renderButton(locale = "en-US") {
  const container = document.createElement("div");
  const root = createRoot(container);
  roots.push(root);
  act(() => root.render(
    <I18nextProvider i18n={i18n.cloneInstance({ lng: locale, initAsync: false })}>
      <MaskingExemptionDraftButton draftId="draft-1" chatId="chat-1" messageId="message-1" />
    </I18nextProvider>
  ));
  const button = container.querySelector("button");
  if (!button) throw new Error("Missing exemption draft action");
  return { container, button, unmount: () => {
    act(() => root.unmount());
    roots.splice(roots.indexOf(root), 1);
  } };
}

beforeEach(() => {
  vi.resetAllMocks();
  mocks.userName = context.userName;
  mocks.workspaceName = context.workspaceName;
  mocks.currentChatId = context.chatId;
  mocks.chat = { id: context.chatId, workspaceName: context.workspaceName };
  mocks.path = "/initial";
  mocks.handoff = undefined;
  mocks.getDraft.mockReturnValue(draft);
  mocks.getSource.mockReturnValue(undefined);
  mocks.revalidate.mockResolvedValue(draft);
  mocks.ready.mockReturnValue(true);
  mocks.resolve.mockReturnValue({ fullPath: targetPath });
  mocks.push.mockImplementation(async () => { mocks.path = targetPath; });
  mocks.recordHandoff.mockImplementation((_chatId, _messageId, draftId, outcome) => { mocks.handoff = { draftId, outcome }; });
});
afterEach(() => {
  act(() => { for (const root of roots.splice(0)) root.unmount(); });
  vi.useRealTimers();
});

describe("MaskingExemptionDraftButton", () => {
  test("uses the Grant Exemption summary labels and no-expiration text", () => {
    mocks.getDraft.mockReturnValue({ ...draft, expirationTimestamp: null });
    const { container, button } = renderButton();
    expect(Array.from(container.querySelectorAll("dt"), (element) => element.textContent)).toEqual([
      "Members", "Resources", "Expiration",
    ]);
    expect(container.textContent).toContain("Never expires");
    expect(button.textContent).toBe("Open exemption draft");
    expect(container.textContent).toContain("Only fills the unsaved form. Review it before selecting Grant Exemption.");
    expect(container.textContent).not.toContain("agent.masking-exemption-draft.");
  });

  test.each(["en-US", "es-ES", "ja-JP", "vi-VN", "zh-CN"])("localizes the draft summary and action states in %s", async (locale) => {
    mocks.getDraft.mockReturnValue({ ...draft, expirationTimestamp: null });
    const { container, button } = renderButton(locale);
    expect(container.textContent).not.toMatch(/(?:agent\.masking-exemption-draft|common|settings\.sensitive-data)\./);
    expect(container.querySelectorAll("dt")).toHaveLength(3);
    await act(async () => button.click());
    expect(button.disabled).toBe(true);
    expect(button.querySelector("svg")).toHaveClass("lucide-check");
    expect(container.textContent).not.toContain("agent.masking-exemption-draft.");
  });

  test("opens the exact project create form only after a member click and never saves", async () => {
    const { button, container } = renderButton();
    expect(button.textContent).toBe("Open exemption draft");
    expect(button).toHaveClass("text-accent", "bg-transparent", "h-7");
    expect(button.querySelector("svg")).toHaveClass("size-4", "lucide-arrow-up-right");
    expect(container.textContent).toContain(draft.members.join(", "));
    expect(container.textContent).toContain(draft.scope.database);
    expect(container.textContent).toContain(draft.expirationTimestamp);
    expect(container.querySelector("code")?.textContent).toBe(draft.condition);
    expect(mocks.revalidate).not.toHaveBeenCalled();
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();

    await act(async () => button.click());
    expect(mocks.resolve).toHaveBeenCalledWith({ name: PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE, params: { projectId: "sample" } });
    expect(mocks.push).toHaveBeenCalledWith({ name: PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE, params: { projectId: "sample" } });
    expect(mocks.open).toHaveBeenCalledWith(draft, context, { draftId: "draft-1", messageId: "message-1" });
    expect(mocks.consume).toHaveBeenCalledWith("draft-1", context);
    expect(mocks.recordHandoff).toHaveBeenCalledWith("chat-1", "message-1", "draft-1", "opened");
    expect(mocks.minimize).toHaveBeenCalledOnce();
    expect(mocks.updatePolicy).not.toHaveBeenCalled();
  });

  test("applies a source-bound proposal without navigating and only minimizes after it finishes", async () => {
    mocks.getSource.mockReturnValue(source);
    let finish!: () => void;
    mocks.apply.mockImplementation(() => new Promise<void>((resolve) => { finish = resolve; }));
    const { button } = renderButton();
    expect(button.textContent).toBe("Apply to draft");
    await act(async () => button.click());
    expect(mocks.apply).toHaveBeenCalledWith(draft, source, context, { draftId: "draft-1", messageId: "message-1" });
    expect(mocks.push).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    await act(async () => finish());
    expect(mocks.minimize).toHaveBeenCalledOnce();
    expect(mocks.updatePolicy).not.toHaveBeenCalled();
  });

  test.each(["opened", "applied"] as const)("retains its %s receipt after the consumed action remounts", async (outcome) => {
    mocks.getSource.mockReturnValue(outcome === "applied" ? source : undefined);
    mocks.consume.mockImplementation(() => { mocks.getDraft.mockReturnValue(undefined); mocks.getSource.mockReturnValue(undefined); });
    const initial = renderButton();
    await act(async () => initial.button.click());
    initial.unmount();
    const { button, container } = renderButton();
    expect(button.textContent).toBe(outcome === "opened" ? "Opened exemption draft" : "Applied to draft");
    expect(button.disabled).toBe(true);
    expect(button.querySelector("svg")).toHaveClass("lucide-check");
    expect(container.querySelector('[role="alert"]')).toBeNull();
    expect(container.textContent).not.toContain("Only fills the unsaved form.");
    await act(async () => button.click());
    expect(mocks.revalidate).toHaveBeenCalledOnce();
  });

  test.each(["en-US", "es-ES", "ja-JP", "vi-VN", "zh-CN"])("shows the persisted created status without an unsaved reminder in %s", locale => {
    mocks.getDraft.mockReturnValue(undefined);
    mocks.handoff = { draftId: "draft-1", outcome: "opened", created: true };
    const { button, container } = renderButton(locale);
    expect(button.disabled).toBe(true);
    expect(button.querySelector("svg")).toHaveClass("lucide-check");
    expect(container.querySelector("p")).toBeNull();
    if (locale === "en-US") expect(button.textContent).toBe("Created");
    expect(container.textContent).not.toContain("agent.masking-exemption-draft.");
    expect(container.querySelector('[role="alert"]')).toBeNull();
  });

  test("shows an inactive note instead of an error for an unavailable old draft", () => {
    mocks.handoff = { draftId: "another-draft", outcome: "opened" };
    mocks.getDraft.mockReturnValue(undefined);
    const { button, container } = renderButton();
    expect(button.disabled).toBe(true);
    expect(container.textContent).toContain("This draft is no longer available. Check the current exemptions before preparing another draft.");
    expect(container.querySelector('[role="alert"]')).toBeNull();
  });

  test.each(["user", "workspace", "chat", "chat-owner"])("rejects a changed %s while validation is pending", async (changed) => {
    let finish!: () => void;
    mocks.revalidate.mockImplementation(() => new Promise<typeof draft>((resolve) => { finish = () => resolve(draft); }));
    const { button, container } = renderButton();
    act(() => button.click());
    if (changed === "user") mocks.userName = "users/other@example.com";
    if (changed === "workspace") mocks.workspaceName = "workspaces/other";
    if (changed === "chat") mocks.currentChatId = "chat-2";
    if (changed === "chat-owner") mocks.chat = { ...mocks.chat, workspaceName: "workspaces/other" };
    await act(async () => finish());
    expect(mocks.push).not.toHaveBeenCalled();
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
  });

  test.each(["resolution", "navigation", "ready"])("rejects an incorrect route at %s", async (stage) => {
    if (stage === "resolution") mocks.resolve.mockReturnValue({ fullPath: "/projects/sample/masking-exemption" });
    if (stage === "navigation") mocks.push.mockImplementation(async () => { mocks.path = "/projects/other/masking-exemption/create"; });
    if (stage === "ready") mocks.ready.mockImplementation(() => { mocks.path = "/other"; return true; });
    const { button } = renderButton();
    await act(async () => button.click());
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(button.disabled).toBe(false);
  });

  test("waits for the exact target and times out without consuming", async () => {
    vi.useFakeTimers();
    mocks.ready.mockReturnValue(false);
    const { button, container } = renderButton();
    await act(async () => button.click());
    expect(mocks.open).not.toHaveBeenCalled();
    await act(async () => vi.advanceTimersByTimeAsync(3_000));
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(button.disabled).toBe(false);
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
  });

  test.each(["open", "apply"] as const)("keeps a rejected %s retryable without leaking errors or minimizing", async (action) => {
    mocks.getSource.mockReturnValue(action === "apply" ? source : undefined);
    mocks[action].mockRejectedValueOnce(new Error("Internal sensitive details"));
    const { button, container } = renderButton();
    await act(async () => button.click());
    expect(button.disabled).toBe(false);
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
    expect(container.textContent).not.toContain("Internal sensitive details");
    expect(container.querySelector('[role="alert"]')?.textContent).toContain("Couldn't apply this draft.");
    await act(async () => button.click());
    expect(mocks.consume).toHaveBeenCalledOnce();
    expect(mocks.minimize).toHaveBeenCalledOnce();
  });

  test("allows only one concurrent handoff", async () => {
    let finish!: () => void;
    mocks.revalidate.mockImplementation(() => new Promise<typeof draft>((resolve) => { finish = () => resolve(draft); }));
    const { button } = renderButton();
    act(() => { button.click(); button.click(); });
    await act(async () => finish());
    expect(mocks.revalidate).toHaveBeenCalledOnce();
    expect(mocks.open).toHaveBeenCalledOnce();
    expect(mocks.consume).toHaveBeenCalledOnce();
  });
});
