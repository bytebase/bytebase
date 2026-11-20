import { beforeEach, expect, test, vi } from "vitest";
import { UNKNOWN_USER_NAME } from "@/types/v1/user";
import {
  captureAgentExecutionContext,
  isAgentExecutionContextCurrent,
} from "./executionContext";

const account = vi.hoisted(() => ({
  name: "users/1",
  currentUserName: "users/1",
  workspace: "workspaces/example",
  userWorkspace: "workspaces/example",
}));
vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      currentUser: { name: account.name, workspace: account.userWorkspace },
      currentUserName: account.currentUserName,
      workspaceResourceName: () => account.workspace,
    }),
  },
}));

beforeEach(() => {
  account.name = account.currentUserName = "users/1";
  account.workspace = account.userWorkspace = "workspaces/example";
});

test.each([
  { name: "" },
  { name: UNKNOWN_USER_NAME, currentUserName: UNKNOWN_USER_NAME },
  { currentUserName: "users/other" },
  { workspace: "", userWorkspace: "" },
  { workspace: "workspaces/-", userWorkspace: "workspaces/-" },
  { userWorkspace: "workspaces/other" },
])("does not capture an unverifiable principal: %j", (state) => {
  Object.assign(account, state);
  expect(captureAgentExecutionContext()).toBeUndefined();
  expect(isAgentExecutionContextCurrent(undefined)).toBe(false);
});

test("matches the captured actor and workspace only while execution is active", () => {
  const context = captureAgentExecutionContext();
  expect(context).toEqual({
    actor: account.name,
    workspace: account.workspace,
  });
  expect(isAgentExecutionContextCurrent(context)).toBe(true);
  expect(isAgentExecutionContextCurrent(context, AbortSignal.abort())).toBe(
    false
  );
  account.workspace = account.userWorkspace = "workspaces/other";
  expect(isAgentExecutionContextCurrent(context)).toBe(false);
  account.workspace = account.userWorkspace = "workspaces/example";
  account.name = account.currentUserName = "users/other";
  expect(isAgentExecutionContextCurrent(context)).toBe(false);
});
