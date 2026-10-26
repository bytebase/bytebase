import { dispatchPreparedApiCall, type PreparedApiCall } from "./callApi";

const PERMISSION_TTL_MS = 5 * 60_000;

interface PendingApiCall {
  chatId: string;
  toolCallId: string;
  userName: string;
  call: PreparedApiCall;
  expiresAt: number;
}

const pendingApiCalls = new Map<string, PendingApiCall>();

export function createPendingApiCall({
  chatId,
  toolCallId,
  userName,
  call,
}: {
  chatId: string;
  toolCallId: string;
  userName: string;
  call: PreparedApiCall;
}): { actionId: string; expiresAt: number } {
  if (!userName) {
    throw new Error("A signed-in user is required to request permission.");
  }
  const now = Date.now();
  for (const [id, pending] of pendingApiCalls) {
    if (
      pending.expiresAt <= now ||
      (pending.chatId === chatId && pending.toolCallId === toolCallId)
    ) {
      pendingApiCalls.delete(id);
    }
  }

  let actionId: string;
  do {
    actionId = Array.from(crypto.getRandomValues(new Uint8Array(16)), (byte) =>
      byte.toString(16).padStart(2, "0")
    ).join("");
  } while (pendingApiCalls.has(actionId));
  const expiresAt = now + PERMISSION_TTL_MS;
  pendingApiCalls.set(actionId, {
    chatId,
    toolCallId,
    userName,
    call: Object.freeze({ ...call }),
    expiresAt,
  });
  return { actionId, expiresAt };
}

export function getPendingApiCallPreview({
  actionId,
  chatId,
  toolCallId,
  userName,
}: {
  actionId: string;
  chatId: string;
  toolCallId: string;
  userName: string;
}): { operationId: string; body: string } | undefined {
  const pending = pendingApiCalls.get(actionId);
  if (
    !pending ||
    pending.expiresAt <= Date.now() ||
    pending.chatId !== chatId ||
    pending.toolCallId !== toolCallId ||
    pending.userName !== userName
  ) {
    return undefined;
  }
  return {
    operationId: pending.call.operationId,
    body: pending.call.body,
  };
}

export async function resolvePendingApiCall({
  actionId,
  chatId,
  toolCallId,
  userName,
  approved,
}: {
  actionId: string;
  chatId: string;
  toolCallId: string;
  userName: string;
  approved: boolean;
}): Promise<string> {
  const pending = pendingApiCalls.get(actionId);
  if (!pending) {
    return JSON.stringify({
      error: "Permission request not found or already used.",
    });
  }
  if (pending.expiresAt <= Date.now()) {
    pendingApiCalls.delete(actionId);
    return JSON.stringify({ error: "Permission request expired." });
  }
  if (
    pending.chatId !== chatId ||
    pending.toolCallId !== toolCallId ||
    pending.userName !== userName
  ) {
    return JSON.stringify({
      error: "Permission request does not match this user, chat, or action.",
    });
  }

  pendingApiCalls.delete(actionId);
  if (!approved) {
    return JSON.stringify({ error: "Permission request denied." });
  }
  return dispatchPreparedApiCall(pending.call);
}
