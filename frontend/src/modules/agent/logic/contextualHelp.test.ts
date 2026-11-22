import { beforeEach, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  open: vi.fn(),
  createChat: vi.fn(),
  queueUserMessage: vi.fn(),
  app: {
    workspace: { name: "workspaces/example" },
    currentUser: {
      name: "users/member@example.com",
      workspace: "workspaces/example",
    },
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => mocks.app },
}));
vi.mock("../store/agent", () => ({
  useAgentStore: { getState: () => mocks },
}));

import {
  canStartAgentPageHelp,
  isAgentPageHelpRequestCurrent,
  openAgentPageHelp,
  subscribeAgentPageHelp,
} from "./contextualHelp";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.app.workspace = { name: "workspaces/example" };
  mocks.app.currentUser = {
    name: "users/member@example.com",
    workspace: "workspaces/example",
  };
});

test("opens contextual help without creating a chat or sending a message", () => {
  const listener = vi.fn();
  const unsubscribe = subscribeAgentPageHelp(listener);
  const onStart = vi.fn(() => true);
  const revoke = openAgentPageHelp({
    page: { kind: "global-masking" },
    maskingRuleEditor: true,
    onStart,
  });
  expect(mocks.open).toHaveBeenCalledOnce();
  expect(mocks.createChat).not.toHaveBeenCalled();
  expect(mocks.queueUserMessage).not.toHaveBeenCalled();
  expect(onStart).not.toHaveBeenCalled();
  expect(listener).toHaveBeenCalledOnce();
  const request = listener.mock.calls[0][0];
  expect(request).toMatchObject({
    page: { kind: "global-masking" },
    maskingRuleEditor: true,
    onStart,
  });
  expect(isAgentPageHelpRequestCurrent(request)).toBe(true);
  revoke();
  expect(isAgentPageHelpRequestCurrent(request)).toBe(false);
  unsubscribe();
});

test("delivers a request once when the Agent window subscribes later", () => {
  const revoke = openAgentPageHelp({ page: { kind: "global-masking" } });
  const listener = vi.fn();
  const unsubscribe = subscribeAgentPageHelp(listener);
  expect(listener).toHaveBeenCalledOnce();
  unsubscribe();
  const resubscribe = subscribeAgentPageHelp(listener);
  expect(listener).toHaveBeenCalledOnce();
  revoke();
  resubscribe();
});

test.each(["user", "workspace"] as const)(
  "rejects a request after its %s changes",
  (change) => {
    const revoke = openAgentPageHelp({ page: { kind: "global-masking" } });
    if (change === "user")
      mocks.app.currentUser.name = "users/another@example.com";
    else {
      mocks.app.workspace.name = "workspaces/another";
      mocks.app.currentUser.workspace = "workspaces/another";
    }
    const listener = vi.fn();
    const unsubscribe = subscribeAgentPageHelp(listener);
    expect(listener).not.toHaveBeenCalled();
    revoke();
    unsubscribe();
  }
);

test("checks source availability again before starting", () => {
  let available = true;
  const listener = vi.fn();
  const unsubscribe = subscribeAgentPageHelp(listener);
  const revoke = openAgentPageHelp({
    page: { kind: "global-masking" },
    isAvailable: () => available,
  });
  const request = listener.mock.calls[0][0];
  expect(canStartAgentPageHelp(request)).toBe(true);
  available = false;
  expect(isAgentPageHelpRequestCurrent(request)).toBe(false);
  expect(canStartAgentPageHelp(request)).toBe(false);
  revoke();
  unsubscribe();
});

test("opens a busy source while disabling its starters", () => {
  let ready = false;
  const listener = vi.fn();
  const unsubscribe = subscribeAgentPageHelp(listener);
  const revoke = openAgentPageHelp({
    page: { kind: "global-masking" },
    isAvailable: () => true,
    canStart: () => ready,
  });
  const request = listener.mock.calls[0][0];
  expect(mocks.open).toHaveBeenCalledOnce();
  expect(isAgentPageHelpRequestCurrent(request)).toBe(true);
  expect(canStartAgentPageHelp(request)).toBe(false);
  ready = true;
  expect(canStartAgentPageHelp(request)).toBe(true);
  revoke();
  unsubscribe();
});

test("an old source cleanup does not revoke the newer request", () => {
  const listener = vi.fn();
  const unsubscribe = subscribeAgentPageHelp(listener);
  const revokeOld = openAgentPageHelp({ page: { kind: "semantic-types" } });
  const oldRequest = listener.mock.calls[0][0];
  const revokeNew = openAgentPageHelp({ page: { kind: "global-masking" } });
  const newRequest = listener.mock.calls[1][0];
  revokeOld();
  expect(isAgentPageHelpRequestCurrent(oldRequest)).toBe(false);
  expect(isAgentPageHelpRequestCurrent(newRequest)).toBe(true);
  expect(listener).toHaveBeenCalledTimes(2);
  revokeNew();
  unsubscribe();
});

test("does not open a mismatched account or unavailable source", () => {
  mocks.app.currentUser.workspace = "workspaces/another";
  openAgentPageHelp({ page: { kind: "global-masking" } });
  mocks.app.currentUser.workspace = "workspaces/example";
  openAgentPageHelp({
    page: { kind: "global-masking" },
    isAvailable: () => false,
  });
  expect(mocks.open).not.toHaveBeenCalled();
});
