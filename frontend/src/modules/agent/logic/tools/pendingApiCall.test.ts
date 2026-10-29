import { beforeEach, describe, expect, test, vi } from "vitest";
import type { PreparedApiCall } from "./callApi";

const mocks = vi.hoisted(() => ({
  checkPreparedApiPermission: vi.fn(),
  dispatchPreparedApiCall: vi.fn(),
  validateIssueDecisionPermission: vi.fn(),
}));

vi.mock("./apiPermissionPreflight", () => ({
  checkPreparedApiPermission: mocks.checkPreparedApiPermission,
}));

vi.mock("./callApi", () => ({
  dispatchPreparedApiCall: mocks.dispatchPreparedApiCall,
}));
vi.mock("./issueDecisionPermission", () => ({
  validateIssueDecisionPermission: mocks.validateIssueDecisionPermission,
}));

import {
  createPendingApiCall,
  getPendingApiCallPreview,
  resolvePendingApiCall,
} from "./pendingApiCall";

const makeCall = (): PreparedApiCall => ({
  operationId: "bytebase.v1.ProjectService.UpdateProject",
  path: "/bytebase.v1.ProjectService/UpdateProject",
  body: JSON.stringify({ project: { name: "projects/demo", title: "Before" } }),
});

describe("pendingApiCall", () => {
  beforeEach(() => {
    vi.useRealTimers();
    mocks.checkPreparedApiPermission.mockReset();
    mocks.checkPreparedApiPermission.mockResolvedValue({ kind: "allowed" });
    mocks.dispatchPreparedApiCall.mockReset();
    mocks.dispatchPreparedApiCall.mockResolvedValue(
      JSON.stringify({ status: 200, response: {} })
    );
    mocks.validateIssueDecisionPermission.mockReset();
    mocks.validateIssueDecisionPermission.mockResolvedValue(true);
  });

  test("rechecks access after approval before sending the target request", async () => {
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-revoked",
      userName: "users/1",
      call: makeCall(),
    });
    mocks.checkPreparedApiPermission.mockResolvedValue({
      kind: "denied",
      result: JSON.stringify({
        status: 403,
        error: "Not called: bb.projects.update is not granted.",
      }),
    });
    const result = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-revoked",
      userName: "users/1",
      approved: true,
    });
    expect(JSON.parse(result)).toMatchObject({ status: 403 });
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not create a permission request without a signed-in principal", () => {
    expect(() =>
      createPendingApiCall({
        chatId: "chat-1",
        toolCallId: "tool-1",
        userName: "",
        call: makeCall(),
      })
    ).toThrow(/user/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not dispatch before approval and uses the prepared snapshot once", async () => {
    const call = makeCall();
    const expected = { ...call };
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call,
    });
    (call as { body: string }).body = JSON.stringify({ changed: true });

    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
    const result = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });

    expect(JSON.parse(result)).toEqual({ status: 200, response: {} });
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledExactlyOnceWith(
      expected
    );
    await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
  });

  test("previews only the matching, unexpired action without consuming it", async () => {
    vi.useFakeTimers();
    const call = makeCall();
    const { actionId, expiresAt } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call,
    });

    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-1",
        toolCallId: "tool-1",
        userName: "users/1",
      })
    ).toEqual({
      operationId: call.operationId,
      body: call.body,
      expiresAt,
    });
    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-2",
        toolCallId: "tool-1",
        userName: "users/1",
      })
    ).toBeUndefined();
    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-1",
        toolCallId: "tool-1",
        userName: "users/2",
      })
    ).toBeUndefined();
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();

    vi.setSystemTime(expiresAt);
    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-1",
        toolCallId: "tool-1",
        userName: "users/1",
      })
    ).toBeUndefined();
    vi.useRealTimers();
  });

  test("rejects a different chat, tool, or user without consuming the approval", async () => {
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });

    const wrongChat = await resolvePendingApiCall({
      actionId,
      chatId: "chat-2",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    const wrongTool = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-2",
      userName: "users/1",
      approved: true,
    });
    const wrongUser = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/2",
      approved: true,
    });
    expect(JSON.parse(wrongChat).error).toMatch(/not found|does not match/i);
    expect(JSON.parse(wrongTool).error).toMatch(/not found|does not match/i);
    expect(JSON.parse(wrongUser).error).toMatch(/not found|does not match/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();

    await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
  });

  test("denial consumes the action without dispatching", async () => {
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });

    const denied = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: false,
    });
    expect(JSON.parse(denied).error).toMatch(/denied/i);
    await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("shows Issue consequence and revalidates its state before approval", async () => {
    const call = {
      operationId: "bytebase.v1.IssueService.ApproveIssue",
      path: "/bytebase.v1.IssueService/ApproveIssue",
      body: JSON.stringify({ name: "projects/demo/issues/12" }),
    };
    const issueDecision = {
      issueName: "projects/demo/issues/12",
      issueTitle: "Change account table",
      impact: "final-approval-with-rollout" as const,
      action: "approve" as const,
      snapshot: "before",
    };
    const { actionId, expiresAt } = createPendingApiCall({
      chatId: "chat-issue",
      toolCallId: "tool-issue",
      userName: "users/1",
      call,
      issueDecision,
    });

    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-issue",
        toolCallId: "tool-issue",
        userName: "users/1",
      })
    ).toEqual({
      operationId: call.operationId,
      body: call.body,
      expiresAt,
      issueDecision: {
        issueName: issueDecision.issueName,
        issueTitle: issueDecision.issueTitle,
        impact: issueDecision.impact,
      },
    });

    mocks.validateIssueDecisionPermission.mockResolvedValueOnce(false);
    const stale = await resolvePendingApiCall({
      actionId,
      chatId: "chat-issue",
      toolCallId: "tool-issue",
      userName: "users/1",
      approved: true,
    });
    expect(JSON.parse(stale).error).toMatch(/changed|unavailable/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
    expect(mocks.validateIssueDecisionPermission).toHaveBeenCalledWith(
      issueDecision
    );
  });

  test("dispatches a current Issue decision once and does not revalidate denial", async () => {
    const call = {
      operationId: "bytebase.v1.IssueService.RejectIssue",
      path: "/bytebase.v1.IssueService/RejectIssue",
      body: JSON.stringify({ name: "projects/demo/issues/12" }),
    };
    const issueDecision = {
      issueName: "projects/demo/issues/12",
      issueTitle: "Change account table",
      impact: "reject" as const,
      action: "reject" as const,
      snapshot: "before",
    };
    const denied = createPendingApiCall({
      chatId: "chat-issue",
      toolCallId: "tool-deny",
      userName: "users/1",
      call,
      issueDecision,
    });
    await resolvePendingApiCall({
      actionId: denied.actionId,
      chatId: "chat-issue",
      toolCallId: "tool-deny",
      userName: "users/1",
      approved: false,
    });
    expect(mocks.validateIssueDecisionPermission).not.toHaveBeenCalled();

    const allowed = createPendingApiCall({
      chatId: "chat-issue",
      toolCallId: "tool-allow",
      userName: "users/1",
      call,
      issueDecision,
    });
    await resolvePendingApiCall({
      actionId: allowed.actionId,
      chatId: "chat-issue",
      toolCallId: "tool-allow",
      userName: "users/1",
      approved: true,
    });
    expect(mocks.validateIssueDecisionPermission).toHaveBeenCalledOnce();
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledExactlyOnceWith(call);
  });

  test("does not dispatch when the approved request was stopped before execution", async () => {
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });
    const controller = new AbortController();
    controller.abort();

    const result = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
      signal: controller.signal,
    });

    expect(JSON.parse(result).error).toMatch(/stopped/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
    expect(
      getPendingApiCallPreview({
        actionId,
        chatId: "chat-1",
        toolCallId: "tool-1",
        userName: "users/1",
      })
    ).toBeUndefined();
  });

  test("does not dispatch when stopped during Issue decision validation", async () => {
    let finishValidation!: (valid: boolean) => void;
    mocks.validateIssueDecisionPermission.mockImplementationOnce(
      () => new Promise<boolean>((resolve) => (finishValidation = resolve))
    );
    const { actionId } = createPendingApiCall({
      chatId: "chat-issue",
      toolCallId: "tool-issue",
      userName: "users/1",
      call: makeCall(),
      issueDecision: {
        issueName: "projects/demo/issues/12",
        issueTitle: "Example change",
        impact: "final-approval-with-rollout",
        action: "approve",
        snapshot: "before",
      },
    });
    const controller = new AbortController();
    const resolution = resolvePendingApiCall({
      actionId,
      chatId: "chat-issue",
      toolCallId: "tool-issue",
      userName: "users/1",
      approved: true,
      signal: controller.signal,
    });
    expect(mocks.validateIssueDecisionPermission).toHaveBeenCalledOnce();
    controller.abort();
    finishValidation(true);

    expect(JSON.parse(await resolution).error).toMatch(/stopped/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("a replacement request invalidates an older action for the same tool", async () => {
    const first = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });
    const second = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });

    const stale = await resolvePendingApiCall({
      actionId: first.actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(JSON.parse(stale).error).toMatch(/not found|used/i);
    await resolvePendingApiCall({
      actionId: second.actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
  });

  test("expires after five minutes without dispatching", async () => {
    vi.useFakeTimers();
    const { actionId, expiresAt } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });
    expect(expiresAt).toBe(Date.now() + 5 * 60_000);
    vi.setSystemTime(expiresAt);

    const result = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });

    expect(JSON.parse(result).error).toMatch(/expired/i);
    expect(mocks.dispatchPreparedApiCall).not.toHaveBeenCalled();
    vi.useRealTimers();
  });

  test("consumes approval before a pending dispatch settles", async () => {
    let release!: (value: string) => void;
    mocks.dispatchPreparedApiCall.mockImplementationOnce(
      () => new Promise<string>((resolve) => (release = resolve))
    );
    const { actionId } = createPendingApiCall({
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      call: makeCall(),
    });

    const first = resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    const replay = await resolvePendingApiCall({
      actionId,
      chatId: "chat-1",
      toolCallId: "tool-1",
      userName: "users/1",
      approved: true,
    });
    expect(JSON.parse(replay).error).toMatch(/not found|used/i);
    expect(mocks.dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
    release(JSON.stringify({ status: 200, response: {} }));
    await first;
  });
});
