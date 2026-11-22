import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { MaskingRuleDraftButton } from "./MaskingRuleDraftButton";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  userName: "users/member@example.com",
  workspaceName: "workspaces/sample",
  currentChatId: "chat-1",
  chat: { id: "chat-1", workspaceName: "workspaces/sample" },
  currentPath: "/initial",
  visible: true,
  getDraft: vi.fn(),
  getDrafts: vi.fn(),
  getSource: vi.fn(),
  revalidate: vi.fn(),
  revalidateMany: vi.fn(),
  consume: vi.fn(),
  handoff: undefined as
    | { draftId: string; outcome: "opened" | "applied" }
    | undefined,
  recordHandoff: vi.fn(),
  ready: vi.fn(),
  open: vi.fn(),
  openMany: vi.fn(),
  apply: vi.fn(),
  push: vi.fn(),
  resolveHandoff: vi.fn(),
  minimize: vi.fn(),
  updatePolicy: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/stores/app", () => {
  const state = {
    get currentUser() {
      return { name: mocks.userName };
    },
    workspaceResourceName: () => mocks.workspaceName,
  };
  const useAppStore = <T,>(selector: (value: typeof state) => T) =>
    selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

vi.mock("../store/agent", () => {
  const state = {
    get currentChatId() {
      return mocks.currentChatId;
    },
    get visible() { return mocks.visible; },
    get workspaceName() { return mocks.workspaceName; },
    get userEmail() { return mocks.userName; },
    getChat: (id: string) => (id === mocks.chat.id ? mocks.chat : null),
    getMessages: () => [
      {
        id: "message-1",
        metadata: { maskingRuleDraftHandoff: mocks.handoff },
      },
    ],
    recordMaskingRuleDraftHandoff: mocks.recordHandoff,
    minimize: mocks.minimize,
  };
  const useAgentStore = <T,>(selector: (value: typeof state) => T) =>
    selector(state);
  useAgentStore.getState = () => state;
  return { useAgentStore };
});

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.push,
    resolve: () => ({ fullPath: "/settings/global-masking" }),
    get currentRoute() {
      return { value: { fullPath: mocks.currentPath } };
    },
  },
}));

vi.mock("@/api", () => ({
  orgPolicyServiceClientConnect: { updatePolicy: mocks.updatePolicy },
}));

vi.mock("../logic/handoff", () => ({
  resolveAgentHandoffRoute: mocks.resolveHandoff,
}));

vi.mock("../logic/maskingRuleDraft", () => ({
  getMaskingRuleDraft: mocks.getDraft,
  getMaskingRuleDrafts: mocks.getDrafts,
  getMaskingRuleDraftSource: mocks.getSource,
  revalidateMaskingRuleDraft: mocks.revalidate,
  revalidateMaskingRuleDrafts: mocks.revalidateMany,
  consumeMaskingRuleDraft: mocks.consume,
}));

vi.mock("../logic/maskingRuleDraftTarget", () => ({
  isMaskingRuleDraftTargetReady: mocks.ready,
  openMaskingRuleDraftInEditor: mocks.open,
  openMaskingRuleDraftsInEditor: mocks.openMany,
  applyMaskingRuleDraftInEditor: mocks.apply,
}));

const context = {
  workspaceName: "workspaces/sample",
  userName: "users/member@example.com",
  chatId: "chat-1",
};
const draft = {
  title: "Mask sensitive columns",
  condition: 'resource.column_name == "sensitive"',
  semanticType: "semanticTypes/default",
  scope: { kind: "workspace" as const },
};
const source = {
  sessionId: "editor-1",
  revision: "revision-1",
  title: "Existing draft",
  condition: "true",
  semanticType: "semanticTypes/default",
  chatId: "chat-1",
};
const roots: Root[] = [];

function renderButton() {
  const container = document.createElement("div");
  const root = createRoot(container);
  roots.push(root);
  act(() => {
    root.render(
      <MaskingRuleDraftButton
        draftId="draft-1"
        chatId="chat-1"
        messageId="message-1"
      />
    );
  });
  const button = container.querySelector("button");
  if (!button) throw new Error("Missing draft button");
  return {
    container,
    button,
    unmount: () => {
      act(() => root.unmount());
      roots.splice(roots.indexOf(root), 1);
    },
  };
}

beforeEach(() => {
  vi.resetAllMocks();
  mocks.userName = context.userName;
  mocks.workspaceName = context.workspaceName;
  mocks.currentChatId = context.chatId;
  mocks.chat = { id: context.chatId, workspaceName: context.workspaceName };
  mocks.currentPath = "/initial";
  mocks.visible = true;
  mocks.handoff = undefined;
  mocks.recordHandoff.mockImplementation(
    (_chatId, _messageId, draftId, outcome) => {
      mocks.handoff = { draftId, outcome };
    }
  );
  mocks.getDraft.mockReturnValue(draft);
  mocks.getDrafts.mockImplementation(() => {
    const value = mocks.getDraft();
    return value ? [value] : undefined;
  });
  mocks.getSource.mockReturnValue(undefined);
  mocks.revalidate.mockResolvedValue(draft);
  mocks.ready.mockReturnValue(true);
  mocks.open.mockResolvedValue(undefined);
  mocks.apply.mockResolvedValue(undefined);
  mocks.resolveHandoff.mockReturnValue({ name: "global-masking" });
  mocks.push.mockImplementation(async () => {
    mocks.currentPath = "/settings/global-masking";
  });
});

afterEach(() => {
  act(() => {
    for (const root of roots.splice(0)) root.unmount();
  });
  vi.useRealTimers();
});

describe("MaskingRuleDraftButton", () => {
  test("reviews every batch summary and opens all drafts in one member handoff", async () => {
    const second = { ...draft, title: "Mask email", condition: 'resource.column_name == "email"', semanticType: "semanticTypes/email" };
    const drafts = [draft, second];
    mocks.getDraft.mockReturnValue(undefined);
    mocks.getDrafts.mockReturnValue(drafts);
    mocks.revalidateMany.mockResolvedValue(drafts);
    const { button, container } = renderButton();

    expect(button.textContent).toContain("agent.masking-rule-draft.review-many");
    expect(container.textContent).toContain(draft.title);
    expect(container.textContent).toContain(second.title);
    expect(container.querySelectorAll("code")).toHaveLength(2);
    expect(mocks.openMany).not.toHaveBeenCalled();
    await act(async () => button.click());

    expect(mocks.revalidateMany).toHaveBeenCalledWith("draft-1", context);
    expect(mocks.openMany).toHaveBeenCalledWith(drafts, context);
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.apply).not.toHaveBeenCalled();
    expect(mocks.consume).toHaveBeenCalledOnce();
    expect(mocks.minimize).toHaveBeenCalledOnce();
    expect(mocks.updatePolicy).not.toHaveBeenCalled();
  });

  test("keeps every batch draft retryable when its editor rejects the handoff", async () => {
    const drafts = [draft, { ...draft, title: "Second" }];
    mocks.getDrafts.mockReturnValue(drafts);
    mocks.revalidateMany.mockResolvedValue(drafts);
    mocks.openMany.mockRejectedValue(new Error("Editor changed"));
    const { button, container } = renderButton();
    await act(async () => button.click());
    expect(button.disabled).toBe(false);
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
  });

  test.each(["opened", "applied"] as const)(
    "keeps the %s completion after chat is reopened without replaying the consumed draft",
    async (outcome) => {
      mocks.getSource.mockReturnValue(outcome === "applied" ? source : undefined);
      mocks.consume.mockImplementation(() => {
        mocks.getDraft.mockReturnValue(undefined);
        mocks.getSource.mockReturnValue(undefined);
      });
      const initial = renderButton();
      await act(async () => initial.button.click());
      initial.unmount();

      const reopened = renderButton();

      expect(reopened.button.textContent).toContain(
        `agent.masking-rule-draft.${outcome}`
      );
      expect(reopened.button.querySelector("svg")).toHaveClass("lucide-check");
      expect(reopened.container.querySelector('[role="alert"]')).toBeNull();
      expect(reopened.button.disabled).toBe(true);
      await act(async () => reopened.button.click());
      expect(mocks.revalidate).toHaveBeenCalledOnce();
      expect(mocks.consume).toHaveBeenCalledOnce();
      expect(mocks.recordHandoff).toHaveBeenCalledWith(
        "chat-1",
        "message-1",
        "draft-1",
        outcome
      );
      expect(mocks.updatePolicy).not.toHaveBeenCalled();
    }
  );

  test("does not use another draft's completion for an unavailable draft", () => {
    mocks.handoff = { draftId: "other-draft", outcome: "opened" };
    mocks.getDraft.mockReturnValue(undefined);
    const { button, container } = renderButton();

    expect(button.textContent).toBe("agent.masking-rule-draft.open");
    expect(container.querySelector('[role="alert"]')).toBeNull();
    expect(container.textContent).toContain("agent.masking-rule-draft.inactive");
    expect(button.disabled).toBe(true);
  });

  test("does not claim an expired or previously consumed draft failed to apply", () => {
    mocks.getDraft.mockReturnValue(undefined);
    const { button, container } = renderButton();

    expect(container.querySelector('[role="alert"]')).toBeNull();
    expect(container.textContent).toContain("agent.masking-rule-draft.inactive");
    expect(button.disabled).toBe(true);
    expect(mocks.revalidate).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
  });

  test("minimizes only after a member opens the filled local draft without saving", async () => {
    const { button, container } = renderButton();
    expect(button.textContent).toContain("agent.masking-rule-draft.open");
    expect(button).toHaveClass("text-accent", "bg-transparent", "h-7", "gap-1.5");
    expect(button.querySelector("svg")).toHaveClass("size-4", "lucide-arrow-up-right");
    expect(button).not.toHaveClass("border");
    expect(container.textContent).toContain("agent.masking-rule-draft.unsaved");
    expect(mocks.revalidate).not.toHaveBeenCalled();
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();

    await act(async () => button.click());

    expect(mocks.revalidate).toHaveBeenCalledWith("draft-1", context);
    expect(mocks.resolveHandoff).toHaveBeenCalledWith({
      kind: "open-global-masking",
    });
    expect(mocks.push).toHaveBeenCalledWith({ name: "global-masking" });
    expect(mocks.open).toHaveBeenCalledWith(draft, context);
    expect(mocks.apply).not.toHaveBeenCalled();
    expect(mocks.consume).toHaveBeenCalledWith("draft-1", context);
    expect(button.disabled).toBe(true);
    expect(button.textContent).toContain("agent.masking-rule-draft.opened");
    expect(button.querySelector("svg")).toHaveClass("size-4", "lucide-check");
    expect(mocks.minimize).toHaveBeenCalledOnce();
    expect(mocks.updatePolicy).not.toHaveBeenCalled();
  });

  test("shows the actual validated scope, semantic type, and condition", () => {
    mocks.getDraft.mockReturnValue({
      ...draft,
      scope: { kind: "project", project: "projects/reviewed" },
    });
    const { container } = renderButton();

    expect(container.textContent).toContain("agent.masking-rule-draft.scope");
    expect(container.textContent).toContain("projects/reviewed");
    expect(container.textContent).toContain(
      "agent.masking-rule-draft.semantic-type"
    );
    expect(container.textContent).toContain(draft.semanticType);
    expect(container.querySelector("code")?.textContent).toBe(draft.condition);
  });

  test("applies a source-bound draft without opening another editor or navigating", async () => {
    mocks.getSource.mockReturnValue(source);
    const { button } = renderButton();
    expect(button.textContent).toContain("agent.masking-rule-draft.apply");
    expect(button.querySelector("svg")).toHaveClass("size-4", "lucide-file-pen-line");

    await act(async () => button.click());

    expect(mocks.apply).toHaveBeenCalledWith(draft, source, context);
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.push).not.toHaveBeenCalled();
    expect(mocks.consume).toHaveBeenCalledWith("draft-1", context);
    expect(button.disabled).toBe(true);
    expect(button.textContent).toContain("agent.masking-rule-draft.applied");
    expect(mocks.minimize).toHaveBeenCalledOnce();
  });

  test.each(["user", "workspace", "chat", "chat-owner"] as const)(
    "does not navigate when the %s changes during validation",
    async (changed) => {
      let finishValidation: (() => void) | undefined;
      mocks.revalidate.mockImplementation(
        () =>
          new Promise<typeof draft>((resolve) => {
            finishValidation = () => resolve(draft);
          })
      );
      const { button, container } = renderButton();
      act(() => button.click());

      if (changed === "user") mocks.userName = "users/another@example.com";
      if (changed === "workspace") mocks.workspaceName = "workspaces/another";
      if (changed === "chat") mocks.currentChatId = "chat-2";
      if (changed === "chat-owner")
        mocks.chat = { ...mocks.chat, workspaceName: "workspaces/another" };
      await act(async () => finishValidation?.());

      expect(mocks.push).not.toHaveBeenCalled();
      expect(mocks.open).not.toHaveBeenCalled();
      expect(mocks.apply).not.toHaveBeenCalled();
      expect(mocks.consume).not.toHaveBeenCalled();
      expect(mocks.minimize).not.toHaveBeenCalled();
      expect(container.querySelector('[role="alert"]')?.textContent).toContain(
        "agent.masking-rule-draft.unavailable"
      );
    }
  );

  test("keeps a failed draft available for another member click", async () => {
    mocks.open.mockRejectedValueOnce(new Error("Internal sensitive context"));
    const { button, container } = renderButton();

    await act(async () => button.click());

    expect(button.disabled).toBe(false);
    expect(container.querySelector('[role="alert"]')?.textContent).toContain(
      "agent.masking-rule-draft.unavailable"
    );
    expect(container.textContent).not.toContain("Internal sensitive context");
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();

    await act(async () => button.click());

    expect(mocks.open).toHaveBeenCalledTimes(2);
    expect(mocks.consume).toHaveBeenCalledOnce();
    expect(container.querySelector('[role="alert"]')).toBeNull();
    expect(button.disabled).toBe(true);
    expect(mocks.minimize).toHaveBeenCalledOnce();
  });

  test("waits for the editor to mount before opening", async () => {
    vi.useFakeTimers();
    mocks.ready.mockReturnValue(false);
    const { button } = renderButton();

    await act(async () => button.click());
    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.minimize).not.toHaveBeenCalled();
    mocks.ready.mockReturnValue(true);
    await act(async () => vi.advanceTimersByTimeAsync(50));

    expect(mocks.open).toHaveBeenCalledWith(draft, context);
    expect(mocks.consume).toHaveBeenCalledOnce();
    expect(mocks.minimize).toHaveBeenCalledOnce();
  });

  test("does not open when navigation leaves the requested target", async () => {
    mocks.push.mockImplementation(async () => {
      mocks.currentPath = "/another-page";
    });
    const { button, container } = renderButton();

    await act(async () => button.click());

    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(button.disabled).toBe(false);
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
    expect(mocks.minimize).not.toHaveBeenCalled();
  });

  test("rechecks the route immediately before opening the ready editor", async () => {
    mocks.ready.mockImplementation(() => {
      mocks.currentPath = "/another-page";
      return true;
    });
    const { button } = renderButton();

    await act(async () => button.click());

    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(button.disabled).toBe(false);
    expect(mocks.minimize).not.toHaveBeenCalled();
  });

  test("stops waiting after three seconds and leaves the draft retryable", async () => {
    vi.useFakeTimers();
    mocks.ready.mockReturnValue(false);
    const { button, container } = renderButton();

    await act(async () => button.click());
    await act(async () => vi.advanceTimersByTimeAsync(3_000));

    expect(mocks.open).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(button.disabled).toBe(false);
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
    expect(mocks.minimize).not.toHaveBeenCalled();
  });

  test("waits for the form to finish filling before minimizing", async () => {
    let finishOpen!: () => void;
    mocks.open.mockImplementation(() => new Promise<void>((resolve) => { finishOpen = resolve; }));
    const { button } = renderButton();
    await act(async () => button.click());
    expect(mocks.open).toHaveBeenCalledOnce();
    expect(mocks.minimize).not.toHaveBeenCalled();
    await act(async () => finishOpen());
    expect(mocks.minimize).toHaveBeenCalledOnce();
  });

  test("keeps the chat open when applying to the current draft fails", async () => {
    mocks.getSource.mockReturnValue(source);
    mocks.apply.mockRejectedValueOnce(new Error("The draft changed"));
    const { button, container } = renderButton();
    await act(async () => button.click());
    expect(mocks.minimize).not.toHaveBeenCalled();
    expect(mocks.consume).not.toHaveBeenCalled();
    expect(button.disabled).toBe(false);
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
  });
});
