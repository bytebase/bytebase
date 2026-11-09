import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  assertCurrentContext,
  assertCurrentTarget,
  waitForTarget,
} from "./draftHandoff";

const mocks = vi.hoisted(() => ({
  userName: "users/member@example.com" as string | undefined,
  workspaceName: "workspaces/sample",
  currentChatId: "chat-1",
  chat: { id: "chat-1", workspaceName: "workspaces/sample" } as
    | { id: string; workspaceName: string }
    | undefined,
  path: "/settings/global-masking?view=rules#editor",
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      currentUser: mocks.userName ? { name: mocks.userName } : undefined,
      workspaceResourceName: () => mocks.workspaceName,
    }),
  },
}));

vi.mock("../store/agent", () => ({
  useAgentStore: {
    getState: () => ({
      currentChatId: mocks.currentChatId,
      getChat: (id: string) => (id === mocks.chat?.id ? mocks.chat : undefined),
    }),
  },
}));

vi.mock("@/app/router", () => ({
  router: {
    get currentRoute() {
      return { value: { fullPath: mocks.path } };
    },
  },
}));

const context = {
  userName: "users/member@example.com",
  workspaceName: "workspaces/sample",
  chatId: "chat-1",
};
const targetPath = "/settings/global-masking?view=rules#editor";
const contextChanges = {
  user: () => {
    mocks.userName = "users/another@example.com";
  },
  signedOut: () => {
    mocks.userName = undefined;
  },
  workspace: () => {
    mocks.workspaceName = "workspaces/another";
  },
  chat: () => {
    mocks.currentChatId = "chat-2";
  },
  chatOwner: () => {
    mocks.chat = { id: context.chatId, workspaceName: "workspaces/another" };
  },
  missingChat: () => {
    mocks.chat = undefined;
  },
};

beforeEach(() => {
  vi.useFakeTimers();
  mocks.userName = context.userName;
  mocks.workspaceName = context.workspaceName;
  mocks.currentChatId = context.chatId;
  mocks.chat = { id: context.chatId, workspaceName: context.workspaceName };
  mocks.path = targetPath;
});

afterEach(() => {
  vi.useRealTimers();
});

describe("draft handoff guards", () => {
  test("accepts the current actor, workspace, selected chat, and exact route", () => {
    expect(() => assertCurrentContext(context)).not.toThrow();
    expect(() => assertCurrentTarget(targetPath)).not.toThrow();
  });

  test.each(Object.entries(contextChanges))(
    "rejects a changed %s",
    (_change, changeContext) => {
      changeContext();
      expect(() => assertCurrentContext(context)).toThrow();
    }
  );

  test.each([
    "/settings/semantic-types?view=rules#editor",
    "/settings/global-masking?view=other#editor",
    "/settings/global-masking?view=rules#other",
  ])("rejects a different full route %s", (path) => {
    mocks.path = path;
    expect(() => assertCurrentTarget(targetPath)).toThrow();
  });
});

describe("draft handoff target wait", () => {
  test("accepts an already-ready target without waiting", async () => {
    const isReady = vi.fn(() => true);

    await expect(waitForTarget(context, targetPath, isReady)).resolves.toBe(
      undefined
    );

    expect(isReady).toHaveBeenCalledExactlyOnceWith(context);
    expect(vi.getTimerCount()).toBe(0);
  });

  test("checks readiness again after each 50ms interval", async () => {
    const isReady = vi.fn(() => false);
    const waiting = waitForTarget(context, targetPath, isReady);

    expect(isReady).toHaveBeenCalledExactlyOnceWith(context);
    isReady.mockReturnValue(true);
    await vi.advanceTimersByTimeAsync(49);
    expect(isReady).toHaveBeenCalledOnce();
    await vi.advanceTimersByTimeAsync(1);

    await expect(waiting).resolves.toBeUndefined();
    expect(isReady).toHaveBeenCalledTimes(2);
    expect(isReady).toHaveBeenLastCalledWith(context);
    expect(vi.getTimerCount()).toBe(0);
  });

  test.each(Object.entries(contextChanges))(
    "checks a changed %s before reading readiness",
    async (_change, changeContext) => {
      changeContext();
      const isReady = vi.fn(() => true);

      await expect(
        waitForTarget(context, targetPath, isReady)
      ).rejects.toThrow();

      expect(isReady).not.toHaveBeenCalled();
      expect(vi.getTimerCount()).toBe(0);
    }
  );

  test("checks the target route before reading readiness", async () => {
    mocks.path = "/another";
    const isReady = vi.fn(() => true);

    await expect(waitForTarget(context, targetPath, isReady)).rejects.toThrow();

    expect(isReady).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
  });

  test.each([
    ...Object.entries(contextChanges),
    ["route", () => (mocks.path = "/another")] as const,
  ])(
    "rejects a changed %s while the target is pending",
    async (_change, change) => {
      const isReady = vi.fn(() => false);
      const waiting = waitForTarget(context, targetPath, isReady);
      const rejected = expect(waiting).rejects.toThrow();

      change();
      isReady.mockReturnValue(true);
      await vi.advanceTimersByTimeAsync(50);

      await rejected;
      expect(isReady).toHaveBeenCalledOnce();
      expect(vi.getTimerCount()).toBe(0);
    }
  );

  test("times out after 60 readiness checks and exactly three seconds", async () => {
    const isReady = vi.fn(() => false);
    const waiting = waitForTarget(context, targetPath, isReady);
    const rejected = expect(waiting).rejects.toThrow();

    await vi.advanceTimersByTimeAsync(2_999);
    expect(isReady).toHaveBeenCalledTimes(60);
    expect(vi.getTimerCount()).toBe(1);
    await vi.advanceTimersByTimeAsync(1);

    await rejected;
    expect(isReady).toHaveBeenCalledTimes(60);
    expect(vi.getTimerCount()).toBe(0);
  });
});
