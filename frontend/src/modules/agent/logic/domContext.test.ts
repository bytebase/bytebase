import { expect, test, vi } from "vitest";
import type { AppRouterInstance } from "@/app/router";
import { captureAgentDomContext } from "./domContext";

const env = vi.hoisted(() => ({
  actor: "users/1",
  workspace: "workspaces/p",
  email: "me@example.com",
  agentWorkspace: "workspaces/p",
  agentEmail: "me@example.com",
  appChange: () => {},
  agentChange: () => {},
}));
vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      currentUser: { name: env.actor, email: env.email },
      workspaceResourceName: () => env.workspace,
    }),
    subscribe: (fn: () => void) => {
      env.appChange = fn;
    },
  },
}));
vi.mock("../store/agent", () => ({
  useAgentStore: {
    getState: () => ({
      workspaceName: env.agentWorkspace,
      userEmail: env.agentEmail,
    }),
    subscribe: (fn: () => void) => {
      env.agentChange = fn;
    },
  },
}));
test.each(["route", "actor", "workspace", "agent scope"])(
  "invalidates %s ABA but not unchanged notifications",
  (change) => {
    let path = "/a",
      routeChanged = () => {};
    const router = {
      currentRoute: {
        value: {
          get fullPath() {
            return path;
          },
        },
      },
      afterEach: vi.fn((fn: () => void) => {
        routeChanged = fn;
        return () => {};
      }),
    } as unknown as AppRouterInstance;
    const opts = {
      chatId: "chat",
      runId: "run",
      runMode: "default" as const,
      isCurrent: () => true,
    };
    const old = captureAgentDomContext(router, opts, true)!;
    env.appChange();
    env.agentChange();
    routeChanged();
    expect(old.isCurrent()).toBe(true);
    if (change === "actor") {
      env.actor = "users/2";
      env.appChange();
      env.actor = "users/1";
      env.appChange();
    }
    if (change === "workspace") {
      env.workspace = "workspaces/q";
      env.appChange();
      env.workspace = "workspaces/p";
      env.appChange();
    }
    if (change === "agent scope") {
      env.agentEmail = "other@example.com";
      env.agentChange();
      env.agentEmail = env.email;
      env.agentChange();
    }
    if (change === "route") {
      path = "/b";
      routeChanged();
      path = "/a";
      routeChanged();
    }
    expect(old.isCurrent()).toBe(false);
    const next = captureAgentDomContext(router, opts)!;
    expect(next.pageGeneration).toBeGreaterThan(old.pageGeneration);
    expect(next.isCurrent()).toBe(true);
    expect(router.afterEach).toHaveBeenCalledOnce();
  }
);
