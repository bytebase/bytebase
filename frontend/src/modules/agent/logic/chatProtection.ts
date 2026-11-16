const scopePrefix = (scopeKey: string): string =>
  `bb-agent-chat-protection:${JSON.stringify(scopeKey)}:`;

const protectionKey = (scopeKey: string, chatId: string): string =>
  `${scopePrefix(scopeKey)}${JSON.stringify(chatId)}`;

export const supportsAgentChatProtection = (): boolean =>
  typeof globalThis.navigator?.locks?.request === "function" &&
  typeof globalThis.navigator?.locks?.query === "function";

interface ChatClaim {
  controller: AbortController;
  ready: boolean;
  released: Promise<void>;
  release: () => void;
}

export const createAgentChatProtection = (onChange: () => void) => {
  const claims = new Map<string, ChatClaim>();
  let currentScope: string | null = null;
  let disposed = false;

  const isCurrent = (scopeKey: string, chatId: string, claim: ChatClaim) =>
    !disposed && currentScope === scopeKey && claims.get(chatId) === claim;

  const acquire = async (
    scopeKey: string,
    chatId: string,
    claim: ChatClaim
  ) => {
    if (!isCurrent(scopeKey, chatId, claim)) return;
    try {
      await navigator.locks.request(
        protectionKey(scopeKey, chatId),
        { mode: "shared", signal: claim.controller.signal },
        async (lock) => {
          if (!isCurrent(scopeKey, chatId, claim)) return;
          if (!lock) throw new Error("Agent chat protection is unavailable");
          claim.ready = true;
          onChange();
          await claim.released;
        }
      );
    } catch {
      if (isCurrent(scopeKey, chatId, claim)) {
        claim.ready = false;
        onChange();
      }
    }
  };

  const remove = (chatId: string, claim: ChatClaim) => {
    claims.delete(chatId);
    claim.controller.abort();
    claim.release();
  };

  return {
    update(scopeKey: string | null, chatIds: readonly string[]): void {
      if (disposed) return;
      const desired = new Set(scopeKey === null ? [] : chatIds);
      let removed = false;
      for (const [chatId, claim] of claims) {
        if (currentScope !== scopeKey || !desired.has(chatId)) {
          remove(chatId, claim);
          removed = true;
        }
      }
      currentScope = scopeKey;
      const added: [string, ChatClaim][] = [];
      if (scopeKey !== null && supportsAgentChatProtection()) {
        for (const chatId of desired) {
          if (claims.has(chatId)) continue;
          let release!: () => void;
          const released = new Promise<void>((resolve) => {
            release = resolve;
          });
          const claim: ChatClaim = {
            controller: new AbortController(),
            ready: false,
            released,
            release,
          };
          claims.set(chatId, claim);
          added.push([chatId, claim]);
        }
        for (const [chatId, claim] of added)
          void acquire(scopeKey, chatId, claim);
      }
      if (removed) onChange();
    },
    isReady(scopeKey: string, chatId: string): boolean {
      if (!supportsAgentChatProtection()) return true;
      return (
        !disposed &&
        currentScope === scopeKey &&
        claims.get(chatId)?.ready === true
      );
    },
    dispose(): void {
      if (disposed) return;
      disposed = true;
      currentScope = null;
      const removed = claims.size > 0;
      for (const [chatId, claim] of claims) remove(chatId, claim);
      if (removed) onChange();
    },
  };
};

export const getProtectedAgentChatIds = async (
  scopeKey: string
): Promise<Set<string>> => {
  if (!supportsAgentChatProtection()) {
    throw new Error("Agent chat protection requires Web Locks");
  }
  const snapshot = await navigator.locks.query();
  const prefix = scopePrefix(scopeKey);
  const protectedIds = new Set<string>();
  for (const { name } of [
    ...(snapshot.held ?? []),
    ...(snapshot.pending ?? []),
  ]) {
    if (!name?.startsWith(prefix)) continue;
    const chatId: unknown = JSON.parse(name.slice(prefix.length));
    if (typeof chatId !== "string") {
      throw new Error("Invalid Agent chat protection claim");
    }
    protectedIds.add(chatId);
  }
  return protectedIds;
};

// Acquire before execution, scope persistence, and origin storage accounting.
export const withAgentChatEvictionLease = async <T>(
  scopeKey: string,
  chatId: string,
  action: () => Promise<T>
): Promise<{ acquired: boolean; value?: T }> => {
  if (!supportsAgentChatProtection()) return { acquired: false };
  let actionStarted = false;
  try {
    return await navigator.locks.request(
      protectionKey(scopeKey, chatId),
      { mode: "exclusive", ifAvailable: true },
      async (lock) => {
        if (!lock) return { acquired: false };
        actionStarted = true;
        return { acquired: true, value: await action() };
      }
    );
  } catch (error) {
    if (actionStarted) throw error;
    return { acquired: false };
  }
};
