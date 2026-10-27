import { beforeEach, describe, expect, test, vi } from "vitest";
import type { PreparedApiCall } from "./callApi";

const mocks = vi.hoisted(() => ({
  dispatchPreparedApiCall: vi.fn(),
}));

vi.mock("./callApi", () => ({
  dispatchPreparedApiCall: mocks.dispatchPreparedApiCall,
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
    mocks.dispatchPreparedApiCall.mockReset();
    mocks.dispatchPreparedApiCall.mockResolvedValue(
      JSON.stringify({ status: 200, response: {} })
    );
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
    ).toEqual({ operationId: call.operationId, body: call.body });
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
