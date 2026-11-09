import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { I18nextProvider } from "react-i18next";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import i18n from "@/lib/i18n";
import { SemanticTypeDraftButton } from "./SemanticTypeDraftButton";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  userName: "users/member@example.com",
  workspaceName: "workspaces/sample",
  currentChatId: "chat-1",
  chat: { id: "chat-1", workspaceName: "workspaces/sample" },
  path: "/initial",
  resolve: vi.fn(), push: vi.fn(), resolveHandoff: vi.fn(),
  getDraft: vi.fn(), getSource: vi.fn(), getSummary: vi.fn(),
  revalidate: vi.fn(), consume: vi.fn(), ready: vi.fn(), open: vi.fn(),
  minimize: vi.fn(), recordHandoff: vi.fn(),
  handoff: undefined as { draftId: string; outcome: "opened"; created?: true } | undefined,
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
    getMessages: () => [{ id: "message-1", metadata: { semanticTypeDraftHandoff: mocks.handoff } }],
    recordSemanticTypeDraftHandoff: mocks.recordHandoff,
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
vi.mock("../logic/handoff", () => ({ resolveAgentHandoffRoute: mocks.resolveHandoff }));
vi.mock("../logic/semanticTypeDraft", () => ({
  getSemanticTypeDraft: mocks.getDraft,
  getSemanticTypeDraftSource: mocks.getSource,
  getSafeSemanticTypeDraftSummary: mocks.getSummary,
  revalidateSemanticTypeDraft: mocks.revalidate,
  consumeSemanticTypeDraft: mocks.consume,
}));
vi.mock("../logic/semanticTypeDraftTarget", () => ({
  isSemanticTypeDraftTargetReady: mocks.ready,
  openSemanticTypeDraftInEditor: mocks.open,
}));

const context = { workspaceName: "workspaces/sample", userName: "users/member@example.com", chatId: "chat-1" };
const target = { name: "setting.semantic-types" };
const targetPath = "/semantic-types";
const draft = {
  title: "Email protection",
  description: "Synthetic masking configuration",
  algorithm: { kind: "md5", salt: "private-test-salt" },
};
const source = { sessionId: "editor-1", revision: "revision-1", busy: false };
const roots: Root[] = [];
function renderButton(locale = "en-US") {
  const container = document.createElement("div");
  const root = createRoot(container);
  roots.push(root);
  act(() => root.render(
    <I18nextProvider i18n={i18n.cloneInstance({ lng: locale, initAsync: false })}>
      <SemanticTypeDraftButton draftId="draft-1" chatId="chat-1" messageId="message-1" />
    </I18nextProvider>
  ));
  const button = container.querySelector("button");
  if (!button) throw new Error("Missing semantic type draft action");
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
  mocks.getSummary.mockReturnValue({ title: draft.title, description: draft.description, algorithm: { kind: "md5-mask" } });
  mocks.revalidate.mockResolvedValue(draft);
  mocks.ready.mockReturnValue(true);
  mocks.resolveHandoff.mockReturnValue(target);
  mocks.resolve.mockReturnValue({ fullPath: targetPath });
  mocks.push.mockImplementation(async () => { mocks.path = targetPath; });
  mocks.recordHandoff.mockImplementation((_chatId, _messageId, draftId) => { mocks.handoff = { draftId, outcome: "opened" }; });
});
afterEach(() => {
  act(() => { for (const root of roots.splice(0)) root.unmount(); });
  vi.useRealTimers();
});

describe("SemanticTypeDraftButton", () => {
  test.each(["en-US", "es-ES", "ja-JP", "vi-VN", "zh-CN"])("localizes the open, opened, and created receipts in %s", async locale => {
    const initial = renderButton(locale);
    expect(initial.container.textContent).not.toContain("agent.semantic-type-draft.");
    await act(async () => initial.button.click());
    expect(initial.container.textContent).not.toContain("agent.semantic-type-draft.");
    initial.unmount();
    mocks.getDraft.mockReturnValue(undefined);
    mocks.handoff = { draftId: "draft-1", outcome: "opened", created: true };
    const { container, button } = renderButton(locale);
    expect(container.textContent).not.toContain("agent.semantic-type-draft.");
    expect(button.disabled).toBe(true);
    expect(container.querySelector('[role="alert"]')).toBeNull();
  });

  test.each([
    [{ kind: "full-mask", substitution: "[hidden]" }, "Full mask"],
    [{ kind: "range-mask", slices: [] }, "Range mask"],
    [{ kind: "md5-mask" }, "MD5 mask"],
    [{ kind: "inner-outer-mask", type: "inner" }, "Inner Mask"],
    [{ kind: "inner-outer-mask", type: "outer" }, "Outer Mask"],
  ])("reuses the localized semantic algorithm label for %o", (algorithm, label) => {
    mocks.getSummary.mockReturnValue({ title: draft.title, description: draft.description, algorithm });
    const { container } = renderButton();
    expect(Array.from(container.querySelectorAll("dt"), element => element.textContent)).toEqual([
      "Title", "Description", "Masking Algorithm",
    ]);
    expect(container.textContent).toContain(label);
    expect(container.textContent).not.toContain("settings.sensitive-data.");
  });

  test("opens only after a member click and summarizes without a salt", async () => {
    const { button, container } = renderButton();
    expect(button).toHaveClass("text-accent", "bg-transparent", "h-7");
    expect(button.querySelector("svg")).toHaveClass("lucide-arrow-up-right");
    expect(container.textContent).toContain(draft.title);
    expect(container.textContent).toContain(draft.description);
    expect(container.textContent).toContain("MD5");
    expect(container.textContent).not.toContain("private-test-salt");
    expect(mocks.revalidate).not.toHaveBeenCalled();
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    await act(async () => button.click());
    expect(mocks.resolveHandoff).toHaveBeenCalledWith({ kind: "open-semantic-types" });
    expect(mocks.push).toHaveBeenCalledWith(target);
    expect(mocks.open).toHaveBeenCalledWith(draft, context, { draftId: "draft-1", messageId: "message-1" }, undefined);
    expect(mocks.recordHandoff).toHaveBeenCalledWith("chat-1", "message-1", "draft-1", "opened");
    expect(mocks.consume).toHaveBeenCalledWith("draft-1", context);
    expect(mocks.minimize).toHaveBeenCalledOnce();
  });

  test("passes a source-bound snapshot without navigation or replacement", async () => {
    mocks.path = targetPath;
    mocks.getSource.mockReturnValue(source);
    const { button } = renderButton();
    await act(async () => button.click());
    expect(mocks.push).not.toHaveBeenCalled();
    expect(mocks.open).toHaveBeenCalledWith(draft, context, { draftId: "draft-1", messageId: "message-1" }, source);
    expect(mocks.minimize).toHaveBeenCalledOnce();
  });

  test("does not navigate away from a same-page active editor to evade busy validation", async () => {
    mocks.path = targetPath;
    mocks.open.mockRejectedValue(new Error("The editor is busy."));
    const { button, container } = renderButton();
    await act(async () => button.click());
    expect(mocks.push).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
  });

  test("rejects a source-bound action after leaving its editor page", async () => {
    mocks.getSource.mockReturnValue(source);
    const { button } = renderButton();
    await act(async () => button.click());
    expect(mocks.push).not.toHaveBeenCalled();
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
  });

  test.each(["permission", "source"])("keeps a rejected %s validation retryable without navigation", async rejected => {
    if (rejected === "source") {
      mocks.path = targetPath;
      mocks.getSource.mockReturnValue(source);
    }
    mocks.revalidate.mockRejectedValueOnce(new Error("Internal sensitive details"));
    const { button, container } = renderButton();
    await act(async () => button.click());
    expect(mocks.push).not.toHaveBeenCalled();
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(button.disabled).toBe(false);
    expect(container.textContent).not.toContain("Internal sensitive details");
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
    await act(async () => button.click());
    expect(mocks.consume).toHaveBeenCalledOnce();
    expect(mocks.minimize).toHaveBeenCalledOnce();
  });

  test("does not consume or minimize when the editor rejects a draft", async () => {
    mocks.open.mockRejectedValueOnce(new Error("Private implementation details"));
    const { button, container } = renderButton();
    await act(async () => button.click());
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(container.textContent).not.toContain("Private implementation details");
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
    expect(button.disabled).toBe(false);
    await act(async () => button.click());
    expect(mocks.open).toHaveBeenCalledTimes(2);
    expect(mocks.consume).toHaveBeenCalledOnce();
    expect(mocks.minimize).toHaveBeenCalledOnce();
  });

  test("rejects an unavailable typed destination", async () => {
    mocks.resolveHandoff.mockReturnValue(undefined);
    const { button, container } = renderButton();
    await act(async () => button.click());
    expect(mocks.push).not.toHaveBeenCalled();
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
  });

  test.each(["user", "workspace", "chat", "chat-owner"])("rejects a changed %s while validation is pending", async changed => {
    let finish!: () => void;
    mocks.revalidate.mockImplementation(() => new Promise<typeof draft>(resolve => { finish = () => resolve(draft); }));
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

  test.each(["navigation", "ready"])("rejects a changed route at %s", async stage => {
    if (stage === "navigation") mocks.push.mockImplementation(async () => { mocks.path = "/other"; });
    if (stage === "ready") mocks.ready.mockImplementation(() => { mocks.path = "/other"; return true; });
    const { button, container } = renderButton();
    await act(async () => button.click());
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
  });

  test("waits for the target and times out without consuming", async () => {
    vi.useFakeTimers();
    mocks.ready.mockReturnValue(false);
    const { button, container } = renderButton();
    await act(async () => button.click());
    expect(mocks.open).not.toHaveBeenCalled();
    await act(async () => vi.advanceTimersByTimeAsync(3_000));
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
  });

  test("only consumes and minimizes after the editor accepts the draft", async () => {
    let finish!: () => void;
    mocks.open.mockImplementation(() => new Promise<void>(resolve => { finish = resolve; }));
    const { button } = renderButton();
    await act(async () => button.click());
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    await act(async () => finish());
    expect(mocks.consume).toHaveBeenCalledOnce();
    expect(mocks.minimize).toHaveBeenCalledOnce();
  });

  test("does not minimize after leaving the page while the target accepts the draft", async () => {
    let finish!: () => void;
    mocks.open.mockImplementation(() => new Promise<void>(resolve => { finish = resolve; }));
    const { button, container } = renderButton();
    await act(async () => button.click());
    mocks.path = "/other";
    await act(async () => finish());
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
  });

  test("retains its opened receipt and unsaved note after reopening chat", async () => {
    mocks.consume.mockImplementation(() => { mocks.getDraft.mockReturnValue(undefined); });
    const initial = renderButton();
    await act(async () => initial.button.click());
    initial.unmount();
    const { button, container } = renderButton();
    expect(button.disabled).toBe(true);
    expect(button.querySelector("svg")).toHaveClass("lucide-check");
    expect(container.querySelector('[role="alert"]')).toBeNull();
    expect(container.querySelector("p")).not.toBeNull();
    await act(async () => button.click());
    expect(mocks.revalidate).toHaveBeenCalledOnce();
  });

  test("retains the created receipt without an unsaved reminder", () => {
    mocks.getDraft.mockReturnValue(undefined);
    mocks.handoff = { draftId: "draft-1", outcome: "opened", created: true };
    const { button, container } = renderButton();
    expect(button.disabled).toBe(true);
    expect(button.querySelector("svg")).toHaveClass("lucide-check");
    expect(container.querySelector("p")?.textContent).toBe(i18n.t("agent.semantic-type-draft.saved"));
    expect(container.textContent).not.toContain(i18n.t("agent.semantic-type-draft.unsaved"));
    expect(container.querySelector('[role="alert"]')).toBeNull();
  });

  test("shows an inactive note for an unavailable old draft", () => {
    mocks.getDraft.mockReturnValue(undefined);
    mocks.handoff = { draftId: "other-draft", outcome: "opened" };
    const { button, container } = renderButton();
    expect(button.disabled).toBe(true);
    expect(container.querySelector("p")?.textContent).toBe(i18n.t("agent.semantic-type-draft.inactive"));
    expect(container.querySelector('[role="alert"]')).toBeNull();
  });

  test("allows only one concurrent handoff", async () => {
    let finish!: () => void;
    mocks.revalidate.mockImplementation(() => new Promise<typeof draft>(resolve => { finish = () => resolve(draft); }));
    const { button } = renderButton();
    act(() => { button.click(); button.click(); });
    await act(async () => finish());
    expect(mocks.revalidate).toHaveBeenCalledOnce();
    expect(mocks.open).toHaveBeenCalledOnce();
    expect(mocks.consume).toHaveBeenCalledOnce();
  });
});
